package usecase_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

func TestComplexGoldenRendersWithXaligo(t *testing.T) {
	t.Parallel()

	binary := strings.TrimSpace(os.Getenv("XALIGO_BIN"))
	if binary == "" {
		t.Skip("set XALIGO_BIN to run xaligo renderer compatibility test")
	}
	inputPath := filepath.Join(repositoryRoot(t), "test", "internal", "usecase", "testdata", "complex", "expected.xal")
	outputPath := filepath.Join(t.TempDir(), "complex.svg")
	command := exec.Command(binary, "render", inputPath, "--output", outputPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("xaligo render error = %v\n%s", err, output)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat rendered SVG: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("xaligo render produced an empty SVG")
	}
}

func TestConvertComplexTerraformMatchesGoldenAndRetainsDiagnostics(t *testing.T) {
	t.Parallel()

	sourceDirectory := filepath.Join(repositoryRoot(t), "test", "internal", "usecase", "testdata", "complex")
	layoutSource, err := os.ReadFile(filepath.Join(sourceDirectory, "90_xaligo.tf"))
	if err != nil {
		t.Fatalf("read complex xaligo definition: %v", err)
	}
	for _, required := range []string{
		`data "xaligo_items" "complex"`, `items = merge(data.xaligo_items.complex.items, {`,
		`application_items = [for address in var.diagram_layout.application : local.items[address]]`,
		`resource "xaligo_diagram" "complex"`, `paper_size  = "A3"`, `container {`, `row {`, `col {`, `layout {`,
		`items    = local.application_items`, `items = [local.items.application_services]`, `item     = local.items.aws_cloud`,
	} {
		if !strings.Contains(string(layoutSource), required) {
			t.Fatalf("90_xaligo.tf does not contain %q", required)
		}
	}
	request := commonentity.ConversionRequest{
		SourceDir: sourceDirectory, FrameID: "complex", Title: "Complex AWS Architecture",
		PaperSize: "A3", Orientation: "landscape", GridGap: 20,
		Containers: []commonentity.ContainerSpec{{
			ID:    "application-services",
			Items: []string{"items.aws_lambda_function.worker", "items.aws_s3_bucket.artifacts"},
			Style: commonentity.LayoutStyle{Layout: "horizontal", Gap: 0, GapSet: true, Align: "middle-spread", Overflow: "visible"},
		}},
		Rows: []commonentity.RowSpec{{Gap: 20, GapSet: true, Overflow: "visible", Columns: []commonentity.ColumnSpec{
			{Items: []string{"items.aws_cloudwatch_log_group.worker"}, Span: 3, Style: commonentity.LayoutStyle{Class: "pa-1"}},
			{Items: []string{"items.application-services"}, Span: 6},
			{Items: []string{"items.aws_sqs_queue.jobs"}, Span: 3, Style: commonentity.LayoutStyle{Align: "middle-center"}},
		}}},
		Layouts: []commonentity.ItemLayoutSpec{{Item: "items.xaligo-aws-cloud", Style: commonentity.LayoutStyle{Align: "top-left", Overflow: "visible", Row: 2}}},
	}
	converter := newTestDiagramUsecase()
	first, err := converter.Convert(context.Background(), request)
	if err != nil {
		t.Fatalf("first Convert() error = %v; diagnostics = %#v", err, first.Diagnostics)
	}
	second, err := converter.Convert(context.Background(), request)
	if err != nil {
		t.Fatalf("second Convert() error = %v; diagnostics = %#v", err, second.Diagnostics)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("complex conversion is not deterministic\nfirst: %#v\nsecond: %#v", first, second)
	}

	expected, err := os.ReadFile(filepath.Join(sourceDirectory, "expected.xal"))
	if err != nil {
		t.Fatalf("read complex golden XAL: %v", err)
	}
	if !reflect.DeepEqual(first.Content, expected) {
		t.Fatalf("complex generated XAL differs from golden\n--- generated ---\n%s\n--- expected ---\n%s", first.Content, expected)
	}
	if first.SourceSHA256 == "" || first.ContentSHA256 == "" {
		t.Fatal("complex conversion returned empty digests")
	}
	if commonentity.HasErrors(first.Diagnostics) {
		t.Fatalf("complex conversion diagnostics contain errors: %#v", first.Diagnostics)
	}

	codeCounts := make(map[string]int)
	for _, diagnostic := range first.Diagnostics {
		codeCounts[diagnostic.Code]++
		if diagnostic.Severity != commonentity.SeverityWarning {
			t.Errorf("diagnostic %s severity = %s, want warning", diagnostic.Code, diagnostic.Severity)
		}
		if diagnostic.Range.Filename == "" || diagnostic.Range.StartLine == 0 {
			t.Errorf("diagnostic %s has no source position: %#v", diagnostic.Code, diagnostic.Range)
		}
	}
	wantCodeCounts := map[string]int{
		"MAPPING-W001":  2,
		"MAPPING-W002":  2,
		"MAPPING-W003":  25,
		"TFCONFIG-W001": 2,
		"TFCONFIG-W002": 1,
	}
	if !reflect.DeepEqual(codeCounts, wantCodeCounts) {
		t.Fatalf("complex diagnostic counts = %v, want %v", codeCounts, wantCodeCounts)
	}

	content := string(first.Content)
	for _, required := range []string{
		`<aws-cloud id="xaligo-aws-cloud"`,
		`<container id="application-services"`,
		`<row gap="20" overflow="visible">`,
		`<col row="224" span="6">`,
		`<region id="xaligo-aws-region-ap-northeast-1"`,
		`<vpc id="aws_vpc-main"`,
		`<security-group id="aws_security_group-application"`,
		`<availability-zone id="aws_vpc-main-availability-zone-ap-northeast-1a"`,
		`<public-subnet id="aws_subnet-public_a"`,
		`<item id="1592" name="aws_lb-application">`,
		`<item id="1790" name="aws_instance-web">`,
		`<rectangle id="data-aws_ami-linux"`,
		`<rectangle id="module-observability"`,
		`<rectangle id="terraform_data-release"`,
	} {
		if !strings.Contains(content, required) {
			t.Errorf("complex generated XAL does not contain %q", required)
		}
	}
	for _, excluded := range []string{"database_password", "queue_depth"} {
		if strings.Contains(content, excluded) {
			t.Errorf("complex generated XAL unexpectedly contains %q", excluded)
		}
	}
}

func TestComplexTerraformItemsDataSourceIncludesEveryInfrastructureAddress(t *testing.T) {
	t.Parallel()

	sourceDirectory := filepath.Join(repositoryRoot(t), "test", "internal", "usecase", "testdata", "complex")
	items, diagnostics, err := newTestDiagramUsecase().Items(context.Background(), sourceDirectory)
	if err != nil || commonentity.HasErrors(diagnostics) {
		t.Fatalf("Items() error = %v; diagnostics = %#v", err, diagnostics)
	}
	if got, want := len(items), 28; got != want {
		t.Fatalf("items count = %d, want %d: %#v", got, want, items)
	}
	for _, address := range []string{
		"aws_vpc.main", "aws_subnet.public_a", "aws_lambda_function.worker",
		"data.aws_ami.linux", "module.observability", "terraform_data.release",
	} {
		if items[address] != address {
			t.Errorf("items[%q] = %q, want address", address, items[address])
		}
	}
	if _, exists := items["xaligo_diagram.complex"]; exists {
		t.Fatal("items map must not include its owning xaligo_diagram resource")
	}
}
