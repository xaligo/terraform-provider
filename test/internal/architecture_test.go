package internal_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const (
	modulePath         = "github.com/xaligo/terraform-provider"
	moduleInternalRoot = modulePath + "/internal"
	moduleInternal     = moduleInternalRoot + "/"
)

func TestCleanArchitectureDependencies(t *testing.T) {
	t.Parallel()

	root := internalRoot(t)
	if _, err := os.Stat(filepath.Join(root, "old")); !os.IsNotExist(err) {
		t.Fatalf("internal/old must not exist: %v", err)
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Errorf("parse %s: %v", relative, err)
			return nil
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Errorf("decode import in %s: %v", relative, err)
				continue
			}
			if violation := dependencyViolation(filepath.ToSlash(relative), importPath); violation != "" {
				t.Errorf("%s imports %s: %s", relative, importPath, violation)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal packages: %v", err)
	}
}

func TestApplicationStructsAreConstructorBacked(t *testing.T) {
	t.Parallel()

	for _, layer := range []string{"controller", "usecase", "repository"} {
		assertStructsAreConstructorBacked(t, filepath.Join(internalRoot(t), layer))
	}
}

func TestRouterOwnsComponentConstruction(t *testing.T) {
	t.Parallel()

	root := internalRoot(t)
	basePath := filepath.Join(root, "base.go")
	if info, err := os.Stat(basePath); err != nil || info.Size() == 0 {
		t.Fatalf("internal/base.go must be the non-empty application entry point: %v", err)
	}
	routerPath := filepath.Join(root, "router.go")
	if info, err := os.Stat(routerPath); err != nil || info.Size() == 0 {
		t.Fatalf("internal/router.go must be the non-empty composition root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "config")); !os.IsNotExist(err) {
		t.Fatalf("internal/config must not exist after composition moves to internal/router.go: %v", err)
	}

	constructors := componentConstructors(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == filepath.Join(root, "entity") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := calledFunctionName(call.Fun)
			owner, known := constructors[name]
			if !known || path == routerPath || path == owner {
				return true
			}
			t.Errorf("%s calls component constructor %s declared in %s; compose components only in internal/router.go",
				fileSet.Position(call.Pos()), name, owner)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("inspect component construction: %v", err)
	}

	parsed, err := parser.ParseFile(token.NewFileSet(), basePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse internal/base.go: %v", err)
	}
	for _, specification := range parsed.Imports {
		importPath, err := strconv.Unquote(specification.Path.Value)
		if err != nil {
			t.Fatalf("decode import in internal/base.go: %v", err)
		}
		if strings.HasPrefix(importPath, moduleInternal) {
			t.Errorf("internal/base.go imports %s; lifecycle entry points must delegate only to internal/router.go", importPath)
		}
	}
}

func TestLayerComponentsDoNotDependOnPeers(t *testing.T) {
	t.Parallel()

	for _, layer := range []string{"controller", "usecase", "repository"} {
		assertNoPeerComponentDependencies(t, filepath.Join(internalRoot(t), layer))
	}
}

func TestCommandEntrypointsDelegateToInternalBase(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	basePath := filepath.Join(root, "internal", "base.go")
	baseFunctions := declaredFunctions(t, basePath)
	for _, relative := range []string{"main.go", "cmd/cli/main.go", "cmd/provider/main.go"} {
		path := filepath.Join(root, filepath.FromSlash(relative))
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", relative, err)
		}

		baseAlias := ""
		for _, specification := range parsed.Imports {
			importPath, err := strconv.Unquote(specification.Path.Value)
			if err != nil {
				t.Fatalf("decode import in %s: %v", relative, err)
			}
			if strings.HasPrefix(importPath, modulePath+"/") && importPath != moduleInternalRoot {
				t.Errorf("%s imports %s; executable entry points may import only the internal/base.go package", relative, importPath)
			}
			if importPath == moduleInternalRoot {
				baseAlias = "internal"
				if specification.Name != nil {
					baseAlias = specification.Name.Name
				}
			}
		}
		if baseAlias == "" {
			t.Errorf("%s must import %s", relative, moduleInternalRoot)
			continue
		}

		callsBase := false
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			identifier, ok := selector.X.(*ast.Ident)
			if !ok || identifier.Name != baseAlias {
				return true
			}
			callsBase = true
			if !baseFunctions[selector.Sel.Name] {
				t.Errorf("%s calls %s.%s, which is not declared in internal/base.go", fileSet.Position(call.Pos()), baseAlias, selector.Sel.Name)
			}
			return true
		})
		if !callsBase {
			t.Errorf("%s must delegate execution to internal/base.go", relative)
		}
	}
}

