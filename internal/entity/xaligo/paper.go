package xaligo

import "strings"

var paperDimensions = map[string][2]int{
	"a5":      {559, 794},
	"a4":      {794, 1123},
	"a3":      {1123, 1587},
	"a2":      {1587, 2245},
	"a1":      {2245, 3179},
	"letter":  {816, 1056},
	"legal":   {816, 1344},
	"tabloid": {1056, 1632},
}

func PaperDimensions(paperSize, orientation string) (int, int, bool) {
	size, ok := paperDimensions[strings.ToLower(strings.TrimSpace(paperSize))]
	if !ok {
		return 0, 0, false
	}
	width, height := size[0], size[1]
	if strings.EqualFold(orientation, "landscape") {
		width, height = height, width
	}
	return width, height, true
}

func ValidPaperSize(value string) bool {
	if strings.EqualFold(value, "screen") {
		return true
	}
	_, ok := paperDimensions[strings.ToLower(strings.TrimSpace(value))]
	return ok
}

func ValidOrientation(value string) bool {
	return strings.EqualFold(value, "portrait") || strings.EqualFold(value, "landscape")
}

func ValidGridColumns(value int) bool {
	return value == 1 || value == 2 || value == 3 || value == 4 || value == 6 || value == 12
}
