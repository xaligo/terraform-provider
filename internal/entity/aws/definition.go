package aws

import "sort"

//go:generate go run ../../../tools/awsdefinitions generate -repo ../../..

// DefinitionKind identifies the Terraform protocol surface where an AWS
// Provider definition is registered.
type DefinitionKind string

const (
	DefinitionKindAction            DefinitionKind = "action"
	DefinitionKindDataSource        DefinitionKind = "data_source"
	DefinitionKindEphemeralResource DefinitionKind = "ephemeral_resource"
	DefinitionKindFunction          DefinitionKind = "function"
	DefinitionKindListResource      DefinitionKind = "list_resource"
	DefinitionKindResource          DefinitionKind = "resource"
)

// ProviderDefinition is one definition exported by the pinned HashiCorp AWS
// Provider release. Service is the owning internal service package; provider
// functions use "provider".
type ProviderDefinition struct {
	Kind     DefinitionKind `json:"kind"`
	Service  string         `json:"service"`
	TypeName string         `json:"type_name"`
}

// ProviderDefinitionSnapshot records the provenance and complete definition
// inventory used to generate this package.
type ProviderDefinitionSnapshot struct {
	Source      string               `json:"source"`
	Version     string               `json:"version"`
	Commit      string               `json:"commit"`
	License     string               `json:"license"`
	Definitions []ProviderDefinition `json:"definitions"`
}

type providerDefinitionGroup struct {
	Kind      DefinitionKind
	Service   string
	TypeNames []string
}

var providerDefinitions = func() []ProviderDefinition {
	definitions := make([]ProviderDefinition, 0, AWSProviderDefinitionCount)
	for _, group := range providerDefinitionGroups {
		for _, typeName := range group.TypeNames {
			definitions = append(definitions, ProviderDefinition{
				Kind:     group.Kind,
				Service:  group.Service,
				TypeName: typeName,
			})
		}
	}
	sort.Slice(definitions, func(left, right int) bool {
		return definitionKey(definitions[left].Kind, definitions[left].TypeName) <
			definitionKey(definitions[right].Kind, definitions[right].TypeName)
	})
	return definitions
}()

var providerDefinitionIndex = func() map[string]int {
	index := make(map[string]int, len(providerDefinitions))
	for position, definition := range providerDefinitions {
		index[definitionKey(definition.Kind, definition.TypeName)] = position
	}
	return index
}()

// Definitions returns a defensive copy of the definitions exported by the
// pinned AWS Provider release.
func Definitions() []ProviderDefinition {
	definitions := make([]ProviderDefinition, len(providerDefinitions))
	copy(definitions, providerDefinitions)
	return definitions
}

// DefinitionSnapshot returns the complete pinned inventory and its upstream
// provenance.
func DefinitionSnapshot() ProviderDefinitionSnapshot {
	return ProviderDefinitionSnapshot{
		Source:      AWSProviderSource,
		Version:     AWSProviderVersion,
		Commit:      AWSProviderCommit,
		License:     AWSProviderLicense,
		Definitions: Definitions(),
	}
}

// LookupDefinition reports whether the exact kind and type name are exported
// by the pinned AWS Provider release.
func LookupDefinition(kind DefinitionKind, typeName string) (ProviderDefinition, bool) {
	position, ok := providerDefinitionIndex[definitionKey(kind, typeName)]
	if !ok {
		return ProviderDefinition{}, false
	}
	return providerDefinitions[position], true
}

func definitionKey(kind DefinitionKind, typeName string) string {
	return string(kind) + "\x00" + typeName
}
