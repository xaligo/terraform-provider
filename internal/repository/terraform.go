package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	commonentity "github.com/xaligo/terraform-provider/internal/entity/common"
)

type TerraformRepository interface {
	Load(directory string) (commonentity.TerraformConfig, []commonentity.Diagnostic, error)
}

type terraformRepository struct{}

func NewTerraformRepository() TerraformRepository {
	return &terraformRepository{}
}

// Load parses regular .tf files directly inside directory in lexical order.
func (rcvr *terraformRepository) Load(directory string) (commonentity.TerraformConfig, []commonentity.Diagnostic, error) {
	abs, err := filepath.Abs(directory)
	if err != nil {
		return commonentity.TerraformConfig{}, nil, fmt.Errorf("resolve source directory: %w", err)
	}
	abs = filepath.Clean(abs)
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return commonentity.TerraformConfig{}, nil, fmt.Errorf("resolve source directory symlinks: %w", err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return commonentity.TerraformConfig{}, nil, fmt.Errorf("stat source directory: %w", err)
	}
	if !info.IsDir() {
		return commonentity.TerraformConfig{}, nil, fmt.Errorf("source path is not a directory: %s", real)
	}

	entries, err := os.ReadDir(real)
	if err != nil {
		return commonentity.TerraformConfig{}, nil, fmt.Errorf("read source directory: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	config := commonentity.TerraformConfig{Directory: filepath.Clean(real)}
	var diagnostics []commonentity.Diagnostic
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".tf" {
			continue
		}
		entryInfo, infoErr := entry.Info()
		if infoErr != nil {
			return commonentity.TerraformConfig{}, diagnostics, fmt.Errorf("inspect Terraform source %s: %w", entry.Name(), infoErr)
		}
		if !entryInfo.Mode().IsRegular() {
			continue
		}

		data, readErr := os.ReadFile(filepath.Join(real, entry.Name()))
		if readErr != nil {
			return commonentity.TerraformConfig{}, diagnostics, fmt.Errorf("read Terraform source %s: %w", entry.Name(), readErr)
		}
		config.Files = append(config.Files, commonentity.SourceFile{Path: entry.Name(), Bytes: data})
		if !utf8.Valid(data) {
			diagnostics = append(diagnostics, commonentity.Diagnostic{
				Code:     "TFCONFIG-E001",
				Severity: commonentity.SeverityError,
				Summary:  "Terraform source is not valid UTF-8",
				Detail:   "Only UTF-8 .tf source files are supported.",
				Range:    commonentity.SourceRange{Filename: entry.Name(), StartLine: 1, StartColumn: 1},
			})
			continue
		}

		file, parseDiagnostics := hclsyntax.ParseConfig(data, entry.Name(), hcl.Pos{Line: 1, Column: 1})
		diagnostics = append(diagnostics, convertHCLDiagnostics(parseDiagnostics)...)
		if file == nil || parseDiagnostics.HasErrors() {
			continue
		}
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			return commonentity.TerraformConfig{}, diagnostics, fmt.Errorf("unexpected HCL body type for %s", entry.Name())
		}
		blocks, blockDiagnostics := decodeBlocks(body, entry.Name())
		config.Blocks = append(config.Blocks, blocks...)
		diagnostics = append(diagnostics, blockDiagnostics...)
	}

	if len(config.Files) == 0 {
		return commonentity.TerraformConfig{}, diagnostics, fmt.Errorf("source directory contains no regular .tf files: %s", real)
	}
	sort.Slice(config.Blocks, func(i, j int) bool { return config.Blocks[i].Address < config.Blocks[j].Address })
	commonentity.SortDiagnostics(diagnostics)
	return config, diagnostics, nil
}