func TestEntityRootContainsSubpackagesOnly(t *testing.T) {
	t.Parallel()

	root := filepath.Join(internalRoot(t), "entity")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read entity root: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
			t.Errorf("entity Go file must be placed in a concept subpackage: %s", filepath.Join(root, entry.Name()))
		}
	}
}

func TestMethodReceiversAreNamedRcvr(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		for _, declaration := range parsed.Decls {
			method, ok := declaration.(*ast.FuncDecl)
			if !ok || method.Recv == nil {
				continue
			}
			if len(method.Recv.List) != 1 || len(method.Recv.List[0].Names) != 1 || method.Recv.List[0].Names[0].Name != "rcvr" {
				t.Errorf("%s method %s must use receiver name rcvr", path, method.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inspect method receivers: %v", err)
	}
}

func TestTestsMirrorInternalDirectoryTree(t *testing.T) {
	t.Parallel()

	productionRoot := internalRoot(t)
	err := filepath.WalkDir(productionRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(path, "_test.go") {
			t.Errorf("test must be under test/internal: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production tree: %v", err)
	}

	testRoot := filepath.Join(repositoryRoot(t), "test", "internal")
	err = filepath.WalkDir(testRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		relativeDirectory, err := filepath.Rel(testRoot, filepath.Dir(path))
		if err != nil {
			return err
		}
		productionDirectory := filepath.Join(productionRoot, relativeDirectory)
		info, err := os.Stat(productionDirectory)
		if err != nil || !info.IsDir() {
			t.Errorf("test directory has no mirrored internal directory: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk test tree: %v", err)
	}
}

func assertStructsAreConstructorBacked(t *testing.T, root string) {
	t.Helper()
	structs := make(map[string]map[string]bool)
	constructed := make(map[string]map[string]bool)
	interfaces := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		if structs[path] == nil {
			structs[path] = make(map[string]bool)
			constructed[path] = make(map[string]bool)
		}
		for _, declaration := range parsed.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if ok {
				for _, specification := range general.Specs {
					typeSpecification, ok := specification.(*ast.TypeSpec)
					if !ok {
						continue
					}
					switch typeSpecification.Type.(type) {
					case *ast.StructType:
						structs[path][typeSpecification.Name.Name] = true
					case *ast.InterfaceType:
						interfaces[path] = true
					}
				}
				continue
			}
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil ||
				(!strings.HasPrefix(function.Name.Name, "New") && !strings.HasPrefix(function.Name.Name, "new")) {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				if identifier, ok := literal.Type.(*ast.Ident); ok {
					constructed[path][identifier.Name] = true
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inspect constructors below %s: %v", root, err)
	}
	for path, names := range structs {
		for name := range names {
			if ast.IsExported(name) {
				t.Errorf("%s declares exported implementation struct %s; expose an interface and keep its implementation private", path, name)
			}
			if !interfaces[path] {
				t.Errorf("%s declares implementation struct %s without an interface in the same concept file", path, name)
			}
			if !constructed[path][name] {
				t.Errorf("%s declares implementation struct %s without a constructor in the same concept file", path, name)
			}
		}
	}
}

func assertNoPeerComponentDependencies(t *testing.T, root string) {
	t.Helper()

	typeOwners := make(map[string]string)
	constructorOwners := make(map[string]string)
	parsedFiles := make(map[string]*ast.File)
	fileSets := make(map[string]*token.FileSet)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fileSet := token.NewFileSet()
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return err
		}
		parsedFiles[path] = parsed
		fileSets[path] = fileSet
		for _, declaration := range parsed.Decls {
			switch value := declaration.(type) {
			case *ast.GenDecl:
				for _, specification := range value.Specs {
					typeSpecification, ok := specification.(*ast.TypeSpec)
					if !ok {
						continue
					}
					switch typeSpecification.Type.(type) {
					case *ast.InterfaceType, *ast.StructType:
						typeOwners[typeSpecification.Name.Name] = path
					}
				}
			case *ast.FuncDecl:
				if value.Recv == nil && strings.HasPrefix(value.Name.Name, "New") {
					constructorOwners[value.Name.Name] = path
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("inspect layer declarations below %s: %v", root, err)
	}

	for path, parsed := range parsedFiles {
		fileSet := fileSets[path]
		reported := make(map[string]bool)
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.CallExpr:
				name := calledFunctionName(value.Fun)
				if owner, exists := constructorOwners[name]; exists && owner != path {
					key := "constructor:" + name
					if !reported[key] {
						t.Errorf("%s calls peer constructor %s from %s; same-layer components must not construct or call one another",
							fileSet.Position(value.Pos()), name, owner)
						reported[key] = true
					}
				}
			case *ast.Ident:
				if owner, exists := typeOwners[value.Name]; exists && owner != path {
					key := "type:" + value.Name
					if !reported[key] {
						t.Errorf("%s references peer component type %s from %s; same-layer components must remain independent",
							fileSet.Position(value.Pos()), value.Name, owner)
						reported[key] = true
					}
				}
			}
			return true
		})
	}
}

func componentConstructors(t *testing.T) map[string]string {
	t.Helper()

	constructors := make(map[string]string)
	for _, layer := range []string{"controller", "usecase", "repository"} {
		root := filepath.Join(internalRoot(t), layer)
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			for _, declaration := range parsed.Decls {
				function, ok := declaration.(*ast.FuncDecl)
				if !ok || function.Recv != nil || !strings.HasPrefix(function.Name.Name, "New") {
					continue
				}
				if previous, exists := constructors[function.Name.Name]; exists && previous != path {
					t.Fatalf("component constructor %s is declared in both %s and %s", function.Name.Name, previous, path)
				}
				constructors[function.Name.Name] = path
			}
			return nil
		})
		if err != nil {
			t.Fatalf("inspect constructors below %s: %v", root, err)
		}
	}
	return constructors
}

func declaredFunctions(t *testing.T, path string) map[string]bool {
	t.Helper()

	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	functions := make(map[string]bool)
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil {
			functions[function.Name.Name] = true
		}
	}
	return functions
}

