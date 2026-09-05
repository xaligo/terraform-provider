package aws_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	awsentity "github.com/xaligo/terraform-provider/internal/entity/aws"
)

const (
	awsProviderSnapshotFile   = "terraform-provider-aws.json"
	awsProviderSnapshotSHA256 = "adb96ba0bbcfe5623987c251ff45c48e3143110d569234c8cf9c28acdd59b78f"
)

func TestAWSProviderDefinitionSnapshotIsCompleteAndCanonical(t *testing.T) {
	t.Parallel()

	snapshot, content := loadAWSProviderSnapshot(t)
	if snapshot.Source != "https://github.com/hashicorp/terraform-provider-aws" ||
		snapshot.Version != "v6.63.0" ||
		snapshot.Commit != "07c0e849a5d45731848cc9b10eab557cbc141d76" ||
		snapshot.License != "MPL-2.0" {
		t.Fatalf("unexpected snapshot provenance: %#v", snapshot)
	}
	digest := sha256.Sum256(content)
	if got := hex.EncodeToString(digest[:]); got != awsProviderSnapshotSHA256 {
		t.Fatalf("snapshot SHA-256 = %s, want %s", got, awsProviderSnapshotSHA256)
	}

	counts := make(map[awsentity.DefinitionKind]int)
	previousKey := ""
	for index, definition := range snapshot.Definitions {
		if definition.Kind == "" || definition.Service == "" || definition.TypeName == "" {
			t.Fatalf("definition %d has an empty field: %#v", index, definition)
		}
		if definition.Kind == awsentity.DefinitionKindFunction {
			if strings.HasPrefix(definition.TypeName, "aws_") {
				t.Errorf("provider function unexpectedly uses an aws_ prefix: %s", definition.TypeName)
			}
		} else if !strings.HasPrefix(definition.TypeName, "aws_") {
			t.Errorf("%s definition does not use an aws_ prefix: %s", definition.Kind, definition.TypeName)
		}
		key := string(definition.Kind) + "\x00" + definition.TypeName
		if key <= previousKey {
			t.Fatalf("definitions are not strictly ordered and unique at %d: %q after %q", index, key, previousKey)
		}
		previousKey = key
		counts[definition.Kind]++
	}

	wantCounts := map[awsentity.DefinitionKind]int{
		awsentity.DefinitionKindAction:            12,
		awsentity.DefinitionKindDataSource:        679,
		awsentity.DefinitionKindEphemeralResource: 10,
		awsentity.DefinitionKindFunction:          4,
		awsentity.DefinitionKindListResource:      220,
		awsentity.DefinitionKindResource:          1715,
	}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("definition counts = %#v, want %#v", counts, wantCounts)
	}
	if len(snapshot.Definitions) != 2640 {
		t.Fatalf("definition total = %d, want 2640", len(snapshot.Definitions))
	}
}

func TestAWSProviderEntityRegistryMatchesSnapshot(t *testing.T) {
	t.Parallel()

	want, _ := loadAWSProviderSnapshot(t)
	got := awsentity.DefinitionSnapshot()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entity definition snapshot does not match %s", awsProviderSnapshotFile)
	}
	for _, definition := range want.Definitions {
		found, ok := awsentity.LookupDefinition(definition.Kind, definition.TypeName)
		if !ok {
			t.Fatalf("LookupDefinition(%q, %q) did not find a pinned definition", definition.Kind, definition.TypeName)
		}
		if found != definition {
			t.Fatalf("LookupDefinition(%q, %q) = %#v, want %#v", definition.Kind, definition.TypeName, found, definition)
		}
	}
	if _, ok := awsentity.LookupDefinition(awsentity.DefinitionKindResource, "aws_not_a_real_resource"); ok {
		t.Fatal("LookupDefinition found an unknown resource")
	}

	definitions := awsentity.Definitions()
	definitions[0].TypeName = "mutated"
	if fresh := awsentity.Definitions(); fresh[0].TypeName == "mutated" {
		t.Fatal("Definitions returned mutable registry storage")
	}
}

func TestAWSProviderDefinitionsArePartitionedByKindAndService(t *testing.T) {
	t.Parallel()

	snapshot, _ := loadAWSProviderSnapshot(t)
	expected := make(map[string]int)
	for _, definition := range snapshot.Definitions {
		relative := filepath.Join(string(definition.Kind), definition.Service+"_definitions_gen.go")
		expected[relative]++
	}

	root := awsEntityRoot(t)
	if _, err := os.Stat(filepath.Join(root, "provider_definitions_gen.go")); !os.IsNotExist(err) {
		t.Fatalf("monolithic generated registry must not exist: %v", err)
	}
	actual := make(map[string]bool, len(expected))
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_definitions_gen.go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		actual[relative] = true
		want, ok := expected[relative]
		if !ok {
			t.Errorf("unexpected generated kind/service file: %s", relative)
			return nil
		}
		assertServiceDefinitionVariable(t, path, want)
		return nil
	})
	if err != nil {
		t.Fatalf("walk generated AWS definitions: %v", err)
	}
	for relative := range expected {
		if !actual[relative] {
			t.Errorf("missing generated kind/service file: %s", relative)
		}
	}
	if len(actual) != len(expected) {
		t.Fatalf("generated kind/service files = %d, want %d", len(actual), len(expected))
	}
}

func assertServiceDefinitionVariable(t *testing.T, path string, want int) {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse generated file %s: %v", path, err)
	}
	variables := 0
	elements := -1
	for _, declaration := range parsed.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
				continue
			}
			literal, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}
			if !ast.IsExported(value.Names[0].Name) {
				t.Errorf("service definition variable is not exported in %s: %s", path, value.Names[0].Name)
			}
			variables++
			elements = len(literal.Elts)
		}
	}
	if variables != 1 || elements != want {
		t.Errorf("generated file %s has %d variables and %d definitions, want 1 variable and %d definitions", path, variables, elements, want)
	}
}

func awsEntityRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
	path := filepath.Join(root, "internal", "entity", "aws")
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("resolve internal/entity/aws: %v", err)
	}
	return path
}

func loadAWSProviderSnapshot(t *testing.T) (awsentity.ProviderDefinitionSnapshot, []byte) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	path := filepath.Join(filepath.Dir(filename), "testdata", awsProviderSnapshotFile)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read AWS Provider definition snapshot: %v", err)
	}
	var snapshot awsentity.ProviderDefinitionSnapshot
	if err := json.Unmarshal(content, &snapshot); err != nil {
		t.Fatalf("decode AWS Provider definition snapshot: %v", err)
	}
	if len(snapshot.Definitions) == 0 {
		t.Fatal("AWS Provider definition snapshot is empty")
	}
	return snapshot, content
}
