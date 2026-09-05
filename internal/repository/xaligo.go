package repository

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"

	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
)

type XaligoRepository interface {
	Marshal(document xaligoentity.XALDocument) ([]byte, error)
}

type xaligoRepository struct{}

func NewXaligoRepository() XaligoRepository {
	return &xaligoRepository{}
}

func (rcvr *xaligoRepository) Marshal(document xaligoentity.XALDocument) ([]byte, error) {
	if err := xaligoentity.ValidateXAL(document); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	encoder.Indent("", "  ")

	root := xml.StartElement{Name: xml.Name{Local: "xaligo"}, Attr: []xml.Attr{attribute("version", xaligoentity.XALVersion)}}
	if err := encoder.EncodeToken(root); err != nil {
		return nil, fmt.Errorf("encode xaligo start: %w", err)
	}
	frames := xml.StartElement{Name: xml.Name{Local: "frames"}}
	if err := encoder.EncodeToken(frames); err != nil {
		return nil, fmt.Errorf("encode frames start: %w", err)
	}

	frame := document.Frame
	frameAttributes := []xml.Attr{attribute("id", frame.ID)}
	if frame.Title != "" {
		frameAttributes = append(frameAttributes, attribute("title", frame.Title))
	}
	frameAttributes = append(frameAttributes,
		attribute("width", strconv.Itoa(frame.Width)),
		attribute("height", strconv.Itoa(frame.Height)),
		attribute("item-size", strconv.Itoa(frame.ItemSize)),
	)
	frameStart := xml.StartElement{Name: xml.Name{Local: "frame"}, Attr: frameAttributes}
	if err := encoder.EncodeToken(frameStart); err != nil {
		return nil, fmt.Errorf("encode frame start: %w", err)
	}
	for _, child := range frame.Children {
		if err := encodeElement(encoder, child); err != nil {
			return nil, err
		}
	}
	if err := encoder.EncodeToken(frameStart.End()); err != nil {
		return nil, fmt.Errorf("encode frame end: %w", err)
	}
	if err := encoder.EncodeToken(frames.End()); err != nil {
		return nil, fmt.Errorf("encode frames end: %w", err)
	}
	if err := encoder.EncodeToken(root.End()); err != nil {
		return nil, fmt.Errorf("encode xaligo end: %w", err)
	}
	if err := encoder.Flush(); err != nil {
		return nil, fmt.Errorf("flush XAL document: %w", err)
	}
	output.WriteByte('\n')
	return output.Bytes(), nil
}

func encodeElement(encoder *xml.Encoder, element xaligoentity.XALElement) error {
	attributes := make([]xml.Attr, 0, 5)
	if element.ID != "" {
		attributes = append(attributes, attribute("id", element.ID))
	}
	if element.Title != "" {
		attributes = append(attributes, attribute("title", element.Title))
	}
	if element.Name != "" {
		attributes = append(attributes, attribute("name", element.Name))
	}
	if element.IconID != "" {
		attributes = append(attributes, attribute("icon-id", element.IconID))
	}
	if element.Row > 0 {
		attributes = append(attributes, attribute("row", formatNumber(element.Row)))
	}
	if element.Col > 0 {
		attributes = append(attributes, attribute("col", formatNumber(element.Col)))
	}
	if element.Span > 0 {
		attributes = append(attributes, attribute("span", formatNumber(element.Span)))
	}
	if element.GapSet || element.Gap > 0 {
		attributes = append(attributes, attribute("gap", formatNumber(element.Gap)))
	}
	attributes = appendOptionalAttribute(attributes, "layout", element.Layout)
	attributes = appendOptionalAttribute(attributes, "class", element.Class)
	attributes = appendOptionalAttribute(attributes, "align", element.Align)
	attributes = appendOptionalAttribute(attributes, "overflow", element.Overflow)
	attributes = appendPositiveNumberAttribute(attributes, "content-width", element.ContentWidth)
	attributes = appendPositiveNumberAttribute(attributes, "content-height", element.ContentHeight)
	attributes = appendPositiveNumberAttribute(attributes, "width", element.Width)
	attributes = appendPositiveNumberAttribute(attributes, "height", element.Height)
	start := xml.StartElement{Name: xml.Name{Local: element.Tag}, Attr: attributes}
	if err := encoder.EncodeToken(start); err != nil {
		return fmt.Errorf("encode <%s> start: %w", element.Tag, err)
	}
	for _, child := range element.Children {
		if err := encodeElement(encoder, child); err != nil {
			return err
		}
	}
	if err := encoder.EncodeToken(start.End()); err != nil {
		return fmt.Errorf("encode <%s> end: %w", element.Tag, err)
	}
	return nil
}

func appendOptionalAttribute(attributes []xml.Attr, name, value string) []xml.Attr {
	if value != "" {
		return append(attributes, attribute(name, value))
	}
	return attributes
}

func appendPositiveNumberAttribute(attributes []xml.Attr, name string, value float64) []xml.Attr {
	if value > 0 {
		return append(attributes, attribute(name, formatNumber(value)))
	}
	return attributes
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func attribute(name, value string) xml.Attr {
	return xml.Attr{Name: xml.Name{Local: name}, Value: value}
}
