package internal_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	application "github.com/xaligo/terraform-provider/internal"
)

func TestCLIRouterRegistersConceptCommands(t *testing.T) {
	t.Parallel()

	command := application.InitCLIRouter("test", &bytes.Buffer{}, &bytes.Buffer{})
	names := make([]string, 0, len(command.Commands()))
	for _, child := range command.Commands() {
		names = append(names, child.Name())
	}
	sort.Strings(names)
	if got, want := strings.Join(names, ","), "convert,serve,version"; got != want {
		t.Fatalf("CLI commands = %q, want %q", got, want)
	}
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := application.CLIMain(context.Background(), "1.2.3", []string{"version"}, &stdout, &stderr); err != nil {
		t.Fatalf("version command error = %v", err)
	}
	if stdout.String() != "1.2.3\n" || stderr.Len() != 0 {
		t.Fatalf("version output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCombinedMainRoutesCLIInvocation(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := application.Main(context.Background(), "2.0.0", []string{"version"}, &stdout, &stderr); err != nil {
		t.Fatalf("combined Main() error = %v", err)
	}
	if stdout.String() != "2.0.0\n" || stderr.Len() != 0 {
		t.Fatalf("combined output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestConvertCommandWritesAndProtectsOutput(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "main.tf"), []byte(`resource "aws_s3_bucket" "logs" {}`), 0o644); err != nil {
		t.Fatalf("write Terraform source: %v", err)
	}
	output := filepath.Join(directory, "diagram.xal")

	execute := func(extra ...string) (string, string, error) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command := application.InitCLIRouter("test", &stdout, &stderr)
		arguments := []string{"convert", source, "--output", output}
		arguments = append(arguments, extra...)
		command.SetArgs(arguments)
		err := command.ExecuteContext(context.Background())
		return stdout.String(), stderr.String(), err
	}

	stdout, stderr, err := execute()
	if err != nil {
		t.Fatalf("first convert command error = %v, stderr=%q", err, stderr)
	}
	if !strings.HasPrefix(stdout, "generated ") || !strings.HasSuffix(stdout, "diagram.xal\n") || stderr != "" {
		t.Fatalf("first convert output stdout=%q stderr=%q", stdout, stderr)
	}
	content, err := os.ReadFile(output)
	if err != nil || !strings.Contains(string(content), `<item id="1642" name="aws_s3_bucket-logs"></item>`) {
		t.Fatalf("generated output = %q, %v", content, err)
	}

	if _, _, err := execute(); err == nil || !strings.Contains(err.Error(), "unowned output") {
		t.Fatalf("second convert command error = %v", err)
	}
	if _, _, err := execute("--overwrite"); err != nil {
		t.Fatalf("overwrite convert command error = %v", err)
	}
}

func TestComplexConvertCommandProducesDeterministicDefaultLayout(t *testing.T) {
	t.Parallel()

	sourceDirectory := filepath.Join(repositoryRoot(t), "test", "internal", "usecase", "testdata", "complex")
	outputPath := filepath.Join(t.TempDir(), "complex.xal")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := application.CLIMain(context.Background(), "test", []string{
		"convert",
		sourceDirectory,
		"--output", outputPath,
		"--frame-id", "complex",
		"--title", "Complex AWS Architecture",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("complex CLI conversion error = %v; stderr = %q", err, stderr.String())
	}
	resolvedDirectory, err := filepath.EvalSymlinks(filepath.Dir(outputPath))
	if err != nil {
		t.Fatalf("resolve complex output directory: %v", err)
	}
	wantStdout := "generated " + filepath.Join(resolvedDirectory, filepath.Base(outputPath)) + "\n"
	if stdout.String() != wantStdout {
		t.Fatalf("complex CLI stdout = %q", stdout.String())
	}
	if count := strings.Count(stderr.String(), "warning: "); count != 32 {
		t.Fatalf("complex CLI warning count = %d, want 32\nstderr:\n%s", count, stderr.String())
	}

	generated, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read complex CLI output: %v", err)
	}
	secondPath := filepath.Join(t.TempDir(), "complex.xal")
	err = application.CLIMain(context.Background(), "test", []string{
		"convert", sourceDirectory, "--output", secondPath,
		"--frame-id", "complex", "--title", "Complex AWS Architecture",
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("second complex CLI conversion error = %v", err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read second complex CLI output: %v", err)
	}
	if !bytes.Equal(generated, second) {
		t.Fatalf("complex CLI output is not deterministic\n--- first ---\n%s\n--- second ---\n%s", generated, second)
	}
	for _, expected := range []string{`<aws-cloud id="xaligo-aws-cloud"`, `<region id="xaligo-aws-region-ap-northeast-1"`, `<vpc id="aws_vpc-main"`} {
		if !strings.Contains(string(generated), expected) {
			t.Fatalf("complex CLI output does not contain %q", expected)
		}
	}
}
