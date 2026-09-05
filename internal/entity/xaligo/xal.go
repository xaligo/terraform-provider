package xaligo

import (
	"fmt"
	"strconv"
	"strings"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

const (
	XALVersion     = "1"
	XALFrameWidth  = 1280
	XALFrameHeight = 720
	XALItemSize    = 32
)

type DiagramOptions struct {
	FrameID     string
	Title       string
	PaperSize   string
	Orientation string
	GridColumns int
	GridGap     int
	Rows        []commonentity.RowSpec
	Containers  []commonentity.ContainerSpec
	Layouts     []commonentity.ItemLayoutSpec
}

type XALDocument struct {
	Frame XALFrame
}

type XALFrame struct {
	ID       string
	Title    string
	Width    int
	Height   int
	ItemSize int
	Children []XALElement
}

type XALElement struct {
	Tag           string
	ID            string
	Title         string
	Name          string
	IconID        string
	Row           float64
	Col           float64
	Span          float64
	Gap           float64
	GapSet        bool
	Layout        string
	Class         string
	Align         string
	Overflow      string
	ContentWidth  float64
	ContentHeight float64
	Width         float64
	Height        float64
	Children      []XALElement
}

func NewXALDocument(options DiagramOptions, children []XALElement) XALDocument {
	children = applyXALGrid(children, options.GridColumns, options.GridGap)
	children, requiredHeight := prepareXALLayout(options.Title, children)
	width := XALFrameWidth
	height := requiredHeight
	if paperWidth, paperHeight, ok := PaperDimensions(options.PaperSize, options.Orientation); ok {
		width, height = paperWidth, paperHeight
		if requiredHeight > height {
			width = (width*requiredHeight + height - 1) / height
			height = requiredHeight
		}
	}
	return XALDocument{Frame: XALFrame{
		ID:       options.FrameID,
		Title:    options.Title,
		Width:    width,
		Height:   height,
		ItemSize: XALItemSize,
		Children: children,
	}}
}

func ValidateXAL(document XALDocument) error {
	if strings.TrimSpace(document.Frame.ID) == "" {
		return fmt.Errorf("frame id must not be empty")
	}
	if document.Frame.Width <= 0 || document.Frame.Height <= 0 || document.Frame.ItemSize <= 0 {
		return fmt.Errorf("frame geometry and item size must be positive")
	}
	seenComponents := map[string]string{document.Frame.ID: "frame"}
	seenItems := map[string]string{}
	for _, child := range document.Frame.Children {
		if err := validateXALElement(child, seenComponents, seenItems); err != nil {
			return err
		}
	}
	return nil
}

func validateXALElement(element XALElement, seenComponents, seenItems map[string]string) error {
	allowed := map[string]bool{
		"container": true, "row": true, "col": true,
		"aws-cloud": true, "aws-cloud-alt": true, "region": true,
		"availability-zone": true, "vpc": true,
		"public-subnet": true, "private-subnet": true,
		"security-group": true, "auto-scaling-group": true, "spot-fleet": true,
		"server-contents": true, "corporate-data-center": true, "ec2-instance-contents": true,
		"aws-account": true, "aws-iot-greengrass-deployment": true,
		"aws-iot-greengrass": true, "elastic-beanstalk-container": true,
		"aws-step-functions-workflow": true, "generic-group": true,
		"capture": true, "rectangle": true, "item": true,
	}
	if !allowed[element.Tag] {
		return fmt.Errorf("unsupported generated XAL tag %q", element.Tag)
	}
	if element.Row < 0 || element.Col < 0 || element.Gap < 0 || element.Span < 0 || element.Span > 12 || element.ContentWidth < 0 || element.ContentHeight < 0 || element.Width < 0 || element.Height < 0 {
		return fmt.Errorf("generated <%s> row must be positive when present", element.Tag)
	}
	if element.Tag == "col" && element.Span == 0 {
		return fmt.Errorf("generated <col> span must be positive")
	}
	if element.Tag == "item" {
		catalogID, err := strconv.ParseInt(element.ID, 10, 32)
		if err != nil || catalogID <= 0 {
			return fmt.Errorf("item catalog id %q is not a positive int32", element.ID)
		}
		if strings.TrimSpace(element.Name) == "" {
			return fmt.Errorf("mapped item %q requires a stable name", element.ID)
		}
		if previous, exists := seenItems[element.Name]; exists {
			return fmt.Errorf("item name %q is used by both %s and %s", element.Name, previous, element.Title)
		}
		seenItems[element.Name] = element.Title
	} else if element.Tag != "row" && element.Tag != "col" {
		if strings.TrimSpace(element.ID) == "" {
			return fmt.Errorf("generated <%s> requires an id", element.Tag)
		}
		if previous, exists := seenComponents[element.ID]; exists {
			return fmt.Errorf("component id %q is used by both %s and %s", element.ID, previous, element.Tag)
		}
		seenComponents[element.ID] = element.Tag
	}
	for _, child := range element.Children {
		if err := validateXALElement(child, seenComponents, seenItems); err != nil {
			return err
		}
	}
	return nil
}
