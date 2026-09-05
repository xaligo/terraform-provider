package xaligo

import (
	"fmt"
	"strings"
	"unicode"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

const (
	xalLayoutLeafHeight       = 64
	xalLayoutItemGroupHeight  = 160
	xalLayoutGroupHeight      = 64
	xalLayoutFrameTitleHeight = 32
	xalLayoutSiblingGap       = 16
)

func ApplyExplicitRows(children []XALElement, rows []commonentity.RowSpec, gap int) ([]XALElement, error) {
	return ApplyExplicitLayout(children, nil, rows, nil, gap)
}

func ApplyExplicitLayout(children []XALElement, containers []commonentity.ContainerSpec, rows []commonentity.RowSpec, layouts []commonentity.ItemLayoutSpec, defaultGap int) ([]XALElement, error) {
	result := append([]XALElement(nil), children...)
	items, err := BuildItemMap(children)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	gap := float64(defaultGap)
	if gap == 0 {
		gap = xalLayoutSiblingGap
	}
	for containerIndex, container := range containers {
		if err := validateLayoutStyle(container.Style); err != nil {
			return nil, fmt.Errorf("container %d: %w", containerIndex+1, err)
		}
		keys, err := layoutKeys(container.Items, items, used, fmt.Sprintf("container %d", containerIndex+1))
		if err != nil {
			return nil, err
		}
		var applied bool
		result, applied = applyExplicitContainer(result, keys, container)
		if !applied {
			return nil, fmt.Errorf("container %d items were not found as unique siblings: %s", containerIndex+1, strings.Join(container.Items, ", "))
		}
		items, err = BuildItemMap(result)
		if err != nil {
			return nil, err
		}
	}
	for layoutIndex, layout := range layouts {
		if err := validateLayoutStyle(layout.Style); err != nil {
			return nil, fmt.Errorf("layout %d: %w", layoutIndex+1, err)
		}
		key := normalizeLayoutReference(layout.Item)
		if key == "" {
			return nil, fmt.Errorf("layout %d item must not be empty", layoutIndex+1)
		}
		target, exists := items[key]
		if !exists {
			return nil, fmt.Errorf("layout %d references unknown item %q", layoutIndex+1, layout.Item)
		}
		if err := validateLayoutTarget(target, layout.Style); err != nil {
			return nil, fmt.Errorf("layout %d: %w", layoutIndex+1, err)
		}
		var matches int
		result, matches = applyItemLayout(result, key, layout.Style)
		if matches != 1 {
			return nil, fmt.Errorf("layout %d item %q resolved %d times; want exactly once", layoutIndex+1, layout.Item, matches)
		}
	}
	for rowIndex, row := range rows {
		if row.Overflow != "" && row.Overflow != "error" && row.Overflow != "visible" {
			return nil, fmt.Errorf("row %d overflow must be error or visible", rowIndex+1)
		}
		if len(row.Items) > 0 && len(row.Columns) > 0 {
			return nil, fmt.Errorf("row %d cannot set both items and col blocks", rowIndex+1)
		}
		if len(row.Columns) > 0 {
			var applied bool
			result, applied, err = applyExplicitColumns(result, row, items, used, gap)
			if err != nil {
				return nil, fmt.Errorf("row %d: %w", rowIndex+1, err)
			}
			if !applied {
				return nil, fmt.Errorf("row %d column items were not found as unique siblings", rowIndex+1)
			}
			continue
		}
		if !ValidGridColumns(len(row.Items)) {
			return nil, fmt.Errorf("row %d must contain 1, 2, 3, 4, 6, or 12 items", rowIndex+1)
		}
		keys, keyError := layoutKeys(row.Items, items, used, fmt.Sprintf("row %d", rowIndex+1))
		if keyError != nil {
			return nil, keyError
		}
		var applied bool
		rowGap := row.Gap
		if !row.GapSet {
			rowGap = gap
		}
		result, applied = applyExplicitRow(result, keys, rowGap, true, row.Overflow)
		if !applied {
			return nil, fmt.Errorf("row %d items were not found as unique siblings: %s", rowIndex+1, strings.Join(row.Items, ", "))
		}
	}
	return result, nil
}

func layoutKeys(values []string, items map[string]XALElement, used map[string]bool, owner string) ([]string, error) {
	keys := make([]string, len(values))
	for index, item := range values {
		keys[index] = normalizeLayoutReference(item)
		if keys[index] == "" || used[keys[index]] {
			return nil, fmt.Errorf("%s contains an empty or duplicate item reference %q", owner, item)
		}
		if _, exists := items[keys[index]]; !exists {
			return nil, fmt.Errorf("%s references unknown item %q", owner, item)
		}
		used[keys[index]] = true
	}
	return keys, nil
}

// BuildItemMap exposes every generated semantic element under the same stable
// key space used by Terraform row blocks. Item nodes use their unique name;
// containers use their generated ID.
func BuildItemMap(children []XALElement) (map[string]XALElement, error) {
	items := map[string]XALElement{}
	var visit func([]XALElement) error
	visit = func(elements []XALElement) error {
		for _, element := range elements {
			key := element.ID
			if element.Name != "" {
				key = element.Name
			}
			key = normalizeLayoutReference(key)
			if key != "" && element.Tag != "row" && element.Tag != "col" {
				if _, exists := items[key]; exists {
					return fmt.Errorf("generated items map contains duplicate key %q", key)
				}
				items[key] = element
			}
			if err := visit(element.Children); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(children); err != nil {
		return nil, err
	}
	return items, nil
}

func applyExplicitRow(elements []XALElement, keys []string, gap float64, gapSet bool, overflow string) ([]XALElement, bool) {
	indices := make([]int, len(keys))
	allDirect := true
	for keyIndex, key := range keys {
		indices[keyIndex] = -1
		for elementIndex, element := range elements {
			if layoutElementMatches(element, key) {
				if indices[keyIndex] != -1 {
					return elements, false
				}
				indices[keyIndex] = elementIndex
			}
		}
		allDirect = allDirect && indices[keyIndex] >= 0
	}
	if allDirect {
		selected := map[int]bool{}
		cols := make([]XALElement, len(indices))
		first := len(elements)
		for index, elementIndex := range indices {
			selected[elementIndex] = true
			if elementIndex < first {
				first = elementIndex
			}
			cols[index] = XALElement{Tag: "col", Span: 12 / float64(len(indices)), Children: []XALElement{elements[elementIndex]}}
		}
		result := make([]XALElement, 0, len(elements)-len(indices)+1)
		for index, element := range elements {
			if index == first {
				result = append(result, XALElement{Tag: "row", Gap: gap, GapSet: gapSet, Overflow: overflow, Children: cols})
			}
			if !selected[index] {
				result = append(result, element)
			}
		}
		return result, true
	}
	for index := range elements {
		children, applied := applyExplicitRow(elements[index].Children, keys, gap, gapSet, overflow)
		if applied {
			result := append([]XALElement(nil), elements...)
			result[index].Children = children
			return result, true
		}
	}
	return elements, false
}

func applyExplicitContainer(elements []XALElement, keys []string, spec commonentity.ContainerSpec) ([]XALElement, bool) {
	indices, ok := directLayoutIndices(elements, keys)
	if ok {
		children, first, selected := selectedLayoutElements(elements, indices)
		container := elementWithStyle(XALElement{Tag: "container", ID: normalizeLayoutReference(spec.ID), Children: children}, spec.Style)
		return replaceSelectedElements(elements, first, selected, container), true
	}
	for index := range elements {
		children, applied := applyExplicitContainer(elements[index].Children, keys, spec)
		if applied {
			result := append([]XALElement(nil), elements...)
			result[index].Children = children
			return result, true
		}
	}
	return elements, false
}

func directLayoutIndices(elements []XALElement, keys []string) ([]int, bool) {
	indices := make([]int, len(keys))
	for keyIndex, key := range keys {
		indices[keyIndex] = -1
		for elementIndex, element := range elements {
			if layoutElementMatches(element, key) {
				if indices[keyIndex] != -1 {
					return nil, false
				}
				indices[keyIndex] = elementIndex
			}
		}
		if indices[keyIndex] < 0 {
			return nil, false
		}
	}
	return indices, true
}

func selectedLayoutElements(elements []XALElement, indices []int) ([]XALElement, int, map[int]bool) {
	children := make([]XALElement, len(indices))
	selected := map[int]bool{}
	first := len(elements)
	for index, elementIndex := range indices {
		children[index] = elements[elementIndex]
		selected[elementIndex] = true
		if elementIndex < first {
			first = elementIndex
		}
	}
	return children, first, selected
}

func replaceSelectedElements(elements []XALElement, first int, selected map[int]bool, replacement XALElement) []XALElement {
	result := make([]XALElement, 0, len(elements)-len(selected)+1)
	for index, element := range elements {
		if index == first {
			result = append(result, replacement)
		}
		if !selected[index] {
			result = append(result, element)
		}
	}
	return result
}

func elementWithStyle(element XALElement, style commonentity.LayoutStyle) XALElement {
	element.Layout = style.Layout
	element.Class = style.Class
	element.Align = style.Align
	element.Overflow = style.Overflow
	element.Gap = style.Gap
	element.GapSet = style.GapSet
	element.ContentWidth = style.ContentWidth
	element.ContentHeight = style.ContentHeight
	element.Width = style.Width
	element.Height = style.Height
	element.Row = style.Row
	element.Col = style.Col
	return element
}

func validateLayoutStyle(style commonentity.LayoutStyle) error {
	if style.Layout != "" && style.Layout != "horizontal" && style.Layout != "staggered" {
		return fmt.Errorf("layout must be horizontal or staggered")
	}
	if style.Overflow != "" && style.Overflow != "error" && style.Overflow != "visible" {
		return fmt.Errorf("overflow must be error or visible")
	}
	validAlign := map[string]bool{
		"top-left": true, "top-center": true, "top-right": true, "top-spread": true,
		"middle-left": true, "middle-center": true, "middle-right": true, "middle-spread": true,
		"bottom-left": true, "bottom-center": true, "bottom-right": true, "bottom-spread": true,
	}
	if style.Align != "" && !validAlign[style.Align] {
		return fmt.Errorf("align %q is not supported", style.Align)
	}
	if style.Gap < 0 || style.ContentWidth < 0 || style.ContentHeight < 0 || style.Width < 0 || style.Height < 0 || style.Row < 0 || style.Col < 0 {
		return fmt.Errorf("layout dimensions must not be negative")
	}
	return nil
}

func validateLayoutTarget(element XALElement, style commonentity.LayoutStyle) error {
	usesContainerGeometry := style.Layout != "" || style.Align != "" || style.GapSet || style.ContentWidth > 0 || style.ContentHeight > 0
	if usesContainerGeometry && len(element.Children) == 0 {
		return fmt.Errorf("item %q is a leaf and cannot use container layout attributes", element.ID+element.Name)
	}
	if style.Layout == "staggered" && !supportsStaggeredLayout(element.Tag) {
		return fmt.Errorf("<%s> does not support staggered layout", element.Tag)
	}
	return nil
}

func supportsStaggeredLayout(tag string) bool {
	switch tag {
	case "aws-cloud", "aws-cloud-alt", "region", "availability-zone", "vpc", "public-subnet", "private-subnet",
		"security-group", "auto-scaling-group", "spot-fleet", "server-contents", "corporate-data-center",
		"ec2-instance-contents", "aws-account", "aws-iot-greengrass-deployment", "aws-iot-greengrass",
		"elastic-beanstalk-container", "aws-step-functions-workflow", "generic-group":
		return true
	default:
		return false
	}
}

func applyItemLayout(elements []XALElement, key string, style commonentity.LayoutStyle) ([]XALElement, int) {
	result := append([]XALElement(nil), elements...)
	matches := 0
	for index, element := range result {
		if layoutElementMatches(element, key) {
			result[index] = elementWithStyle(element, style)
			matches++
		}
		children, childMatches := applyItemLayout(result[index].Children, key, style)
		result[index].Children = children
		matches += childMatches
	}
	return result, matches
}

func applyExplicitColumns(elements []XALElement, row commonentity.RowSpec, items map[string]XALElement, used map[string]bool, defaultGap float64) ([]XALElement, bool, error) {
	var allKeys []string
	columnKeys := make([][]string, len(row.Columns))
	totalSpan := 0.0
	unspecifiedSpans := 0
	for index, column := range row.Columns {
		if err := validateLayoutStyle(column.Style); err != nil {
			return elements, false, fmt.Errorf("col %d: %w", index+1, err)
		}
		keys, err := layoutKeys(column.Items, items, used, fmt.Sprintf("col %d", index+1))
		if err != nil {
			return elements, false, err
		}
		columnKeys[index] = keys
		allKeys = append(allKeys, keys...)
		if column.Span > 0 {
			totalSpan += column.Span
		} else {
			unspecifiedSpans++
		}
	}
	if totalSpan > 12 {
		return elements, false, fmt.Errorf("col spans total %.4g; maximum is 12", totalSpan)
	}
	if unspecifiedSpans > 0 && totalSpan >= 12 {
		return elements, false, fmt.Errorf("no grid span remains for %d col blocks without span", unspecifiedSpans)
	}
	defaultSpan := 0.0
	if unspecifiedSpans > 0 {
		defaultSpan = (12 - totalSpan) / float64(unspecifiedSpans)
	}
	return applyColumnRowAtLevel(elements, row, columnKeys, allKeys, defaultGap, defaultSpan)
}

func applyColumnRowAtLevel(elements []XALElement, row commonentity.RowSpec, columnKeys [][]string, allKeys []string, defaultGap, defaultSpan float64) ([]XALElement, bool, error) {
	indices, ok := directLayoutIndices(elements, allKeys)
	if ok {
		_, first, selected := selectedLayoutElements(elements, indices)
		cols := make([]XALElement, len(columnKeys))
		for index, keys := range columnKeys {
			columnIndices, _ := directLayoutIndices(elements, keys)
			children, _, _ := selectedLayoutElements(elements, columnIndices)
			span := row.Columns[index].Span
			if span == 0 {
				span = defaultSpan
			}
			cols[index] = elementWithStyle(XALElement{Tag: "col", Span: span, Children: children}, row.Columns[index].Style)
		}
		gap := row.Gap
		if !row.GapSet {
			gap = defaultGap
		}
		return replaceSelectedElements(elements, first, selected, XALElement{Tag: "row", Gap: gap, GapSet: true, Overflow: row.Overflow, Children: cols}), true, nil
	}
	for index := range elements {
		children, applied, err := applyColumnRowAtLevel(elements[index].Children, row, columnKeys, allKeys, defaultGap, defaultSpan)
		if err != nil {
			return elements, false, err
		}
		if applied {
			result := append([]XALElement(nil), elements...)
			result[index].Children = children
			return result, true, nil
		}
	}
	return elements, false, nil
}

func layoutElementMatches(element XALElement, key string) bool {
	return normalizeLayoutReference(element.ID) == key || normalizeLayoutReference(element.Name) == key
}

func normalizeLayoutReference(value string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), "items.")
	var result strings.Builder
	dash := false
	for _, current := range strings.ToLower(value) {
		switch {
		case unicode.IsLetter(current), unicode.IsDigit(current), current == '_':
			result.WriteRune(current)
			dash = false
		default:
			if !dash && result.Len() > 0 {
				result.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(result.String(), "-")
}

// prepareXALLayout gives each generated vertical branch enough relative space
// for its descendants. XAL keeps row values as ratios, so these measurements
// remain independent of the renderer's final output format.
func prepareXALLayout(title string, children []XALElement) ([]XALElement, int) {
	prepared, heights := prepareXALChildren(children)
	applyXALRowWeights(prepared, heights)

	height := sumXALHeights(heights) + xalLayoutSiblingGap*gapCount(len(prepared))
	if title != "" {
		height += xalLayoutFrameTitleHeight
	}
	if height < XALFrameHeight {
		height = XALFrameHeight
	}
	return prepared, height
}

func prepareXALChildren(children []XALElement) ([]XALElement, []int) {
	prepared := make([]XALElement, len(children))
	heights := make([]int, len(children))
	for index, child := range children {
		prepared[index], heights[index] = prepareXALElement(child)
	}
	return prepared, heights
}

func prepareXALElement(element XALElement) (XALElement, int) {
	prepared := element
	prepared.Row = 0
	if len(element.Children) == 0 {
		prepared.Children = nil
		return prepared, xalLayoutLeafHeight
	}

	var childHeights []int
	prepared.Children, childHeights = prepareXALChildren(element.Children)
	if prepared.Tag == "row" {
		applyXALRowWeights(prepared.Children, childHeights)
		return prepared, maxXALHeight(childHeights)
	}
	if hasOnlyXALItems(prepared.Children) {
		return prepared, xalLayoutItemGroupHeight
	}
	applyXALRowWeights(prepared.Children, childHeights)
	height := xalLayoutGroupHeight + sumXALHeights(childHeights)
	height += xalLayoutSiblingGap * gapCount(len(prepared.Children))
	return prepared, height
}

func applyXALGrid(children []XALElement, columns, gap int) []XALElement {
	prepared := make([]XALElement, len(children))
	for index, child := range children {
		prepared[index] = child
		prepared[index].Children = applyXALGrid(child.Children, columns, gap)
	}
	if !ValidGridColumns(columns) || columns == 1 || len(prepared) < 2 || hasOnlyXALItems(prepared) || !hasOnlyXALLeaves(prepared) {
		return prepared
	}
	span := 12 / float64(columns)
	rows := make([]XALElement, 0, (len(prepared)+columns-1)/columns)
	for start := 0; start < len(prepared); start += columns {
		end := start + columns
		if end > len(prepared) {
			end = len(prepared)
		}
		cols := make([]XALElement, 0, end-start)
		for _, child := range prepared[start:end] {
			cols = append(cols, XALElement{Tag: "col", Span: span, Children: []XALElement{child}})
		}
		rows = append(rows, XALElement{Tag: "row", Gap: float64(gap), GapSet: true, Children: cols})
	}
	return rows
}

func hasOnlyXALLeaves(elements []XALElement) bool {
	for _, element := range elements {
		if len(element.Children) != 0 {
			return false
		}
	}
	return true
}

func applyXALRowWeights(elements []XALElement, heights []int) {
	if len(elements) < 2 || len(elements) != len(heights) {
		return
	}
	equal := true
	for index := 1; index < len(heights); index++ {
		if heights[index] != heights[0] {
			equal = false
			break
		}
	}
	if equal {
		return
	}
	for index := range elements {
		elements[index].Row = float64(heights[index])
	}
}

func hasOnlyXALItems(elements []XALElement) bool {
	if len(elements) == 0 {
		return false
	}
	for _, element := range elements {
		if element.Tag != "item" {
			return false
		}
	}
	return true
}

func sumXALHeights(heights []int) int {
	total := 0
	for _, height := range heights {
		total += height
	}
	return total
}

func maxXALHeight(heights []int) int {
	maximum := 0
	for _, height := range heights {
		if height > maximum {
			maximum = height
		}
	}
	return maximum
}

func gapCount(count int) int {
	if count <= 1 {
		return 0
	}
	return count - 1
}
