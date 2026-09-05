package common

import (
	"fmt"
	"sort"
)

type NodeKind string

const (
	NodeResource NodeKind = "resource"
	NodeData     NodeKind = "data"
	NodeModule   NodeKind = "module"
)

type Node struct {
	Kind       NodeKind
	Address    string
	Type       string
	Name       string
	Attributes map[string]Attribute
	DependsOn  []string
	Range      SourceRange
}

func (rcvr Node) Attribute(name string) (Attribute, bool) {
	attribute, ok := rcvr.Attributes[name]
	return attribute, ok
}

func (rcvr Node) String(name string) (string, bool) {
	attribute, ok := rcvr.Attribute(name)
	if !ok {
		return "", false
	}
	return attribute.Value.AsString()
}

func (rcvr Node) Bool(name string) (bool, bool) {
	attribute, ok := rcvr.Attribute(name)
	if !ok {
		return false, false
	}
	return attribute.Value.AsBool()
}

func (rcvr Node) ObjectString(attributeName, key string) (string, bool) {
	attribute, ok := rcvr.Attribute(attributeName)
	if !ok {
		return "", false
	}
	return attribute.Value.ObjectString(key)
}

func (rcvr Node) AttributeReferences(name string) []string {
	attribute, ok := rcvr.Attribute(name)
	if !ok {
		return nil
	}
	return append([]string(nil), attribute.References...)
}

type Graph struct {
	Nodes     []Node
	byAddress map[string]Node
}

func (rcvr Graph) Node(address string) (Node, bool) {
	node, ok := rcvr.byAddress[address]
	return node, ok
}

// BuildGraph normalizes parsed Terraform blocks into provider-neutral nodes.
func BuildGraph(config TerraformConfig) (Graph, []Diagnostic) {
	graph := Graph{byAddress: make(map[string]Node, len(config.Blocks))}
	var diagnostics []Diagnostic
	for _, block := range config.Blocks {
		kind := NodeResource
		switch block.Kind {
		case BlockData:
			kind = NodeData
		case BlockModule:
			kind = NodeModule
		}
		if _, exists := graph.byAddress[block.Address]; exists {
			diagnostics = append(diagnostics, Diagnostic{
				Code:     "INFRA-E001",
				Severity: SeverityError,
				Summary:  "Duplicate Terraform address",
				Detail:   fmt.Sprintf("Terraform address %q occurs more than once in the selected source directory.", block.Address),
				Range:    block.Range,
			})
			continue
		}
		node := Node{
			Kind:       kind,
			Address:    block.Address,
			Type:       block.Type,
			Name:       block.Name,
			Attributes: block.Attributes,
			DependsOn:  append([]string(nil), block.References...),
			Range:      block.Range,
		}
		graph.Nodes = append(graph.Nodes, node)
		graph.byAddress[node.Address] = node
	}
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].Address < graph.Nodes[j].Address })
	if len(graph.Nodes) == 0 && !HasErrors(diagnostics) {
		diagnostics = append(diagnostics, Diagnostic{
			Code:     "INFRA-E002",
			Severity: SeverityError,
			Summary:  "No infrastructure blocks found",
			Detail:   "The selected source directory contains no resource, data, or module blocks after xaligo blocks are excluded.",
		})
	}
	SortDiagnostics(diagnostics)
	return graph, diagnostics
}
