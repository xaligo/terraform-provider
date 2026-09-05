package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

func TestDiagramServicePlanApplyReadAndDelete(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	sourceDir := filepath.Join(directory, "source")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	writeSource(t, sourceDir, `resource "aws_s3_bucket" "logs" {}`)
	outputPath := filepath.Join(directory, "diagram.xal")
	spec := commonentity.DiagramSpec{SourceDir: sourceDir, OutputPath: outputPath, FrameID: "main"}
	service := newTestDiagramUsecase()

	disabled, diagnostics, err := service.Plan(context.Background(), commonentity.PlanInput{Spec: spec, Export: commonentity.ExportDisable})
	if err != nil || len(diagnostics) != 0 || disabled.EffectiveExport != commonentity.ExportDisable {
		t.Fatalf("disabled Plan() = %#v, %#v, %v", disabled, diagnostics, err)
	}
	assertPathAbsent(t, outputPath, "disabled plan")

	planned, diagnostics, err := service.Plan(context.Background(), commonentity.PlanInput{Spec: spec, Export: commonentity.ExportEnable, Prior: &disabled})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("enabled Plan() = %#v, %#v, %v", planned, diagnostics, err)
	}
	if planned.SourceSHA256 == "" || planned.ContentSHA256 == "" || planned.ObservedContentSHA256 != planned.ContentSHA256 {
		t.Fatalf("enabled planned state = %#v", planned)
	}
	assertPathAbsent(t, outputPath, "enabled plan")

	applied, diagnostics, err := service.Apply(context.Background(), commonentity.ApplyInput{
		Spec:          spec,
		CurrentExport: commonentity.ExportEnable,
		Planned:       planned,
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("Apply() = %#v, %#v, %v", applied, diagnostics, err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil || len(content) == 0 {
		t.Fatalf("read applied output = %q, %v", content, err)
	}

	if err := os.WriteFile(outputPath, []byte("external\n"), 0o644); err != nil {
		t.Fatalf("edit output: %v", err)
	}
	refreshed, err := service.Read(spec, applied)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	externalDigest := sha256.Sum256([]byte("external\n"))
	if refreshed.ObservedContentSHA256 != hex.EncodeToString(externalDigest[:]) {
		t.Fatalf("observed digest = %q", refreshed.ObservedContentSHA256)
	}
	warning, err := service.Delete(spec, applied)
	if err != nil || !strings.Contains(warning, "externally modified") {
		t.Fatalf("Delete() warning = %q, error = %v", warning, err)
	}
	if data, err := os.ReadFile(outputPath); err != nil || string(data) != "external\n" {
		t.Fatalf("preserved output = %q, %v", data, err)
	}
}

func TestDiagramServiceRejectsUnownedOutputAndSourceChanges(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	sourceDir := filepath.Join(directory, "source")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatalf("create source directory: %v", err)
	}
	writeSource(t, sourceDir, `resource "aws_s3_bucket" "logs" {}`)
	outputPath := filepath.Join(directory, "diagram.xal")
	spec := commonentity.DiagramSpec{SourceDir: sourceDir, OutputPath: outputPath, FrameID: "main"}
	service := newTestDiagramUsecase()

	if err := os.WriteFile(outputPath, []byte("unowned\n"), 0o644); err != nil {
		t.Fatalf("write unowned output: %v", err)
	}
	_, _, err := service.Plan(context.Background(), commonentity.PlanInput{Spec: spec, Export: commonentity.ExportEnable})
	var failure *commonentity.Failure
	if !errors.As(err, &failure) || failure.Summary != "Output already exists" {
		t.Fatalf("unowned Plan() error = %#v", err)
	}
	if err := os.Remove(outputPath); err != nil {
		t.Fatalf("remove unowned output: %v", err)
	}

	planned, _, err := service.Plan(context.Background(), commonentity.PlanInput{Spec: spec, Export: commonentity.ExportEnable})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	writeSource(t, sourceDir, `resource "aws_sqs_queue" "events" {}`)
	_, _, err = service.Apply(context.Background(), commonentity.ApplyInput{
		Spec:          spec,
		CurrentExport: commonentity.ExportEnable,
		Planned:       planned,
	})
	if !errors.As(err, &failure) || failure.Summary != "Terraform source changed after planning" {
		t.Fatalf("changed-source Apply() error = %#v", err)
	}
	assertPathAbsent(t, outputPath, "changed-source apply")
}

func writeSource(t *testing.T, directory, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, "main.tf"), []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("write Terraform source: %v", err)
	}
}

func assertPathAbsent(t *testing.T, path, operation string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s created output: %v", operation, err)
	}
}
