package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	localentity "github.com/xaligo/terraform-provider/internal/entity/local"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
	"github.com/xaligo/terraform-provider/internal/repository"
)

var ErrConversion = errors.New("Terraform to XAL conversion failed")

type DiagramUsecase interface {
	Convert(context.Context, commonentity.ConversionRequest) (commonentity.ConversionResult, error)
	Plan(context.Context, commonentity.PlanInput) (commonentity.DiagramComputed, []commonentity.Diagnostic, error)
	Apply(context.Context, commonentity.ApplyInput) (commonentity.DiagramComputed, []commonentity.Diagnostic, error)
	Read(commonentity.DiagramSpec, commonentity.DiagramComputed) (commonentity.DiagramComputed, error)
	Delete(commonentity.DiagramSpec, commonentity.DiagramComputed) (string, error)
	Generate(context.Context, commonentity.DiagramSpec) (commonentity.GenerateResult, error)
	Items(context.Context, string) (map[string]string, []commonentity.Diagnostic, error)
}

func (rcvr *diagramUsecase) Items(ctx context.Context, sourceDir string) (map[string]string, []commonentity.Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	config, diagnostics, err := rcvr.sources.Load(sourceDir)
	if err != nil {
		return nil, diagnostics, fmt.Errorf("load Terraform items: %w", err)
	}
	if commonentity.HasErrors(diagnostics) {
		return nil, diagnostics, ErrConversion
	}
	graph, graphDiagnostics := commonentity.BuildGraph(config)
	diagnostics = append(diagnostics, graphDiagnostics...)
	commonentity.SortDiagnostics(diagnostics)
	if commonentity.HasErrors(diagnostics) {
		return nil, diagnostics, ErrConversion
	}
	items := make(map[string]string, len(graph.Nodes))
	for _, node := range graph.Nodes {
		items[node.Address] = node.Address
	}
	return items, diagnostics, nil
}

type diagramUsecase struct {
	sources   repository.TerraformRepository
	mapper    repository.AWSRepository
	marshaler repository.XaligoRepository
	paths     repository.PathRepository
	artifacts repository.ArtifactRepository
}

func NewDiagramUsecase(
	sources repository.TerraformRepository,
	mapper repository.AWSRepository,
	marshaler repository.XaligoRepository,
	paths repository.PathRepository,
	artifacts repository.ArtifactRepository,
) DiagramUsecase {
	return &diagramUsecase{
		sources:   sources,
		mapper:    mapper,
		marshaler: marshaler,
		paths:     paths,
		artifacts: artifacts,
	}
}

