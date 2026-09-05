package controller

import (
	"context"
	"fmt"
	"path/filepath"

	frameworkdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
	"github.com/xaligo/terraform-provider/internal/usecase"
)

type DiagramResourceController interface {
	resource.Resource
	resource.ResourceWithConfigure
	resource.ResourceWithModifyPlan
	resource.ResourceWithValidateConfig
}

type diagramResourceController struct {
	providerData *xaligoentity.ProviderData
}

var (
	_ resource.Resource                   = &diagramResourceController{}
	_ resource.ResourceWithConfigure      = &diagramResourceController{}
	_ resource.ResourceWithModifyPlan     = &diagramResourceController{}
	_ resource.ResourceWithValidateConfig = &diagramResourceController{}
)

func NewDiagramResourceController() DiagramResourceController {
	return &diagramResourceController{}
}

func (rcvr *diagramResourceController) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_diagram"
}

func (rcvr *diagramResourceController) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = resourceschema.Schema{
		Description: "Manage one deterministic local .xal artifact generated from Terraform source.",
		Attributes: map[string]resourceschema.Attribute{
			"source_dir": resourceschema.StringAttribute{
				Required:    true,
				Description: "Directory containing direct regular .tf inputs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"output_path": resourceschema.StringAttribute{
				Required:    true,
				Description: "Destination .xal path. Relative paths resolve against source_dir.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"frame_id": resourceschema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
				Description: "Stable XAL frame identifier.",
			},
			"title": resourceschema.StringAttribute{
				Optional:    true,
				Description: "Optional visible frame title.",
			},
			"paper_size": resourceschema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("screen"),
				Description: "Frame paper size: screen, A5, A4, A3, A2, A1, Letter, Legal, or Tabloid.",
			},
			"orientation": resourceschema.StringAttribute{
				Optional: true, Computed: true, Default: stringdefault.StaticString("landscape"),
				Description: "Paper orientation: landscape or portrait.",
			},
			"grid_columns": resourceschema.Int64Attribute{
				Optional: true, Computed: true, Default: int64default.StaticInt64(1),
				Description: "Automatic XAL 12-column grid division: 1, 2, 3, 4, 6, or 12 columns.",
			},
			"grid_gap": resourceschema.Int64Attribute{
				Optional: true, Computed: true, Default: int64default.StaticInt64(16),
				Description: "Non-negative gap in pixels between generated grid columns.",
			},
			"fail_on_warning": resourceschema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Promote conversion warnings to errors.",
			},
			"overwrite": resourceschema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Allow replacing an unowned or externally modified destination.",
			},
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Stable identity derived from the normalized output path.",
			},
			"effective_export": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Provider export mode captured in this resource state.",
			},
			"source_sha256": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Digest of ordered Terraform input paths and bytes.",
			},
			"content_sha256": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Digest of the last successfully managed XAL content.",
			},
			"observed_content_sha256": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Digest currently observed on disk, or null when absent.",
			},
		},
		Blocks: map[string]resourceschema.Block{
			"row": resourceschema.ListNestedBlock{
				Description: "XAL 12-column row. Use items for automatic columns or nested col blocks for explicit spans.",
				NestedObject: resourceschema.NestedBlockObject{
					Attributes: map[string]resourceschema.Attribute{
						"items":    resourceschema.ListAttribute{Optional: true, ElementType: types.StringType},
						"gap":      resourceschema.Float64Attribute{Optional: true},
						"overflow": resourceschema.StringAttribute{Optional: true},
					},
					Blocks: map[string]resourceschema.Block{
						"col": resourceschema.ListNestedBlock{
							NestedObject: resourceschema.NestedBlockObject{Attributes: layoutBlockAttributes(true)},
						},
					},
				},
			},
			"container": resourceschema.ListNestedBlock{
				Description:  "XAL container composed from entries in the generated items map.",
				NestedObject: resourceschema.NestedBlockObject{Attributes: layoutBlockAttributes(false)},
			},
			"layout": resourceschema.ListNestedBlock{
				Description:  "Apply XAL layout attributes to one entry in the generated items map.",
				NestedObject: resourceschema.NestedBlockObject{Attributes: itemLayoutAttributes()},
			},
		},
	}
}

