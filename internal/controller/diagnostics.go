package controller

import (
	"errors"
	"strings"

	frameworkdiag "github.com/hashicorp/terraform-plugin-framework/diag"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	"github.com/xaligo/terraform-provider/internal/usecase"
)

func appendDomainDiagnostics(target *frameworkdiag.Diagnostics, values []commonentity.Diagnostic) {
	for _, value := range values {
		summary := value.Summary
		if value.Code != "" {
			summary = value.Code + ": " + summary
		}
		detail := value.Detail
		if location := value.Range.String(); location != "" {
			detail = strings.TrimSpace(location + ": " + detail)
		}
		if value.Severity == commonentity.SeverityError {
			target.AddError(summary, detail)
		} else {
			target.AddWarning(summary, detail)
		}
	}
}

func appendUsecaseError(target *frameworkdiag.Diagnostics, err error) {
	if err == nil || errors.Is(err, usecase.ErrConversion) {
		return
	}
	var failure *commonentity.Failure
	if errors.As(err, &failure) {
		detail := ""
		if failure.Err != nil {
			detail = failure.Err.Error()
		}
		target.AddError(failure.Summary, detail)
		return
	}
	target.AddError("Diagram operation failed", err.Error())
}