func (rcvr *diagramUsecase) Convert(ctx context.Context, request commonentity.ConversionRequest) (commonentity.ConversionResult, error) {
	if err := ctx.Err(); err != nil {
		return commonentity.ConversionResult{}, err
	}
	if err := ValidateFrameID(request.FrameID); err != nil {
		result := commonentity.ConversionResult{Diagnostics: []commonentity.Diagnostic{{
			Code:     "CONVERT-E001",
			Severity: commonentity.SeverityError,
			Summary:  "Invalid frame id",
			Detail:   err.Error(),
		}}}
		return result, ErrConversion
	}
	if err := ValidateDiagramLayout(request.PaperSize, request.Orientation, request.GridColumns, request.GridGap); err != nil {
		result := commonentity.ConversionResult{Diagnostics: []commonentity.Diagnostic{{
			Code: "CONVERT-E004", Severity: commonentity.SeverityError,
			Summary: "Invalid diagram layout", Detail: err.Error(),
		}}}
		return result, ErrConversion
	}

	config, loadDiagnostics, err := rcvr.sources.Load(request.SourceDir)
	result := commonentity.ConversionResult{Diagnostics: append([]commonentity.Diagnostic(nil), loadDiagnostics...)}
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, commonentity.Diagnostic{
			Code:     "CONVERT-E002",
			Severity: commonentity.SeverityError,
			Summary:  "Terraform source could not be loaded",
			Detail:   err.Error(),
		})
		commonentity.SortDiagnostics(result.Diagnostics)
		return result, ErrConversion
	}
	result.SourceSHA256 = sourceDigest(config.Files)
	if commonentity.HasErrors(result.Diagnostics) {
		return result, ErrConversion
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	graph, graphDiagnostics := commonentity.BuildGraph(config)
	result.Diagnostics = append(result.Diagnostics, graphDiagnostics...)
	if commonentity.HasErrors(result.Diagnostics) {
		commonentity.SortDiagnostics(result.Diagnostics)
		return result, ErrConversion
	}

	document, mappingDiagnostics := rcvr.mapper.Map(graph, xaligoentity.DiagramOptions{
		FrameID:     request.FrameID,
		Title:       request.Title,
		PaperSize:   request.PaperSize,
		Orientation: request.Orientation,
		GridColumns: request.GridColumns,
		GridGap:     request.GridGap,
		Rows:        request.Rows,
		Containers:  request.Containers,
		Layouts:     request.Layouts,
	})
	result.Diagnostics = append(result.Diagnostics, mappingDiagnostics...)
	if request.FailOnWarning {
		commonentity.PromoteWarnings(result.Diagnostics)
	}
	if commonentity.HasErrors(result.Diagnostics) {
		commonentity.SortDiagnostics(result.Diagnostics)
		return result, ErrConversion
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}

	content, err := rcvr.marshaler.Marshal(document)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, commonentity.Diagnostic{
			Code:     "CONVERT-E003",
			Severity: commonentity.SeverityError,
			Summary:  "Generated XAL is invalid",
			Detail:   err.Error(),
		})
		commonentity.SortDiagnostics(result.Diagnostics)
		return result, ErrConversion
	}
	result.Content = content
	contentHash := sha256.Sum256(content)
	result.ContentSHA256 = hex.EncodeToString(contentHash[:])
	commonentity.SortDiagnostics(result.Diagnostics)
	return result, nil
}

func (rcvr *diagramUsecase) Plan(ctx context.Context, input commonentity.PlanInput) (commonentity.DiagramComputed, []commonentity.Diagnostic, error) {
	if !input.Export.Valid() {
		return commonentity.DiagramComputed{}, nil, fail("Invalid xaligo export mode", fmt.Errorf("expected enable or disable, got %q", input.Export))
	}
	if input.Export == commonentity.ExportDisable {
		paths, err := rcvr.paths.ResolveLexical(input.Spec.SourceDir, input.Spec.OutputPath)
		if err != nil {
			return commonentity.DiagramComputed{}, nil, fail("Invalid diagram paths", err)
		}
		computed := commonentity.DiagramComputed{ID: rcvr.paths.StableID(paths.OutputPath)}
		if input.Prior != nil {
			computed = *input.Prior
			if computed.ID == "" {
				computed.ID = rcvr.paths.StableID(paths.OutputPath)
			}
		}
		computed.EffectiveExport = commonentity.ExportDisable
		return computed, nil, nil
	}

	paths, err := rcvr.paths.Resolve(input.Spec.SourceDir, input.Spec.OutputPath)
	if err != nil {
		return commonentity.DiagramComputed{}, nil, fail("Invalid diagram paths", err)
	}
	conversion, err := rcvr.Convert(ctx, conversionRequest(paths.SourceDir, input.Spec))
	if err != nil {
		return commonentity.DiagramComputed{}, conversion.Diagnostics, conversionError(err)
	}
	inspection, err := rcvr.artifacts.Inspect(paths.OutputPath)
	if err != nil {
		return commonentity.DiagramComputed{}, conversion.Diagnostics, fail("Output inspection failed", err)
	}
	if inspection.Exists && !input.Spec.Overwrite {
		ownedDigest := ""
		if input.Prior != nil {
			ownedDigest = input.Prior.ContentSHA256
		}
		if ownedDigest == "" {
			return commonentity.DiagramComputed{}, conversion.Diagnostics, fail(
				"Output already exists",
				errors.New("the destination is not owned by this resource; set overwrite = true to take ownership"),
			)
		}
		if inspection.Digest != ownedDigest {
			return commonentity.DiagramComputed{}, conversion.Diagnostics, fail(
				"Output was modified externally",
				errors.New("the destination no longer matches the managed digest; set overwrite = true to replace it"),
			)
		}
	}
	return commonentity.DiagramComputed{
		ID:                    rcvr.paths.StableID(paths.OutputPath),
		EffectiveExport:       commonentity.ExportEnable,
		SourceSHA256:          conversion.SourceSHA256,
		ContentSHA256:         conversion.ContentSHA256,
		ObservedContentSHA256: conversion.ContentSHA256,
	}, conversion.Diagnostics, nil
}

