package common

import "context"

type ExportMode string

const (
	ExportEnable  ExportMode = "enable"
	ExportDisable ExportMode = "disable"
)

func (rcvr ExportMode) Valid() bool {
	return rcvr == ExportEnable || rcvr == ExportDisable
}

type DiagramSpec struct {
	SourceDir     string
	OutputPath    string
	FrameID       string
	Title         string
	PaperSize     string
	Orientation   string
	GridColumns   int
	GridGap       int
	Rows          []RowSpec
	Containers    []ContainerSpec
	Layouts       []ItemLayoutSpec
	FailOnWarning bool
	Overwrite     bool
}

type RowSpec struct {
	Items    []string
	Columns  []ColumnSpec
	Gap      float64
	GapSet   bool
	Overflow string
}

type ColumnSpec struct {
	Items []string
	Span  float64
	Style LayoutStyle
}

type ContainerSpec struct {
	ID    string
	Items []string
	Style LayoutStyle
}

type LayoutStyle struct {
	Layout        string
	Class         string
	Align         string
	Overflow      string
	Gap           float64
	GapSet        bool
	ContentWidth  float64
	ContentHeight float64
	Width         float64
	Height        float64
	Row           float64
	Col           float64
}

type ItemLayoutSpec struct {
	Item  string
	Style LayoutStyle
}

type DiagramComputed struct {
	ID                    string
	EffectiveExport       ExportMode
	SourceSHA256          string
	ContentSHA256         string
	ObservedContentSHA256 string
}

type PlanInput struct {
	Spec   DiagramSpec
	Export ExportMode
	Prior  *DiagramComputed
}

type ApplyInput struct {
	Spec                  DiagramSpec
	CurrentExport         ExportMode
	Planned               DiagramComputed
	PreviousContentSHA256 string
}

type GenerateResult struct {
	OutputPath  string
	Diagnostics []Diagnostic
}

type DiagramService interface {
	Plan(context.Context, PlanInput) (DiagramComputed, []Diagnostic, error)
	Apply(context.Context, ApplyInput) (DiagramComputed, []Diagnostic, error)
	Read(DiagramSpec, DiagramComputed) (DiagramComputed, error)
	Delete(DiagramSpec, DiagramComputed) (string, error)
	Generate(context.Context, DiagramSpec) (GenerateResult, error)
}

type ItemCatalogService interface {
	Items(context.Context, string) (map[string]string, []Diagnostic, error)
}
