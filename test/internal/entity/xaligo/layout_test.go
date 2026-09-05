package xaligo_test

import (
	"testing"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
)

func TestApplyExplicitRowsResolvesTerraformAddressesFromItemMap(t *testing.T) {
	t.Parallel()

	children := []xaligoentity.XALElement{{
		Tag: "generic-group", ID: "services", Children: []xaligoentity.XALElement{
			{Tag: "item", ID: "1", Name: "aws_s3_bucket-logs"},
			{Tag: "item", ID: "2", Name: "aws_lambda_function-worker"},
		},
	}}
	result, err := xaligoentity.ApplyExplicitRows(children, []commonentity.RowSpec{{Items: []string{
		"items.aws_s3_bucket.logs", "items.aws_lambda_function.worker",
	}}}, 24)
	if err != nil {
		t.Fatalf("ApplyExplicitRows() error = %v", err)
	}
	items, err := xaligoentity.BuildItemMap(children)
	if err != nil || items["aws_s3_bucket-logs"].Name != "aws_s3_bucket-logs" || items["services"].ID != "services" {
		t.Fatalf("BuildItemMap() = %#v, %v", items, err)
	}
	row := result[0].Children[0]
	if row.Tag != "row" || row.Gap != 24 || len(row.Children) != 2 || row.Children[0].Span != 6 {
		t.Fatalf("explicit row = %#v", row)
	}
}

func TestApplyExplicitRowsRejectsItemsFromDifferentParents(t *testing.T) {
	t.Parallel()

	children := []xaligoentity.XALElement{
		{Tag: "generic-group", ID: "left", Children: []xaligoentity.XALElement{{Tag: "rectangle", ID: "one"}}},
		{Tag: "generic-group", ID: "right", Children: []xaligoentity.XALElement{{Tag: "rectangle", ID: "two"}}},
	}
	if _, err := xaligoentity.ApplyExplicitRows(children, []commonentity.RowSpec{{Items: []string{"one", "two"}}}, 16); err == nil {
		t.Fatal("ApplyExplicitRows() unexpectedly accepted cross-parent items")
	}
}

func TestApplyExplicitLayoutSupportsContainerRowColAndSharedAttributes(t *testing.T) {
	t.Parallel()

	children := []xaligoentity.XALElement{{Tag: "generic-group", ID: "services", Children: rectangles(4, "node")}}
	result, err := xaligoentity.ApplyExplicitLayout(children,
		[]commonentity.ContainerSpec{{
			ID: "pair", Items: []string{"node-node-a", "node-node-b"},
			Style: commonentity.LayoutStyle{Layout: "horizontal", Class: "pa-2", Align: "middle-spread", Overflow: "visible", Gap: 0, GapSet: true, ContentWidth: 300, ContentHeight: 160, Width: 320, Height: 180},
		}},
		[]commonentity.RowSpec{{Gap: 12, GapSet: true, Overflow: "visible", Columns: []commonentity.ColumnSpec{
			{Items: []string{"pair"}, Span: 4, Style: commonentity.LayoutStyle{Class: "pa-1"}},
			{Items: []string{"node-node-c", "node-node-d"}, Span: 8, Style: commonentity.LayoutStyle{Layout: "horizontal", Col: 2}},
		}}},
		[]commonentity.ItemLayoutSpec{{Item: "services", Style: commonentity.LayoutStyle{Layout: "staggered", Row: 2}}}, 16)
	if err != nil {
		t.Fatalf("ApplyExplicitLayout() error = %v", err)
	}
	group := result[0]
	if group.Layout != "staggered" || group.Row != 2 || len(group.Children) != 1 {
		t.Fatalf("styled group = %#v", group)
	}
	row := group.Children[0]
	if row.Tag != "row" || row.Gap != 12 || row.Overflow != "visible" || len(row.Children) != 2 {
		t.Fatalf("explicit row = %#v", row)
	}
	container := row.Children[0].Children[0]
	if container.Tag != "container" || container.Layout != "horizontal" || !container.GapSet || container.Gap != 0 || container.ContentWidth != 300 {
		t.Fatalf("explicit container = %#v", container)
	}
	if row.Children[0].Span != 4 || row.Children[1].Span != 8 || row.Children[1].Layout != "horizontal" {
		t.Fatalf("explicit columns = %#v", row.Children)
	}
}

