package common_test

import (
	"testing"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

func TestExportModeIsExact(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"enable", "disable"} {
		if !commonentity.ExportMode(value).Valid() {
			t.Errorf("ExportMode(%q).Valid() = false", value)
		}
	}
	for _, value := range []string{"", "enabled", "disabled", "ENABLE", " enable"} {
		if commonentity.ExportMode(value).Valid() {
			t.Errorf("ExportMode(%q).Valid() = true", value)
		}
	}
}
