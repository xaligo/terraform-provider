package repository_test

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
	"github.com/xaligo/terraform-provider/internal/repository"
)

func TestXALMarshalerProducesCanonicalEscapedXML(t *testing.T) {
	t.Parallel()

	document := xaligoentity.NewXALDocument(xaligoentity.DiagramOptions{FrameID: "main", Title: `A & "B"`}, []xaligoentity.XALElement{{
		Tag:   "rectangle",
		ID:    "node-1",
		Title: `<node & "service">`,
	}})
	content, err := repository.NewXaligoRepository().Marshal(document)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.HasSuffix(string(content), "\n") {
		t.Error("Marshal() output does not end with a newline")
	}
	if !strings.Contains(string(content), `<rectangle id="node-1" title="&lt;node &amp; &#34;service&#34;&gt;"></rectangle>`) {
		t.Fatalf("Marshal() output is not canonical or escaped:\n%s", content)
	}
	var decoded any
	if err := xml.Unmarshal(content, &decoded); err != nil {
		t.Fatalf("generated XML is not well formed: %v", err)
	}
}

func TestXALMarshalerRejectsDuplicateComponentIdentities(t *testing.T) {
	t.Parallel()

	document := xaligoentity.NewXALDocument(xaligoentity.DiagramOptions{FrameID: "main"}, []xaligoentity.XALElement{
		{Tag: "rectangle", ID: "duplicate"},
		{Tag: "generic-group", ID: "duplicate"},
	})
	if _, err := repository.NewXaligoRepository().Marshal(document); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("Marshal() error = %v, want duplicate identity error", err)
	}
}

func TestXALMarshalerSerializesAdaptiveRowWeight(t *testing.T) {
	t.Parallel()

	document := xaligoentity.XALDocument{Frame: xaligoentity.XALFrame{
		ID:       "main",
		Width:    1280,
		Height:   720,
		ItemSize: 32,
		Children: []xaligoentity.XALElement{{
			Tag: "rectangle",
			ID:  "weighted",
			Row: 288,
		}},
	}}
	content, err := repository.NewXaligoRepository().Marshal(document)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(content), `<rectangle id="weighted" row="288"></rectangle>`) {
		t.Fatalf("Marshal() output does not contain row weight:\n%s", content)
	}
}

func TestXALMarshalerSerializesGridLayout(t *testing.T) {
	t.Parallel()

	document := xaligoentity.NewXALDocument(xaligoentity.DiagramOptions{
		FrameID: "main", PaperSize: "A4", Orientation: "landscape", GridColumns: 2, GridGap: 20,
	}, []xaligoentity.XALElement{{Tag: "rectangle", ID: "left"}, {Tag: "rectangle", ID: "right"}})
	content, err := repository.NewXaligoRepository().Marshal(document)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	output := string(content)
	for _, expected := range []string{`<frame id="main" width="1123" height="794" item-size="32">`, `<row gap="20">`, `<col span="6">`} {
		if !strings.Contains(output, expected) {
			t.Fatalf("Marshal() output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestXALMarshalerSerializesEveryLayoutTagAndAttribute(t *testing.T) {
	t.Parallel()

	document := xaligoentity.XALDocument{Frame: xaligoentity.XALFrame{ID: "main", Width: 1280, Height: 720, ItemSize: 32, Children: []xaligoentity.XALElement{{
		Tag: "container", ID: "layout", Layout: "horizontal", Class: "pa-2", Align: "middle-spread", Overflow: "visible", Gap: 0, GapSet: true,
		ContentWidth: 600, ContentHeight: 240, Width: 640, Height: 320, Row: 2, Col: 3,
		Children: []xaligoentity.XALElement{{Tag: "row", Gap: 12, GapSet: true, Overflow: "visible", Children: []xaligoentity.XALElement{{
			Tag: "col", Span: 7.5, Class: "pa-1", Children: []xaligoentity.XALElement{{Tag: "rectangle", ID: "node"}},
		}}}},
	}}}}
	content, err := repository.NewXaligoRepository().Marshal(document)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	output := string(content)
	for _, expected := range []string{
		`<container id="layout" row="2" col="3" gap="0" layout="horizontal" class="pa-2" align="middle-spread" overflow="visible" content-width="600" content-height="240" width="640" height="320">`,
		`<row gap="12" overflow="visible">`, `<col span="7.5" class="pa-1">`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("Marshal() output does not contain %q:\n%s", expected, output)
		}
	}
	binary := strings.TrimSpace(os.Getenv("XALIGO_BIN"))
	if binary == "" {
		return
	}
	inputPath := filepath.Join(t.TempDir(), "layout.xal")
	outputPath := filepath.Join(t.TempDir(), "layout.svg")
	if err := os.WriteFile(inputPath, content, 0o600); err != nil {
		t.Fatalf("write layout XAL: %v", err)
	}
	if output, err := exec.Command(binary, "render", inputPath, "--output", outputPath).CombinedOutput(); err != nil {
		t.Fatalf("xaligo render error = %v\n%s", err, output)
	}
}