func itemLayoutAttributes() map[string]resourceschema.Attribute {
	attributes := layoutBlockAttributes(true)
	delete(attributes, "items")
	delete(attributes, "span")
	attributes["item"] = resourceschema.StringAttribute{Required: true}
	attributes["row"] = resourceschema.Float64Attribute{Optional: true}
	attributes["col"] = resourceschema.Float64Attribute{Optional: true}
	return attributes
}

func layoutBlockAttributes(column bool) map[string]resourceschema.Attribute {
	attributes := map[string]resourceschema.Attribute{
		"items":          resourceschema.ListAttribute{Required: true, ElementType: types.StringType},
		"layout":         resourceschema.StringAttribute{Optional: true},
		"class":          resourceschema.StringAttribute{Optional: true},
		"align":          resourceschema.StringAttribute{Optional: true},
		"overflow":       resourceschema.StringAttribute{Optional: true},
		"gap":            resourceschema.Float64Attribute{Optional: true},
		"content_width":  resourceschema.Float64Attribute{Optional: true},
		"content_height": resourceschema.Float64Attribute{Optional: true},
		"width":          resourceschema.Float64Attribute{Optional: true},
		"height":         resourceschema.Float64Attribute{Optional: true},
	}
	if column {
		attributes["span"] = resourceschema.Float64Attribute{Optional: true}
	} else {
		attributes["id"] = resourceschema.StringAttribute{Required: true}
	}
	return attributes
}

func (rcvr *diagramResourceController) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}
	data, ok := request.ProviderData.(*xaligoentity.ProviderData)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected provider configuration type",
			fmt.Sprintf("Expected configured xaligo provider data, got %T.", request.ProviderData),
		)
		return
	}
	rcvr.providerData = data
}

func (rcvr *diagramResourceController) ValidateConfig(ctx context.Context, request resource.ValidateConfigRequest, response *resource.ValidateConfigResponse) {
	var config xaligoentity.DiagramResourceModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}
	if !config.SourceDir.IsNull() && !config.SourceDir.IsUnknown() && config.SourceDir.ValueString() == "" {
		response.Diagnostics.AddError("Invalid source_dir", "source_dir must not be empty.")
	}
	if !config.OutputPath.IsNull() && !config.OutputPath.IsUnknown() && filepath.Ext(config.OutputPath.ValueString()) != ".xal" {
		response.Diagnostics.AddError("Invalid output_path", "output_path must use the lowercase .xal extension.")
	}
	if !config.FrameID.IsNull() && !config.FrameID.IsUnknown() {
		if err := usecase.ValidateFrameID(config.FrameID.ValueString()); err != nil {
			response.Diagnostics.AddError("Invalid frame_id", err.Error())
		}
	}
	if !config.PaperSize.IsNull() && !config.PaperSize.IsUnknown() && !xaligoentity.ValidPaperSize(config.PaperSize.ValueString()) {
		response.Diagnostics.AddError("Invalid paper_size", "paper_size must be screen, A5, A4, A3, A2, A1, Letter, Legal, or Tabloid.")
	}
	if !config.Orientation.IsNull() && !config.Orientation.IsUnknown() && !xaligoentity.ValidOrientation(config.Orientation.ValueString()) {
		response.Diagnostics.AddError("Invalid orientation", "orientation must be landscape or portrait.")
	}
	if !config.GridColumns.IsNull() && !config.GridColumns.IsUnknown() && !xaligoentity.ValidGridColumns(int(config.GridColumns.ValueInt64())) {
		response.Diagnostics.AddError("Invalid grid_columns", "grid_columns must be one of 1, 2, 3, 4, 6, or 12.")
	}
	if !config.GridGap.IsNull() && !config.GridGap.IsUnknown() && config.GridGap.ValueInt64() < 0 {
		response.Diagnostics.AddError("Invalid grid_gap", "grid_gap must be zero or greater.")
	}
}

