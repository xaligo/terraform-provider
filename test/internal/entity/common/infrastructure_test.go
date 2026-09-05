package common_test

import (
	"reflect"
	"testing"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

func TestBuildGraphCreatesStableGraphAndRetainsCycles(t *testing.T) {
	t.Parallel()

	config := commonentity.TerraformConfig{Blocks: []commonentity.Block{
		{Kind: commonentity.BlockResource, Type: "test", Name: "z", Address: "test.z", References: []string{"test.a"}},
		{Kind: commonentity.BlockResource, Type: "test", Name: "a", Address: "test.a", References: []string{"test.z"}},
	}}
	graph, diagnostics := commonentity.BuildGraph(config)
	if len(diagnostics) != 0 {
		t.Fatalf("BuildGraph() diagnostics = %#v", diagnostics)
	}
	got := []string{graph.Nodes[0].Address, graph.Nodes[1].Address}
	if want := []string{"test.a", "test.z"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("node order = %v, want %v", got, want)
	}
	if node, ok := graph.Node("test.a"); !ok || !reflect.DeepEqual(node.DependsOn, []string{"test.z"}) {
		t.Fatalf("Graph.Node(test.a) = %#v, %v", node, ok)
	}
}

func TestBuildGraphRejectsDuplicateAddresses(t *testing.T) {
	t.Parallel()

	block := commonentity.Block{Kind: commonentity.BlockResource, Type: "test", Name: "same", Address: "test.same"}
	graph, diagnostics := commonentity.BuildGraph(commonentity.TerraformConfig{Blocks: []commonentity.Block{block, block}})
	if len(graph.Nodes) != 1 {
		t.Fatalf("node count = %d, want 1", len(graph.Nodes))
	}
	if !commonentity.HasErrors(diagnostics) || len(diagnostics) != 1 || diagnostics[0].Code != "INFRA-E001" {
		t.Fatalf("BuildGraph() diagnostics = %#v", diagnostics)
	}
}
