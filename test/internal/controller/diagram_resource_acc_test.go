package controller_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	application "github.com/xaligo/terraform-provider/internal"
)

const acceptanceResourceName = "xaligo_diagram.test"

func TestAccDiagramLifecycle(t *testing.T) {
	requireAcceptance(t)

	directory := t.TempDir()
	sourceDirectory := filepath.Join(directory, "source")
	if err := os.Mkdir(sourceDirectory, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	outputPath := filepath.Join(directory, "architecture.xal")
	writeAcceptanceSource(t, sourceDirectory, `resource "aws_s3_bucket" "logs" {}`)

	disableConfig := acceptanceConfig(sourceDirectory, outputPath, "disable", false)
	enableConfig := acceptanceConfig(sourceDirectory, outputPath, "enable", false)
	enableOverwriteConfig := acceptanceConfig(sourceDirectory, outputPath, "enable", true)
	var firstDigest string
	var updatedDigest string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			return checkPathAbsent(outputPath)
		},
		Steps: []resource.TestStep{
			{
				Config: disableConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(acceptanceResourceName, "effective_export", "disable"),
					checkOutputAbsent(outputPath),
				),
			},
			{
				Config:             enableConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if err := checkPathAbsent(outputPath); err != nil {
						t.Fatalf("enable plan mutated output: %v", err)
					}
				},
				Config: enableConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(acceptanceResourceName, "effective_export", "enable"),
					resource.TestCheckResourceAttrSet(acceptanceResourceName, "source_sha256"),
					resource.TestCheckResourceAttrSet(acceptanceResourceName, "content_sha256"),
					checkManagedDigest(outputPath, &firstDigest, false),
					checkValidXAL(outputPath),
					checkXaligoCLI(outputPath),
				),
			},
			{
				Config: enableConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkManagedDigest(outputPath, &firstDigest, false),
					checkValidXAL(outputPath),
				),
			},
			{
				PreConfig: func() {
					writeAcceptanceSource(t, sourceDirectory, `resource "aws_sqs_queue" "events" {}`)
				},
				Config: enableConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkManagedDigest(outputPath, &firstDigest, true),
					checkManagedDigest(outputPath, &updatedDigest, false),
					checkValidXAL(outputPath),
				),
			},
			{
				PreConfig: func() {
					if err := os.Remove(outputPath); err != nil {
						t.Fatalf("remove managed output to simulate drift: %v", err)
					}
				},
				Config:             enableConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				PreConfig: func() {
					if err := checkPathAbsent(outputPath); err != nil {
						t.Fatalf("drift plan rewrote missing output: %v", err)
					}
				},
				Config: enableConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					checkManagedDigest(outputPath, &updatedDigest, false),
					checkValidXAL(outputPath),
				),
			},
			{
				Config: disableConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(acceptanceResourceName, "effective_export", "disable"),
					checkManagedDigest(outputPath, &updatedDigest, false),
				),
			},
			{
				PreConfig: func() {
					if err := os.WriteFile(outputPath, []byte("external while disabled\n"), 0o644); err != nil {
						t.Fatalf("modify disabled output: %v", err)
					}
				},
				Config: disableConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(acceptanceResourceName, "effective_export", "disable"),
					checkOutputContent(outputPath, "external while disabled\n"),
				),
			},
			{
				Config:      enableConfig,
				ExpectError: regexp.MustCompile("Output was modified externally"),
			},
			{
				Config: enableOverwriteConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(acceptanceResourceName, "effective_export", "enable"),
					checkValidXAL(outputPath),
				),
			},
		},
	})
}

func TestAccDiagramDestroyPreservesExternalEdit(t *testing.T) {
	requireAcceptance(t)

	directory := t.TempDir()
	sourceDirectory := filepath.Join(directory, "source")
	if err := os.Mkdir(sourceDirectory, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	outputPath := filepath.Join(directory, "architecture.xal")
	writeAcceptanceSource(t, sourceDirectory, `resource "aws_s3_bucket" "logs" {}`)
	providerConfig := acceptanceConfig(sourceDirectory, outputPath, "enable", false)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			content, err := os.ReadFile(outputPath)
			if err != nil {
				return fmt.Errorf("read preserved output: %w", err)
			}
			if string(content) != "external before destroy\n" {
				return fmt.Errorf("preserved output = %q", content)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: providerConfig,
				Check:  checkValidXAL(outputPath),
			},
			{
				ResourceName: acceptanceResourceName,
				Config:       acceptanceConfig(sourceDirectory, outputPath, "enable", true),
				PreConfig: func() {
					if err := os.WriteFile(outputPath, []byte("external before destroy\n"), 0o644); err != nil {
						t.Fatalf("modify output before destroy: %v", err)
					}
				},
				Destroy: true,
			},
		},
	})
}

