package common

type ConversionRequest struct {
	SourceDir     string
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
}

type ConversionResult struct {
	Content       []byte
	SourceSHA256  string
	ContentSHA256 string
	Diagnostics   []Diagnostic
}