func TestNewXALDocumentExpandsFrameAndWeightsUnevenBranches(t *testing.T) {
	t.Parallel()

	children := []xaligoentity.XALElement{
		{
			Tag:      "generic-group",
			ID:       "large",
			Children: rectangles(8, "large"),
		},
		{
			Tag:      "generic-group",
			ID:       "small",
			Children: rectangles(4, "small"),
		},
	}
	document := xaligoentity.NewXALDocument(xaligoentity.DiagramOptions{
		FrameID: "main",
		Title:   "Adaptive layout",
	}, children)

	if got, want := document.Frame.Width, 1280; got != want {
		t.Fatalf("frame width = %d, want %d", got, want)
	}
	if got, want := document.Frame.Height, 1104; got != want {
		t.Fatalf("frame height = %d, want %d", got, want)
	}
	if got, want := document.Frame.Children[0].Row, float64(688); got != want {
		t.Errorf("large branch row = %g, want %g", got, want)
	}
	if got, want := document.Frame.Children[1].Row, float64(368); got != want {
		t.Errorf("small branch row = %g, want %g", got, want)
	}
	if children[0].Row != 0 || children[1].Row != 0 {
		t.Fatal("NewXALDocument mutated caller-owned elements")
	}
}

func TestNewXALDocumentKeepsDefaultHeightForCompactDiagram(t *testing.T) {
	t.Parallel()

	document := xaligoentity.NewXALDocument(xaligoentity.DiagramOptions{FrameID: "main"}, []xaligoentity.XALElement{{
		Tag: "rectangle",
		ID:  "service",
	}})

	if got, want := document.Frame.Height, 720; got != want {
		t.Fatalf("compact frame height = %d, want %d", got, want)
	}
	if got := document.Frame.Children[0].Row; got != 0 {
		t.Fatalf("single child row = %g, want omitted", got)
	}
}

func TestNewXALDocumentUsesPaperAndTwelveColumnGrid(t *testing.T) {
	t.Parallel()

	document := xaligoentity.NewXALDocument(xaligoentity.DiagramOptions{
		FrameID: "main", PaperSize: "A4", Orientation: "landscape", GridColumns: 3, GridGap: 24,
	}, rectangles(4, "service"))
	if document.Frame.Width != 1123 || document.Frame.Height != 794 {
		t.Fatalf("A4 landscape frame = %dx%d, want 1123x794", document.Frame.Width, document.Frame.Height)
	}
	if got := len(document.Frame.Children); got != 2 {
		t.Fatalf("grid rows = %d, want 2", got)
	}
	if row := document.Frame.Children[0]; row.Tag != "row" || row.Gap != 24 || len(row.Children) != 3 {
		t.Fatalf("first grid row = %#v", row)
	}
	if col := document.Frame.Children[0].Children[0]; col.Tag != "col" || col.Span != 4 {
		t.Fatalf("first grid column = %#v", col)
	}
}

func TestNewXALDocumentExpandsPaperCanvasWithoutChangingAspectRatio(t *testing.T) {
	t.Parallel()

	document := xaligoentity.NewXALDocument(xaligoentity.DiagramOptions{
		FrameID: "main", Title: "Large", PaperSize: "A4", Orientation: "landscape",
	}, []xaligoentity.XALElement{
		{Tag: "generic-group", ID: "large", Children: rectangles(8, "large")},
		{Tag: "generic-group", ID: "small", Children: rectangles(4, "small")},
	})
	if document.Frame.Height != 1104 || document.Frame.Width != 1562 {
		t.Fatalf("expanded A4 landscape frame = %dx%d, want 1562x1104", document.Frame.Width, document.Frame.Height)
	}
}

func rectangles(count int, prefix string) []xaligoentity.XALElement {
	result := make([]xaligoentity.XALElement, count)
	for index := range result {
		result[index] = xaligoentity.XALElement{
			Tag: "rectangle",
			ID:  prefix + "-node-" + string(rune('a'+index)),
		}
	}
	return result
}