func (rcvr *diagramResourceController) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	if request.Plan.Raw.IsNull() {
		return
	}
	if !rcvr.configured(&response.Diagnostics) {
		return
	}
	var plan xaligoentity.DiagramResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() || !resourceInputsKnown(plan, &response.Diagnostics) {
		return
	}

	var prior *commonentity.DiagramComputed
	if !request.State.Raw.IsNull() {
		var state xaligoentity.DiagramResourceModel
		response.Diagnostics.Append(request.State.Get(ctx, &state)...)
		if response.Diagnostics.HasError() {
			return
		}
		value := computedFromModel(state)
		prior = &value
	}
	computed, diagnostics, err := rcvr.providerData.Diagrams.Plan(ctx, commonentity.PlanInput{
		Spec:   specFromModel(ctx, plan, &response.Diagnostics),
		Export: rcvr.providerData.Export,
		Prior:  prior,
	})
	appendDomainDiagnostics(&response.Diagnostics, diagnostics)
	appendUsecaseError(&response.Diagnostics, err)
	if err != nil || response.Diagnostics.HasError() {
		return
	}
	setComputed(&plan, computed)
	response.Diagnostics.Append(response.Plan.Set(ctx, &plan)...)
}

func (rcvr *diagramResourceController) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan xaligoentity.DiagramResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}
	rcvr.apply(ctx, plan, "", &response.State, &response.Diagnostics)
}

func (rcvr *diagramResourceController) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan xaligoentity.DiagramResourceModel
	var state xaligoentity.DiagramResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	rcvr.apply(ctx, plan, stringValue(state.ContentSHA256), &response.State, &response.Diagnostics)
}

func (rcvr *diagramResourceController) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	if !rcvr.configured(&response.Diagnostics) {
		return
	}
	var state xaligoentity.DiagramResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	computed, err := rcvr.providerData.Diagrams.Read(specFromModel(ctx, state, &response.Diagnostics), computedFromModel(state))
	appendUsecaseError(&response.Diagnostics, err)
	if err != nil {
		return
	}
	setComputed(&state, computed)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (rcvr *diagramResourceController) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	if !rcvr.configured(&response.Diagnostics) {
		return
	}
	var state xaligoentity.DiagramResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}
	warning, err := rcvr.providerData.Diagrams.Delete(specFromModel(ctx, state, &response.Diagnostics), computedFromModel(state))
	appendUsecaseError(&response.Diagnostics, err)
	if err == nil && warning != "" {
		response.Diagnostics.AddWarning("Managed output was preserved", warning)
	}
}

func (rcvr *diagramResourceController) apply(ctx context.Context, plan xaligoentity.DiagramResourceModel, previousDigest string, state *tfsdk.State, diagnostics *frameworkdiag.Diagnostics) {
	if !rcvr.configured(diagnostics) {
		return
	}
	computed, domainDiagnostics, err := rcvr.providerData.Diagrams.Apply(ctx, commonentity.ApplyInput{
		Spec:                  specFromModel(ctx, plan, diagnostics),
		CurrentExport:         rcvr.providerData.Export,
		Planned:               computedFromModel(plan),
		PreviousContentSHA256: previousDigest,
	})
	appendDomainDiagnostics(diagnostics, domainDiagnostics)
	appendUsecaseError(diagnostics, err)
	if err != nil || diagnostics.HasError() {
		return
	}
	setComputed(&plan, computed)
	diagnostics.Append(state.Set(ctx, &plan)...)
}

func (rcvr *diagramResourceController) configured(diagnostics *frameworkdiag.Diagnostics) bool {
	if rcvr.providerData == nil || rcvr.providerData.Diagrams == nil {
		diagnostics.AddError("Provider is not configured", "The xaligo provider did not supply resource configuration.")
		return false
	}
	return true
}

