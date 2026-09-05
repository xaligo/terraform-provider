package repository_test

import (
	"strings"
	"testing"

	awsentity "github.com/xaligo/terraform-provider/internal/entity/aws"
	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
	xaligoentity "github.com/xaligo/terraform-provider/internal/entity/xaligo"
	"github.com/xaligo/terraform-provider/internal/repository"
)

// Every managed resource and data source exposed by the pinned AWS Provider
// must have an explicit mapping policy. A neutral rectangle is acceptable when
// no reviewed xaligo catalog icon exists; MAPPING-W002 is not, because it means
// the upstream definition is still unknown to this provider.
func TestMapperRecognizesEveryAWSProviderDiagramDefinition(t *testing.T) {
	t.Parallel()

	mapper := repository.NewAWSRepository()
	missing := make([]string, 0)
	checked := 0
	for _, definition := range awsentity.Definitions() {
		blockKind := commonentity.BlockResource
		address := definition.TypeName + ".fixture"
		switch definition.Kind {
		case awsentity.DefinitionKindResource:
		case awsentity.DefinitionKindDataSource:
			blockKind = commonentity.BlockData
			address = "data." + definition.TypeName + ".fixture"
		default:
			continue
		}
		checked++
		graph, graphDiagnostics := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{{
			Kind:    blockKind,
			Type:    definition.TypeName,
			Name:    "fixture",
			Address: address,
		}}})
		if commonentity.HasErrors(graphDiagnostics) {
			t.Fatalf("build %s %s graph: %#v", definition.Kind, definition.TypeName, graphDiagnostics)
		}
		_, diagnostics := mapper.Map(graph, xaligoentity.DiagramOptions{FrameID: "main"})
		if commonentity.HasErrors(diagnostics) {
			t.Fatalf("map %s %s: %#v", definition.Kind, definition.TypeName, diagnostics)
		}
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == "MAPPING-W002" {
				missing = append(missing, string(definition.Kind)+":"+definition.TypeName)
				break
			}
		}
	}

	if checked != 2394 {
		t.Fatalf("checked diagram definitions = %d, want 2394", checked)
	}
	if len(missing) > 0 {
		const displayLimit = 25
		displayed := missing
		if len(displayed) > displayLimit {
			displayed = displayed[:displayLimit]
		}
		t.Fatalf(
			"%d of %d AWS Provider diagram definitions have no explicit mapping policy; first %d: %s",
			len(missing), checked, len(displayed), strings.Join(displayed, ", "),
		)
	}
}