func (rcvr *diagramUsecase) Apply(ctx context.Context, input commonentity.ApplyInput) (commonentity.DiagramComputed, []commonentity.Diagnostic, error) {
	if input.Planned.EffectiveExport == "" {
		return commonentity.DiagramComputed{}, nil, fail("Unknown effective export mode", errors.New("effective_export must be known before apply"))
	}
	if input.Planned.EffectiveExport != input.CurrentExport {
		return commonentity.DiagramComputed{}, nil, fail(
			"Provider export mode changed after planning",
			errors.New("create a new Terraform plan before applying the changed xaligo export mode"),
		)
	}
	if input.Planned.EffectiveExport == commonentity.ExportDisable {
		return input.Planned, nil, nil
	}

	paths, err := rcvr.paths.Resolve(input.Spec.SourceDir, input.Spec.OutputPath)
	if err != nil {
		return commonentity.DiagramComputed{}, nil, fail("Invalid diagram paths", err)
	}
	if input.Planned.ID == "" || input.Planned.ID != rcvr.paths.StableID(paths.OutputPath) {
		return commonentity.DiagramComputed{}, nil, fail(
			"Diagram path changed after planning",
			errors.New("the normalized output path no longer matches the planned resource identity; create a new plan"),
		)
	}
	conversion, err := rcvr.Convert(ctx, conversionRequest(paths.SourceDir, input.Spec))
	if err != nil {
		return commonentity.DiagramComputed{}, conversion.Diagnostics, conversionError(err)
	}
	if input.Planned.SourceSHA256 == "" || input.Planned.SourceSHA256 != conversion.SourceSHA256 {
		return commonentity.DiagramComputed{}, conversion.Diagnostics, fail(
			"Terraform source changed after planning",
			errors.New("the source digest no longer matches the plan; create a new Terraform plan before applying"),
		)
	}
	if input.Planned.ContentSHA256 == "" || input.Planned.ContentSHA256 != conversion.ContentSHA256 {
		return commonentity.DiagramComputed{}, conversion.Diagnostics, fail(
			"Generated XAL changed after planning",
			errors.New("the generated content digest no longer matches the plan; create a new Terraform plan before applying"),
		)
	}
	if err := rcvr.artifacts.Write(paths.OutputPath, conversion.Content, localentity.WriteOptions{
		Overwrite:              input.Spec.Overwrite,
		ExpectedPreviousDigest: input.PreviousContentSHA256,
	}); err != nil {
		return commonentity.DiagramComputed{}, conversion.Diagnostics, fail("XAL output write failed", err)
	}
	computed := input.Planned
	computed.SourceSHA256 = conversion.SourceSHA256
	computed.ContentSHA256 = conversion.ContentSHA256
	computed.ObservedContentSHA256 = conversion.ContentSHA256
	return computed, conversion.Diagnostics, nil
}

func (rcvr *diagramUsecase) Read(spec commonentity.DiagramSpec, state commonentity.DiagramComputed) (commonentity.DiagramComputed, error) {
	if state.EffectiveExport == commonentity.ExportDisable {
		return state, nil
	}
	paths, err := rcvr.paths.ResolveLexical(spec.SourceDir, spec.OutputPath)
	if err != nil {
		return commonentity.DiagramComputed{}, fail("Invalid diagram paths", err)
	}
	inspection, err := rcvr.artifacts.Inspect(paths.OutputPath)
	if err != nil {
		return commonentity.DiagramComputed{}, fail("Output inspection failed", err)
	}
	state.ObservedContentSHA256 = ""
	if inspection.Exists {
		state.ObservedContentSHA256 = inspection.Digest
	}
	return state, nil
}