func specFromModel(ctx context.Context, model xaligoentity.DiagramResourceModel, diagnostics *frameworkdiag.Diagnostics) commonentity.DiagramSpec {
	return commonentity.DiagramSpec{
		SourceDir:     stringValue(model.SourceDir),
		OutputPath:    stringValue(model.OutputPath),
		FrameID:       stringValue(model.FrameID),
		Title:         stringValue(model.Title),
		PaperSize:     stringValue(model.PaperSize),
		Orientation:   stringValue(model.Orientation),
		GridColumns:   intValue(model.GridColumns),
		GridGap:       intValue(model.GridGap),
		Rows:          rowsFromModel(ctx, model.Rows, diagnostics),
		Containers:    containersFromModel(ctx, model.Containers, diagnostics),
		Layouts:       layoutsFromModel(ctx, model.Layouts, diagnostics),
		FailOnWarning: boolValue(model.FailOnWarning),
		Overwrite:     boolValue(model.Overwrite),
	}
}

func rowsFromModel(ctx context.Context, value types.List, diagnostics *frameworkdiag.Diagnostics) []commonentity.RowSpec {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var models []xaligoentity.DiagramRowModel
	diagnostics.Append(value.ElementsAs(ctx, &models, false)...)
	rows := make([]commonentity.RowSpec, 0, len(models))
	for _, model := range models {
		var items []string
		if !model.Items.IsNull() && !model.Items.IsUnknown() {
			diagnostics.Append(model.Items.ElementsAs(ctx, &items, false)...)
		}
		var columnModels []xaligoentity.DiagramColumnModel
		if !model.Columns.IsNull() && !model.Columns.IsUnknown() {
			diagnostics.Append(model.Columns.ElementsAs(ctx, &columnModels, false)...)
		}
		columns := make([]commonentity.ColumnSpec, 0, len(columnModels))
		for _, column := range columnModels {
			var columnItems []string
			diagnostics.Append(column.Items.ElementsAs(ctx, &columnItems, false)...)
			columns = append(columns, commonentity.ColumnSpec{Items: columnItems, Span: floatValue(column.Span), Style: layoutStyleFromColumn(column)})
		}
		rows = append(rows, commonentity.RowSpec{Items: items, Columns: columns, Gap: floatValue(model.Gap), GapSet: !model.Gap.IsNull() && !model.Gap.IsUnknown(), Overflow: stringValue(model.Overflow)})
	}
	return rows
}

func containersFromModel(ctx context.Context, value types.List, diagnostics *frameworkdiag.Diagnostics) []commonentity.ContainerSpec {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var models []xaligoentity.DiagramContainerModel
	diagnostics.Append(value.ElementsAs(ctx, &models, false)...)
	containers := make([]commonentity.ContainerSpec, 0, len(models))
	for _, model := range models {
		var items []string
		diagnostics.Append(model.Items.ElementsAs(ctx, &items, false)...)
		containers = append(containers, commonentity.ContainerSpec{ID: stringValue(model.ID), Items: items, Style: layoutStyleFromContainer(model)})
	}
	return containers
}

func layoutsFromModel(ctx context.Context, value types.List, diagnostics *frameworkdiag.Diagnostics) []commonentity.ItemLayoutSpec {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	var models []xaligoentity.DiagramItemLayoutModel
	diagnostics.Append(value.ElementsAs(ctx, &models, false)...)
	layouts := make([]commonentity.ItemLayoutSpec, 0, len(models))
	for _, model := range models {
		layouts = append(layouts, commonentity.ItemLayoutSpec{Item: stringValue(model.Item), Style: commonentity.LayoutStyle{
			Layout: stringValue(model.Layout), Class: stringValue(model.Class), Align: stringValue(model.Align), Overflow: stringValue(model.Overflow),
			Gap: floatValue(model.Gap), GapSet: !model.Gap.IsNull() && !model.Gap.IsUnknown(), ContentWidth: floatValue(model.ContentWidth), ContentHeight: floatValue(model.ContentHeight),
			Width: floatValue(model.Width), Height: floatValue(model.Height), Row: floatValue(model.Row), Col: floatValue(model.Col),
		}})
	}
	return layouts
}

