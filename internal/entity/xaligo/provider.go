package xaligo

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

type ProviderData struct {
	Export   commonentity.ExportMode
	Diagrams commonentity.DiagramService
	Items    commonentity.ItemCatalogService
}

type ItemsDataSourceModel struct {
	SourceDir types.String `tfsdk:"source_dir"`
	Items     types.Map    `tfsdk:"items"`
}

type ProviderModel struct {
	Export types.String `tfsdk:"export"`
}

type DiagramResourceModel struct {
	SourceDir             types.String `tfsdk:"source_dir"`
	OutputPath            types.String `tfsdk:"output_path"`
	FrameID               types.String `tfsdk:"frame_id"`
	Title                 types.String `tfsdk:"title"`
	PaperSize             types.String `tfsdk:"paper_size"`
	Orientation           types.String `tfsdk:"orientation"`
	GridColumns           types.Int64  `tfsdk:"grid_columns"`
	GridGap               types.Int64  `tfsdk:"grid_gap"`
	Rows                  types.List   `tfsdk:"row"`
	Containers            types.List   `tfsdk:"container"`
	Layouts               types.List   `tfsdk:"layout"`
	FailOnWarning         types.Bool   `tfsdk:"fail_on_warning"`
	Overwrite             types.Bool   `tfsdk:"overwrite"`
	ID                    types.String `tfsdk:"id"`
	EffectiveExport       types.String `tfsdk:"effective_export"`
	SourceSHA256          types.String `tfsdk:"source_sha256"`
	ContentSHA256         types.String `tfsdk:"content_sha256"`
	ObservedContentSHA256 types.String `tfsdk:"observed_content_sha256"`
}

type DiagramRowModel struct {
	Items    types.List    `tfsdk:"items"`
	Columns  types.List    `tfsdk:"col"`
	Gap      types.Float64 `tfsdk:"gap"`
	Overflow types.String  `tfsdk:"overflow"`
}

type DiagramColumnModel struct {
	Items         types.List    `tfsdk:"items"`
	Span          types.Float64 `tfsdk:"span"`
	Layout        types.String  `tfsdk:"layout"`
	Class         types.String  `tfsdk:"class"`
	Align         types.String  `tfsdk:"align"`
	Overflow      types.String  `tfsdk:"overflow"`
	Gap           types.Float64 `tfsdk:"gap"`
	ContentWidth  types.Float64 `tfsdk:"content_width"`
	ContentHeight types.Float64 `tfsdk:"content_height"`
	Width         types.Float64 `tfsdk:"width"`
	Height        types.Float64 `tfsdk:"height"`
}

type DiagramContainerModel struct {
	ID            types.String  `tfsdk:"id"`
	Items         types.List    `tfsdk:"items"`
	Layout        types.String  `tfsdk:"layout"`
	Class         types.String  `tfsdk:"class"`
	Align         types.String  `tfsdk:"align"`
	Overflow      types.String  `tfsdk:"overflow"`
	Gap           types.Float64 `tfsdk:"gap"`
	ContentWidth  types.Float64 `tfsdk:"content_width"`
	ContentHeight types.Float64 `tfsdk:"content_height"`
	Width         types.Float64 `tfsdk:"width"`
	Height        types.Float64 `tfsdk:"height"`
}

type DiagramItemLayoutModel struct {
	Item          types.String  `tfsdk:"item"`
	Layout        types.String  `tfsdk:"layout"`
	Class         types.String  `tfsdk:"class"`
	Align         types.String  `tfsdk:"align"`
	Overflow      types.String  `tfsdk:"overflow"`
	Gap           types.Float64 `tfsdk:"gap"`
	ContentWidth  types.Float64 `tfsdk:"content_width"`
	ContentHeight types.Float64 `tfsdk:"content_height"`
	Width         types.Float64 `tfsdk:"width"`
	Height        types.Float64 `tfsdk:"height"`
	Row           types.Float64 `tfsdk:"row"`
	Col           types.Float64 `tfsdk:"col"`
}
