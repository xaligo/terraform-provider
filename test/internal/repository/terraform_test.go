package repository_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	"github.com/xaligo/terraform-provider/internal/repository"
)

func TestLoadParsesLexicalFilesAndReferences(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "a.tf"), `
terraform {}

provider "xaligo" {
  export = "enable"
}

resource "xaligo_diagram" "self" {
  source_dir  = "."
  output_path = "ignored.xal"
}

data "xaligo_catalog" "self" {}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  tags = {
    Name = "main-vpc"
  }
}

module "network" {
  source = "./network"
  vpc_id = aws_vpc.main.id
}
`)
	writeTestFile(t, filepath.Join(directory, "z.tf"), `
resource "aws_subnet" "public" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "ap-northeast-1a"
}
`)
	writeTestFile(t, filepath.Join(directory, "ignored.txt"), `resource "ignored" "text" {}`)
	if err := os.Mkdir(filepath.Join(directory, "nested"), 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	writeTestFile(t, filepath.Join(directory, "nested", "ignored.tf"), `resource "ignored" "nested" {}`)

	config, diagnostics, err := repository.NewTerraformRepository().Load(directory)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("Load() diagnostics = %#v", diagnostics)
	}

	gotFiles := make([]string, 0, len(config.Files))
	for _, file := range config.Files {
		gotFiles = append(gotFiles, file.Path)
	}
	if want := []string{"a.tf", "z.tf"}; !reflect.DeepEqual(gotFiles, want) {
		t.Fatalf("file order = %v, want %v", gotFiles, want)
	}

	gotAddresses := make([]string, 0, len(config.Blocks))
	byAddress := make(map[string]commonentity.Block, len(config.Blocks))
	for _, block := range config.Blocks {
		gotAddresses = append(gotAddresses, block.Address)
		byAddress[block.Address] = block
	}
	wantAddresses := []string{"aws_subnet.public", "aws_vpc.main", "module.network"}
	if !reflect.DeepEqual(gotAddresses, wantAddresses) {
		t.Fatalf("block addresses = %v, want %v", gotAddresses, wantAddresses)
	}

	subnet := byAddress["aws_subnet.public"]
	if got, want := subnet.Attributes["vpc_id"].References, []string{"aws_vpc.main"}; !reflect.DeepEqual(got, want) {
		t.Errorf("subnet vpc references = %v, want %v", got, want)
	}
	if subnet.Range.Filename != "z.tf" || subnet.Range.StartLine != 2 {
		t.Errorf("subnet range = %#v", subnet.Range)
	}
	vpc := byAddress["aws_vpc.main"]
	if got, ok := vpc.Attributes["tags"].Value.ObjectString("Name"); !ok || got != "main-vpc" {
		t.Errorf("VPC Name = %q, %v", got, ok)
	}
	if got, want := byAddress["module.network"].References, []string{"aws_vpc.main"}; !reflect.DeepEqual(got, want) {
		t.Errorf("module references = %v, want %v", got, want)
	}
}

func TestLoadReportsMalformedHCLWithSourceRange(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "broken.tf"), "resource \"aws_vpc\" \"broken\" {\n  cidr_block =\n}\n")

	_, diagnostics, err := repository.NewTerraformRepository().Load(directory)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !commonentity.HasErrors(diagnostics) {
		t.Fatalf("Load() diagnostics = %#v, want an error", diagnostics)
	}
	foundRange := false
	for _, value := range diagnostics {
		if value.Code == "TFCONFIG-E003" && value.Range.Filename == "broken.tf" && value.Range.StartLine > 0 {
			foundRange = true
		}
	}
	if !foundRange {
		t.Errorf("diagnostics do not include a malformed-HCL source range: %#v", diagnostics)
	}
}

func TestLoadWarnsForUnevaluatedExpansionAndDynamicBlocks(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "main.tf"), `
resource "aws_security_group" "other" {}

resource "aws_security_group" "main" {
  count = var.instances

  dynamic "ingress" {
    for_each = var.rules
    content {
      security_groups = [aws_security_group.other.id]
    }
  }
}
`)

	config, diagnostics, err := repository.NewTerraformRepository().Load(directory)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := diagnosticCodes(diagnostics), []string{"TFCONFIG-W001", "TFCONFIG-W002"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostic codes = %v, want %v", got, want)
	}
	var main commonentity.Block
	for _, block := range config.Blocks {
		if block.Address == "aws_security_group.main" {
			main = block
		}
	}
	if got, want := main.References, []string{"aws_security_group.other"}; !reflect.DeepEqual(got, want) {
		t.Errorf("dynamic references = %v, want %v", got, want)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func diagnosticCodes(values []commonentity.Diagnostic) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Code)
	}
	return result
}