func layoutStyleFromColumn(model xaligoentity.DiagramColumnModel) commonentity.LayoutStyle {
	return commonentity.LayoutStyle{Layout: stringValue(model.Layout), Class: stringValue(model.Class), Align: stringValue(model.Align), Overflow: stringValue(model.Overflow), Gap: floatValue(model.Gap), GapSet: !model.Gap.IsNull() && !model.Gap.IsUnknown(), ContentWidth: floatValue(model.ContentWidth), ContentHeight: floatValue(model.ContentHeight), Width: floatValue(model.Width), Height: floatValue(model.Height)}
}

func layoutStyleFromContainer(model xaligoentity.DiagramContainerModel) commonentity.LayoutStyle {
	return commonentity.LayoutStyle{Layout: stringValue(model.Layout), Class: stringValue(model.Class), Align: stringValue(model.Align), Overflow: stringValue(model.Overflow), Gap: floatValue(model.Gap), GapSet: !model.Gap.IsNull() && !model.Gap.IsUnknown(), ContentWidth: floatValue(model.ContentWidth), ContentHeight: floatValue(model.ContentHeight), Width: floatValue(model.Width), Height: floatValue(model.Height)}
}

func computedFromModel(model xaligoentity.DiagramResourceModel) commonentity.DiagramComputed {
	return commonentity.DiagramComputed{
		ID:                    stringValue(model.ID),
		EffectiveExport:       commonentity.ExportMode(stringValue(model.EffectiveExport)),
		SourceSHA256:          stringValue(model.SourceSHA256),
		ContentSHA256:         stringValue(model.ContentSHA256),
		ObservedContentSHA256: stringValue(model.ObservedContentSHA256),
	}
}

func setComputed(model *xaligoentity.DiagramResourceModel, computed commonentity.DiagramComputed) {
	model.ID = nullableString(computed.ID)
	model.EffectiveExport = nullableString(string(computed.EffectiveExport))
	model.SourceSHA256 = nullableString(computed.SourceSHA256)
	model.ContentSHA256 = nullableString(computed.ContentSHA256)
	model.ObservedContentSHA256 = nullableString(computed.ObservedContentSHA256)
}

func stringValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func boolValue(value types.Bool) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueBool()
}

func intValue(value types.Int64) int {
	if value.IsNull() || value.IsUnknown() {
		return 0
	}
	return int(value.ValueInt64())
}

func floatValue(value types.Float64) float64 {
	if value.IsNull() || value.IsUnknown() {
		return 0
	}
	return value.ValueFloat64()
}

func nullableString(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func resourceInputsKnown(model xaligoentity.DiagramResourceModel, diagnostics interface {
	AddError(string, string)
}) bool {
	if model.SourceDir.IsNull() || model.SourceDir.IsUnknown() || model.OutputPath.IsNull() || model.OutputPath.IsUnknown() {
		diagnostics.AddError("Unknown diagram path", "source_dir and output_path must be known during planning.")
		return false
	}
	if model.FrameID.IsNull() || model.FrameID.IsUnknown() || model.PaperSize.IsNull() || model.PaperSize.IsUnknown() || model.Orientation.IsNull() || model.Orientation.IsUnknown() || model.GridColumns.IsNull() || model.GridColumns.IsUnknown() || model.GridGap.IsNull() || model.GridGap.IsUnknown() || model.FailOnWarning.IsNull() || model.FailOnWarning.IsUnknown() || model.Overwrite.IsNull() || model.Overwrite.IsUnknown() {
		diagnostics.AddError("Unknown diagram option", "frame_id, paper_size, orientation, grid_columns, grid_gap, fail_on_warning, and overwrite must be known during planning.")
		return false
	}
	if model.Rows.IsUnknown() {
		diagnostics.AddError("Unknown row layout", "row blocks and their items must be known during planning.")
		return false
	}
	if model.Containers.IsUnknown() {
		diagnostics.AddError("Unknown container layout", "container blocks and their items must be known during planning.")
		return false
	}
	if model.Layouts.IsUnknown() {
		diagnostics.AddError("Unknown item layout", "layout blocks must be known during planning.")
		return false
	}
	return true
}