func calledFunctionName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	default:
		return ""
	}
}

func dependencyViolation(file, importPath string) string {
	if !strings.HasPrefix(importPath, moduleInternal) {
		return ""
	}
	dependency := strings.TrimPrefix(importPath, moduleInternal)
	layer := strings.SplitN(file, "/", 2)[0]
	switch layer {
	case "entity":
		if dependency != "entity" && !strings.HasPrefix(dependency, "entity/") {
			return "entities may depend only on entity packages"
		}
	case "repository":
		if hasLayer(dependency, "config", "controller", "share", "usecase") {
			return "repositories must not depend on delivery or use-case layers"
		}
	case "usecase":
		if dependency == "entity" || strings.HasPrefix(dependency, "entity/") || dependency == "repository" {
			return ""
		}
		return "use cases may depend only on entities and repository interfaces"
	case "controller":
		if hasLayer(dependency, "config", "repository", "share") {
			return "controllers must not depend on composition or repository implementations"
		}
	case "share":
		return "shared delivery helpers must not depend on other internal layers"
	}
	return ""
}

func hasLayer(importPath string, layers ...string) bool {
	for _, layer := range layers {
		if importPath == layer || strings.HasPrefix(importPath, layer+"/") {
			return true
		}
	}
	return false
}

func internalRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(repositoryRoot(t), "internal")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