func (rcvr *diagramUsecase) Delete(spec commonentity.DiagramSpec, state commonentity.DiagramComputed) (string, error) {
	if state.ContentSHA256 == "" {
		return "", nil
	}
	paths, err := rcvr.paths.ResolveLexical(spec.SourceDir, spec.OutputPath)
	if err != nil {
		return "", fail("Invalid diagram paths", err)
	}
	result, err := rcvr.artifacts.Delete(paths.OutputPath, state.ContentSHA256)
	if err != nil {
		return "", fail("Managed output deletion failed", err)
	}
	return result.Warning, nil
}

func (rcvr *diagramUsecase) Generate(ctx context.Context, spec commonentity.DiagramSpec) (commonentity.GenerateResult, error) {
	paths, err := rcvr.paths.Resolve(spec.SourceDir, spec.OutputPath)
	if err != nil {
		return commonentity.GenerateResult{}, fail("Invalid diagram paths", err)
	}
	conversion, err := rcvr.Convert(ctx, conversionRequest(paths.SourceDir, spec))
	result := commonentity.GenerateResult{OutputPath: paths.OutputPath, Diagnostics: conversion.Diagnostics}
	if err != nil {
		return result, conversionError(err)
	}
	if err := rcvr.artifacts.Write(paths.OutputPath, conversion.Content, localentity.WriteOptions{Overwrite: spec.Overwrite}); err != nil {
		return result, fail("XAL output write failed", err)
	}
	return result, nil
}

func conversionRequest(sourceDir string, spec commonentity.DiagramSpec) commonentity.ConversionRequest {
	return commonentity.ConversionRequest{
		SourceDir:     sourceDir,
		FrameID:       spec.FrameID,
		Title:         spec.Title,
		PaperSize:     spec.PaperSize,
		Orientation:   spec.Orientation,
		GridColumns:   spec.GridColumns,
		GridGap:       spec.GridGap,
		Rows:          append([]commonentity.RowSpec(nil), spec.Rows...),
		Containers:    append([]commonentity.ContainerSpec(nil), spec.Containers...),
		Layouts:       append([]commonentity.ItemLayoutSpec(nil), spec.Layouts...),
		FailOnWarning: spec.FailOnWarning,
	}
}

func conversionError(err error) error {
	if errors.Is(err, ErrConversion) {
		return err
	}
	return fail("Terraform conversion failed", err)
}

func sourceDigest(files []commonentity.SourceFile) string {
	hash := sha256.New()
	var length [8]byte
	for _, file := range files {
		binary.BigEndian.PutUint64(length[:], uint64(len(file.Path)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(file.Path))
		binary.BigEndian.PutUint64(length[:], uint64(len(file.Bytes)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write(file.Bytes)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func ValidateFrameID(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("frame_id must not be empty")
	}
	for _, current := range value {
		if unicode.IsLetter(current) || unicode.IsDigit(current) || current == '_' || current == '-' {
			continue
		}
		return fmt.Errorf("frame_id %q may contain only letters, digits, underscores, and hyphens", value)
	}
	return nil
}

func ValidateDiagramLayout(paperSize, orientation string, gridColumns, gridGap int) error {
	if paperSize != "" && !xaligoentity.ValidPaperSize(paperSize) {
		return fmt.Errorf("unsupported paper_size %q", paperSize)
	}
	if orientation != "" && !xaligoentity.ValidOrientation(orientation) {
		return fmt.Errorf("unsupported orientation %q", orientation)
	}
	if gridColumns != 0 && !xaligoentity.ValidGridColumns(gridColumns) {
		return fmt.Errorf("grid_columns must be one of 1, 2, 3, 4, 6, or 12")
	}
	if gridGap < 0 {
		return fmt.Errorf("grid_gap must be zero or greater")
	}
	return nil
}
