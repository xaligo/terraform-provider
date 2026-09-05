package usecase_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	"github.com/xaligo/terraform-provider/internal/repository"
	"github.com/xaligo/terraform-provider/internal/usecase"
)

func TestConvertMatchesSimpleVPCGoldenAndIsDeterministic(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	request := commonentity.ConversionRequest{
		SourceDir: filepath.Join(root, "samples", "simple-vpc", "source"),
		FrameID:   "main",
		Title:     "Simple VPC",
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
	expected, err := os.ReadFile(filepath.Join(root, "samples", "simple-vpc", "expected.xal"))
	if err != nil {
		t.Fatalf("read golden XAL: %v", err)
	}
	if !reflect.DeepEqual(first.Content, expected) {
		t.Fatalf("generated XAL differs from golden\n--- generated ---\n%s\n--- expected ---\n%s", first.Content, expected)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("repeated conversions differ\nfirst: %#v\nsecond: %#v", first, second)
	}
	if len(first.Diagnostics) != 1 || first.Diagnostics[0].Code != "MAPPING-W001" {
		t.Fatalf("Convert() diagnostics = %#v", first.Diagnostics)
	}
	if first.SourceSHA256 == "" || first.ContentSHA256 == "" {
		t.Fatal("Convert() returned empty digests")
	}
}

func TestConvertFailOnWarning(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	result, err := newTestDiagramUsecase().Convert(context.Background(), commonentity.ConversionRequest{
		SourceDir:     filepath.Join(root, "samples", "simple-vpc", "source"),
		FrameID:       "main",
		FailOnWarning: true,
	})
	if !errors.Is(err, usecase.ErrConversion) {
		t.Fatalf("Convert() error = %v, want ErrConversion", err)
	}
	if !commonentity.HasErrors(result.Diagnostics) || len(result.Content) != 0 {
		t.Fatalf("Convert() result = %#v", result)
	}
}

func TestValidateFrameID(t *testing.T) {
	t.Parallel()

	for _, valid := range []string{"main", "frame_1", "日本語-1"} {
		if err := usecase.ValidateFrameID(valid); err != nil {
			t.Errorf("ValidateFrameID(%q) error = %v", valid, err)
		}
	}
	for _, invalid := range []string{"", " ", "frame.one", "frame/one"} {
		if err := usecase.ValidateFrameID(invalid); err == nil {
			t.Errorf("ValidateFrameID(%q) unexpectedly succeeded", invalid)
		}
	}
}

func TestValidateDiagramLayout(t *testing.T) {
	t.Parallel()

	if err := usecase.ValidateDiagramLayout("A3", "landscape", 3, 20); err != nil {
		t.Fatalf("ValidateDiagramLayout() error = %v", err)
	}
	for _, test := range []struct {
		paper, orientation string
		columns, gap       int
	}{
		{paper: "A0", orientation: "landscape", columns: 3, gap: 16},
		{paper: "A4", orientation: "diagonal", columns: 3, gap: 16},
		{paper: "A4", orientation: "portrait", columns: 5, gap: 16},
		{paper: "A4", orientation: "portrait", columns: 3, gap: -1},
	} {
		if err := usecase.ValidateDiagramLayout(test.paper, test.orientation, test.columns, test.gap); err == nil {
			t.Errorf("ValidateDiagramLayout(%q, %q, %d, %d) unexpectedly succeeded", test.paper, test.orientation, test.columns, test.gap)
		}
	}
}

func newTestDiagramUsecase() usecase.DiagramUsecase {
	return usecase.NewDiagramUsecase(
		repository.NewTerraformRepository(),
		repository.NewAWSRepository(),
		repository.NewXaligoRepository(),
		repository.NewPathRepository(),
		repository.NewArtifactRepository(),
	)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