func TestAccDiagramUnownedOutputRequiresOverwrite(t *testing.T) {
	requireAcceptance(t)

	directory := t.TempDir()
	sourceDirectory := filepath.Join(directory, "source")
	if err := os.Mkdir(sourceDirectory, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	outputPath := filepath.Join(directory, "architecture.xal")
	writeAcceptanceSource(t, sourceDirectory, `resource "aws_s3_bucket" "logs" {}`)
	if err := os.WriteFile(outputPath, []byte("unowned\n"), 0o644); err != nil {
		t.Fatalf("write unowned output: %v", err)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: acceptanceProviderFactories(),
		CheckDestroy: func(_ *terraform.State) error {
			return checkPathAbsent(outputPath)
		},
		Steps: []resource.TestStep{
			{
				Config:      acceptanceConfig(sourceDirectory, outputPath, "enable", false),
				ExpectError: regexp.MustCompile("Output already exists"),
			},
			{
				Config: acceptanceConfig(sourceDirectory, outputPath, "enable", true),
				Check:  checkValidXAL(outputPath),
			},
		},
	})
}

func requireAcceptance(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("set TF_ACC=1 to run Terraform acceptance tests")
	}
}

func acceptanceProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"xaligo": providerserver.NewProtocol6WithError(application.Provider("test")()),
	}
}

func acceptanceConfig(sourceDirectory, outputPath, export string, overwrite bool) string {
	return fmt.Sprintf(`
provider "xaligo" {
  export = %q
}

resource "xaligo_diagram" "test" {
  source_dir  = %q
  output_path = %q
  title       = "Acceptance"
  overwrite   = %t
}
`, export, sourceDirectory, outputPath, overwrite)
}

func writeAcceptanceSource(t *testing.T, directory, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, "main.tf"), []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("write acceptance Terraform source: %v", err)
	}
}

func checkOutputAbsent(path string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		return checkPathAbsent(path)
	}
}

func checkPathAbsent(path string) error {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect expected-absent output: %w", err)
	}
	return fmt.Errorf("output unexpectedly exists: %s", path)
}

func checkOutputContent(path, expected string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read output: %w", err)
		}
		if string(content) != expected {
			return fmt.Errorf("output = %q, want %q", content, expected)
		}
		return nil
	}
}

func checkManagedDigest(path string, retained *string, requireChange bool) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrWith(acceptanceResourceName, "content_sha256", func(value string) error {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read managed output: %w", err)
			}
			digest := sha256.Sum256(content)
			actual := hex.EncodeToString(digest[:])
			if value != actual {
				return fmt.Errorf("state content_sha256 = %q, output digest = %q", value, actual)
			}
			if requireChange && *retained == actual {
				return fmt.Errorf("output digest did not change: %s", actual)
			}
			if !requireChange && *retained != "" && *retained != actual {
				return fmt.Errorf("output digest changed: got %s, want %s", actual, *retained)
			}
			*retained = actual
			return nil
		}),
		resource.TestCheckResourceAttrWith(acceptanceResourceName, "observed_content_sha256", func(value string) error {
			if *retained != "" && value != *retained {
				return fmt.Errorf("state observed_content_sha256 = %q, want %q", value, *retained)
			}
			return nil
		}),
	)
}

func checkValidXAL(path string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read generated XAL: %w", err)
		}
		var root struct {
			XMLName xml.Name
			Version string `xml:"version,attr"`
		}
		if err := xml.Unmarshal(content, &root); err != nil {
			return fmt.Errorf("parse generated XAL: %w", err)
		}
		if root.XMLName.Local != "xaligo" || root.Version != "1" {
			return fmt.Errorf("generated XAL root = <%s version=%q>", root.XMLName.Local, root.Version)
		}
		return nil
	}
}

func checkXaligoCLI(path string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		binary := strings.TrimSpace(os.Getenv("XALIGO_BIN"))
		if binary == "" {
			resolved, err := exec.LookPath("xaligo")
			if err != nil {
				return fmt.Errorf("xaligo CLI is required for acceptance validation; set XALIGO_BIN: %w", err)
			}
			binary = resolved
		}
		command := exec.Command(binary, "validate", path)
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("xaligo validate failed: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	}
}
