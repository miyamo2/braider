package detect

import (
	"fmt"
	"go/ast"
	"go/types"
	"iter"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// LoadTestPackage loads a test package from testdata/ using the real Go toolchain.
// This approach provides realistic type checking and dependency resolution.
//
// Usage:
//
//	pkg, pass := LoadTestPackage(t, "option_extractor/default")
//
// The testdata package must have:
//   - main.go with valid Go source code
//   - go.mod with replace directive for github.com/miyamo2/braider/pkg
func LoadTestPackage(t *testing.T, relativeDir string) (*packages.Package, *analysis.Pass) {
	t.Helper()

	// Get absolute path to testdata directory
	testdataDir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("Failed to get testdata directory: %v", err)
	}

	dir := filepath.Join(testdataDir, relativeDir)

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo,
		Dir:   dir,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		t.Fatalf("Failed to load package: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		t.Fatal("Package has errors")
	}

	if len(pkgs) == 0 {
		t.Fatal("No packages loaded")
	}

	pkg := pkgs[0]

	pass := &analysis.Pass{
		Fset:      pkg.Fset,
		Files:     pkg.Syntax,
		TypesInfo: pkg.TypesInfo,
		Pkg:       pkg.Types,
	}

	return pkg, pass
}

// FindInjectableField finds the first Injectable field in the package
// and returns the field expression and the containing struct type.
func FindInjectableField(t *testing.T, pkg *packages.Package) (ast.Expr, *ast.StructType, string) {
	t.Helper()
	var injectableExpr ast.Expr
	var structType *ast.StructType
	var structName string

	for _, file := range pkg.Syntax {
		ast.Inspect(
			file, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok {
						for _, field := range st.Fields.List {
							// Check for IndexExpr (Go 1.18+)
							if indexExpr, ok := field.Type.(*ast.IndexExpr); ok {
								if sel, ok := indexExpr.X.(*ast.SelectorExpr); ok {
									if sel.Sel.Name == "Injectable" {
										injectableExpr = indexExpr
										structType = st
										structName = ts.Name.Name
										return false
									}
								}
							}
							// Check for IndexListExpr (Go 1.19+, for multiple type params)
							if indexListExpr, ok := field.Type.(*ast.IndexListExpr); ok {
								if sel, ok := indexListExpr.X.(*ast.SelectorExpr); ok {
									if sel.Sel.Name == "Injectable" {
										injectableExpr = indexListExpr
										structType = st
										structName = ts.Name.Name
										return false
									}
								}
							}
						}
					}
				}
				return true
			},
		)
		if injectableExpr != nil {
			break
		}
	}
	if injectableExpr == nil {
		t.Fatal("Injectable field not found in package")
	}

	return injectableExpr, structType, structName
}

// FindVariableCall finds the first annotation.Variable[T](value) call expression in the package
// and returns the call expression and the argument type.
func FindVariableCall(t *testing.T, pkg *packages.Package) (*ast.CallExpr, types.Type) {
	t.Helper()

	detector := NewVariableCallDetector(ResolveMarkers())
	pass := &analysis.Pass{
		Fset:      pkg.Fset,
		Files:     pkg.Syntax,
		TypesInfo: pkg.TypesInfo,
		Pkg:       pkg.Types,
	}

	candidates, errs := detector.DetectVariables(pass)
	if len(errs) != 0 {
		t.Fatalf("unexpected detection errors in FindVariableCall: %v", errs)
	}
	if len(candidates) == 0 {
		t.Fatal("No Variable call found in package")
	}

	return candidates[0].CallExpr, candidates[0].ArgumentType
}

// MockNamerValidator is a mock implementation of NamerValidator for testing.
type MockNamerValidator struct {
	ExtractNameFn func(pass *analysis.Pass, namerType types.Type) (string, error)
}

// ExtractName implements the NamerValidator interface.
func (m *MockNamerValidator) ExtractName(pass *analysis.Pass, namerType types.Type) (string, error) {
	if m.ExtractNameFn != nil {
		return m.ExtractNameFn(pass, namerType)
	}
	// Default: extract name from type name if it ends with "Name"
	if named, ok := namerType.(*types.Named); ok {
		typeName := named.Obj().Name()
		if suffix, found := strings.CutSuffix(typeName, "Name"); found {
			return strings.ToLower(suffix), nil
		}
	}
	return "", nil
}

// FindNamedType finds a named type by name in a package.
func FindNamedType(pkg *packages.Package, typeName string) *types.Named {
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return nil
	}
	namedType, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}
	return namedType
}

// MockPackageLoader implements loader.PackageLoader for testing external package validation.
type MockPackageLoader struct {
	Packages map[string]*packages.Package
}

// LoadPackage implements the loader.PackageLoader interface.
func (m *MockPackageLoader) LoadPackage(pkgPath string) (*packages.Package, error) {
	pkg, ok := m.Packages[pkgPath]
	if !ok {
		return nil, fmt.Errorf("package not found: %s", pkgPath)
	}
	return pkg, nil
}

// LoadModulePackageNames implements the loader.PackageLoader interface (not used in tests).
func (m *MockPackageLoader) LoadModulePackageNames(dir string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

// LoadModulePackageAST implements the loader.PackageLoader interface (not used in tests).
func (m *MockPackageLoader) LoadModulePackageAST(dir string) (iter.Seq[*packages.Package], error) {
	return nil, fmt.Errorf("not implemented")
}

// FindModuleRoot implements the loader.PackageLoader interface (not used in tests).
func (m *MockPackageLoader) FindModuleRoot(dir string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