func decodeBlocks(body *hclsyntax.Body, filename string) ([]commonentity.Block, []commonentity.Diagnostic) {
	var blocks []commonentity.Block
	var diagnostics []commonentity.Diagnostic
	for _, sourceBlock := range body.Blocks {
		kind, resourceType, name, address, include, err := blockIdentity(sourceBlock)
		if err != nil {
			diagnostics = append(diagnostics, commonentity.Diagnostic{
				Code:     "TFCONFIG-E002",
				Severity: commonentity.SeverityError,
				Summary:  "Invalid Terraform block labels",
				Detail:   err.Error(),
				Range:    sourceRange(filename, sourceBlock.TypeRange),
			})
			continue
		}
		if !include {
			continue
		}

		attributes := make(map[string]commonentity.Attribute, len(sourceBlock.Body.Attributes))
		allReferences := make([]string, 0)
		attributeNames := make([]string, 0, len(sourceBlock.Body.Attributes))
		for attributeName := range sourceBlock.Body.Attributes {
			attributeNames = append(attributeNames, attributeName)
		}
		sort.Strings(attributeNames)
		for _, attributeName := range attributeNames {
			attribute := sourceBlock.Body.Attributes[attributeName]
			references := expressionReferences(attribute.Expr)
			attributes[attributeName] = commonentity.Attribute{
				Name:       attributeName,
				Value:      constantValue(attribute.Expr),
				References: references,
				Range:      sourceRange(filename, attribute.Range()),
			}
			allReferences = append(allReferences, references...)
			if attributeName == "count" || attributeName == "for_each" {
				diagnostics = append(diagnostics, commonentity.Diagnostic{
					Code:     "TFCONFIG-W001",
					Severity: commonentity.SeverityWarning,
					Summary:  "Resource expansion is not evaluated",
					Detail:   fmt.Sprintf("%s is represented once because %s expansion is outside the initial converter scope.", address, attributeName),
					Range:    sourceRange(filename, attribute.Range()),
				})
			}
		}

		nestedTypes, nestedReferences, nestedDiagnostics := inspectNestedBlocks(sourceBlock.Body, filename, address)
		allReferences = append(allReferences, nestedReferences...)
		diagnostics = append(diagnostics, nestedDiagnostics...)
		blocks = append(blocks, commonentity.Block{
			Kind:         kind,
			Type:         resourceType,
			Name:         name,
			Address:      address,
			Attributes:   attributes,
			References:   sortedUnique(allReferences),
			Range:        sourceRange(filename, sourceBlock.TypeRange),
			NestedBlocks: nestedTypes,
		})
	}
	return blocks, diagnostics
}

func blockIdentity(block *hclsyntax.Block) (commonentity.BlockKind, string, string, string, bool, error) {
	switch block.Type {
	case "resource":
		if len(block.Labels) != 2 {
			return "", "", "", "", false, fmt.Errorf("resource block requires type and name labels")
		}
		if strings.HasPrefix(block.Labels[0], "xaligo_") {
			return "", "", "", "", false, nil
		}
		return commonentity.BlockResource, block.Labels[0], block.Labels[1], block.Labels[0] + "." + block.Labels[1], true, nil
	case "data":
		if len(block.Labels) != 2 {
			return "", "", "", "", false, fmt.Errorf("data block requires type and name labels")
		}
		if strings.HasPrefix(block.Labels[0], "xaligo_") {
			return "", "", "", "", false, nil
		}
		return commonentity.BlockData, block.Labels[0], block.Labels[1], "data." + block.Labels[0] + "." + block.Labels[1], true, nil
	case "module":
		if len(block.Labels) != 1 {
			return "", "", "", "", false, fmt.Errorf("module block requires a name label")
		}
		return commonentity.BlockModule, "module", block.Labels[0], "module." + block.Labels[0], true, nil
	default:
		return "", "", "", "", false, nil
	}
}

func inspectNestedBlocks(body *hclsyntax.Body, filename, address string) ([]string, []string, []commonentity.Diagnostic) {
	var types []string
	var references []string
	var diagnostics []commonentity.Diagnostic
	for _, block := range body.Blocks {
		types = append(types, block.Type)
		if block.Type == "dynamic" {
			diagnostics = append(diagnostics, commonentity.Diagnostic{
				Code:     "TFCONFIG-W002",
				Severity: commonentity.SeverityWarning,
				Summary:  "Dynamic block is not evaluated",
				Detail:   fmt.Sprintf("%s contains a dynamic block; static references are retained but expanded content is unknown.", address),
				Range:    sourceRange(filename, block.TypeRange),
			})
		}
		for _, attribute := range block.Body.Attributes {
			references = append(references, expressionReferences(attribute.Expr)...)
		}
		childTypes, childReferences, childDiagnostics := inspectNestedBlocks(block.Body, filename, address)
		types = append(types, childTypes...)
		references = append(references, childReferences...)
		diagnostics = append(diagnostics, childDiagnostics...)
	}
	return sortedUnique(types), sortedUnique(references), diagnostics
}

