package common

import (
	"fmt"
	"sort"
	"strings"
)

// Severity is independent from HCL and Terraform Plugin Framework severities.
type Severity uint8

const (
	SeverityWarning Severity = iota + 1
	SeverityError
)

func (rcvr Severity) String() string {
	if rcvr == SeverityError {
		return "error"
	}
	return "warning"
}

// SourceRange identifies a source location without leaking parser types into
// the application and presentation layers.
type SourceRange struct {
	Filename    string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
}

func (rcvr SourceRange) String() string {
	if rcvr.Filename == "" {
		return ""
	}
	if rcvr.StartLine <= 0 {
		return rcvr.Filename
	}
	return fmt.Sprintf("%s:%d:%d", rcvr.Filename, rcvr.StartLine, rcvr.StartColumn)
}

// Diagnostic is the portable diagnostic shared by every delivery adapter.
type Diagnostic struct {
	Code     string
	Severity Severity
	Summary  string
	Detail   string
	Range    SourceRange
}

func (rcvr Diagnostic) Message() string {
	parts := make([]string, 0, 3)
	if location := rcvr.Range.String(); location != "" {
		parts = append(parts, location)
	}
	if rcvr.Code != "" {
		parts = append(parts, rcvr.Code)
	}
	parts = append(parts, rcvr.Summary)
	message := strings.Join(parts, ": ")
	if rcvr.Detail != "" && rcvr.Detail != rcvr.Summary {
		message += ": " + rcvr.Detail
	}
	return message
}

func SortDiagnostics(values []Diagnostic) {
	sort.SliceStable(values, func(i, j int) bool {
		left, right := values[i], values[j]
		if left.Range.Filename != right.Range.Filename {
			return left.Range.Filename < right.Range.Filename
		}
		if left.Range.StartLine != right.Range.StartLine {
			return left.Range.StartLine < right.Range.StartLine
		}
		if left.Range.StartColumn != right.Range.StartColumn {
			return left.Range.StartColumn < right.Range.StartColumn
		}
		if left.Severity != right.Severity {
			return left.Severity > right.Severity
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Summary != right.Summary {
			return left.Summary < right.Summary
		}
		return left.Detail < right.Detail
	})
}

func HasErrors(values []Diagnostic) bool {
	for _, value := range values {
		if value.Severity == SeverityError {
			return true
		}
	}
	return false
}

func HasWarnings(values []Diagnostic) bool {
	for _, value := range values {
		if value.Severity == SeverityWarning {
			return true
		}
	}
	return false
}

func PromoteWarnings(values []Diagnostic) {
	for i := range values {
		if values[i].Severity == SeverityWarning {
			values[i].Severity = SeverityError
			values[i].Detail = strings.TrimSpace(values[i].Detail + " Warning promoted by fail_on_warning.")
		}
	}
}