func expressionReferences(expression hcl.Expression) []string {
	var addresses []string
	for _, traversal := range expression.Variables() {
		if address := traversalAddress(traversal); address != "" {
			addresses = append(addresses, address)
		}
	}
	return sortedUnique(addresses)
}

func traversalAddress(traversal hcl.Traversal) string {
	if len(traversal) == 0 {
		return ""
	}
	root, ok := traversal[0].(hcl.TraverseRoot)
	if !ok {
		return ""
	}
	ignored := map[string]bool{
		"count": true, "each": true, "local": true, "path": true,
		"self": true, "terraform": true, "var": true,
	}
	if ignored[root.Name] {
		return ""
	}
	attributes := make([]string, 0, 2)
	for _, traverser := range traversal[1:] {
		attribute, ok := traverser.(hcl.TraverseAttr)
		if !ok {
			continue
		}
		attributes = append(attributes, attribute.Name)
		if root.Name == "data" && len(attributes) == 2 {
			return "data." + attributes[0] + "." + attributes[1]
		}
		if root.Name == "module" && len(attributes) == 1 {
			return "module." + attributes[0]
		}
		if root.Name != "data" && root.Name != "module" && len(attributes) == 1 {
			return root.Name + "." + attributes[0]
		}
	}
	return ""
}

func constantValue(expression hcl.Expression) commonentity.Value {
	value, diagnostics := expression.Value(&hcl.EvalContext{})
	if diagnostics.HasErrors() || value == cty.NilVal || !value.IsKnown() || value.IsNull() {
		return commonentity.Value{}
	}
	return fromCTY(value)
}

func fromCTY(value cty.Value) commonentity.Value {
	if value == cty.NilVal || !value.IsKnown() || value.IsNull() {
		return commonentity.Value{}
	}
	typeValue := value.Type()
	switch {
	case typeValue.Equals(cty.String):
		return commonentity.Value{Kind: commonentity.ValueString, String: value.AsString()}
	case typeValue.Equals(cty.Bool):
		return commonentity.Value{Kind: commonentity.ValueBool, Bool: value.True()}
	case typeValue.Equals(cty.Number):
		return commonentity.Value{Kind: commonentity.ValueNumber, String: value.AsBigFloat().Text('g', -1)}
	case typeValue.IsObjectType() || typeValue.IsMapType():
		result := commonentity.Value{Kind: commonentity.ValueObject, Object: map[string]commonentity.Value{}}
		iterator := value.ElementIterator()
		for iterator.Next() {
			key, child := iterator.Element()
			if !key.IsKnown() || key.IsNull() {
				return commonentity.Value{}
			}
			result.Object[key.AsString()] = fromCTY(child)
		}
		return result
	case typeValue.IsTupleType() || typeValue.IsListType() || typeValue.IsSetType():
		result := commonentity.Value{Kind: commonentity.ValueList}
		iterator := value.ElementIterator()
		for iterator.Next() {
			_, child := iterator.Element()
			result.List = append(result.List, fromCTY(child))
		}
		return result
	default:
		return commonentity.Value{}
	}
}

func convertHCLDiagnostics(values hcl.Diagnostics) []commonentity.Diagnostic {
	result := make([]commonentity.Diagnostic, 0, len(values))
	for _, value := range values {
		severity := commonentity.SeverityWarning
		if value.Severity == hcl.DiagError {
			severity = commonentity.SeverityError
		}
		rangeValue := commonentity.SourceRange{}
		if value.Subject != nil {
			rangeValue = sourceRange(value.Subject.Filename, *value.Subject)
		}
		result = append(result, commonentity.Diagnostic{
			Code:     "TFCONFIG-E003",
			Severity: severity,
			Summary:  value.Summary,
			Detail:   value.Detail,
			Range:    rangeValue,
		})
	}
	return result
}

func sourceRange(filename string, value hcl.Range) commonentity.SourceRange {
	if filename == "" {
		filename = value.Filename
	}
	return commonentity.SourceRange{
		Filename:    filename,
		StartLine:   value.Start.Line,
		StartColumn: value.Start.Column,
		EndLine:     value.End.Line,
		EndColumn:   value.End.Column,
	}
}

func sortedUnique(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if value == "" || (len(result) > 0 && result[len(result)-1] == value) {
			continue
		}
		result = append(result, value)
	}
	return result
}
