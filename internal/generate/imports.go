package generate

import (
	"go/ast"
	"go/types"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/miyamo2/braider/internal/graph"
)

// ImportInfo represents a package import with optional alias.
type ImportInfo struct {
	Path  string // Import path (e.g., "example.com/v1/user")
	Alias string // Alias name (empty string = no alias needed)
}

// HasAlias returns true if this import requires an alias.
func (i *ImportInfo) HasAlias() bool {
	return i.Alias != ""
}

// Go reserved keywords
var goReservedKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// CollectImports extracts unique package paths from the dependency graph.
// It excludes the current package and returns a sorted list of import info with aliases,
// along with an alias map (pkgPath -> alias) for use by qualifier functions.
func CollectImports(
	g *graph.Graph,
	currentPackage, currentPkgName string,
	existingAliases map[string]string,
) ([]ImportInfo, map[string]string) {
	if g == nil {
		return []ImportInfo{}, make(map[string]string)
	}

	importSet := make(map[string]bool)

	// Extract package paths from all nodes
	for _, node := range g.Nodes {
		pkgPath := node.PackagePath
		if pkgPath == "" {
			continue
		}

		// Include if different package (by name)
		// For main packages, also check path to distinguish entry points
		if node.PackageName == "main" && currentPkgName == "main" {
			if pkgPath != currentPackage {
				// Different main package - shouldn't import each other
				continue
			}
		} else if node.PackageName != currentPkgName {
			importSet[pkgPath] = true
		}

		// Also include packages referenced by RegisteredType (for Typed[I] option)
		if node.RegisteredType != nil {
			for _, regPkgPath := range extractPackagePaths(node.RegisteredType) {
				if regPkgPath != "" && regPkgPath != currentPackage {
					importSet[regPkgPath] = true
				}
			}
		}

		// Also include packages referenced by expression (for Variable nodes)
		for _, exprPkgPath := range node.ExpressionPkgs {
			if exprPkgPath != "" && exprPkgPath != currentPackage {
				importSet[exprPkgPath] = true
			}
		}

		// Also include the constructor function's package (for Provide nodes where
		// the function is defined in a different package than the return type)
		if node.ConstructorPkgPath != "" && node.ConstructorPkgPath != currentPackage {
			if node.ConstructorPkgName == "main" && currentPkgName == "main" {
				// Different main package - shouldn't import each other
			} else if node.ConstructorPkgName != currentPkgName || node.ConstructorPkgPath != node.PackagePath {
				importSet[node.ConstructorPkgPath] = true
			}
		}
	}

	// Detect collisions and generate aliases
	collisions := detectPackageCollisions(g)
	aliasMap := generateAliases(collisions, existingAliases)

	// Apply aliases to graph nodes
	for _, node := range g.Nodes {
		if alias, exists := aliasMap[node.PackagePath]; exists {
			node.PackageAlias = alias
		}
		if alias, exists := aliasMap[node.ConstructorPkgPath]; exists {
			node.ConstructorPkgAlias = alias
		}
	}

	// Build ImportInfo list
	var imports []ImportInfo
	for pkgPath := range importSet {
		imp := ImportInfo{
			Path:  pkgPath,
			Alias: aliasMap[pkgPath], // empty string if no alias needed
		}
		imports = append(imports, imp)
	}

	// Sort by path for deterministic output
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Path < imports[j].Path
	})

	return imports, aliasMap
}

// extractPackagePaths extracts all unique package paths from a types.Type.
// This handles named types, pointer types, and interface types with embedded types.
func extractPackagePaths(t types.Type) []string {
	var paths []string
	seen := make(map[string]bool)

	var visit func(types.Type)
	visit = func(t types.Type) {
		switch typ := t.(type) {
		case *types.Named:
			if obj := typ.Obj(); obj != nil {
				if pkg := obj.Pkg(); pkg != nil {
					p := pkg.Path()
					if !seen[p] {
						seen[p] = true
						paths = append(paths, p)
					}
				}
			}
		case *types.Pointer:
			visit(typ.Elem())
		case *types.Interface:
			for i := 0; i < typ.NumEmbeddeds(); i++ {
				visit(typ.EmbeddedType(i))
			}
		}
	}
	visit(t)
	return paths
}

// detectExistingAliases scans the target file for user-defined import aliases.
// Returns map[packagePath]alias for all aliased imports.
func detectExistingAliases(file *ast.File) map[string]string {
	aliasMap := make(map[string]string)

	if file == nil {
		return aliasMap
	}

	for _, importSpec := range file.Imports {
		pkgPath := strings.Trim(importSpec.Path.Value, "\"")

		// Check for user-defined alias (skip . and _ imports)
		if importSpec.Name != nil &&
			importSpec.Name.Name != "" &&
			importSpec.Name.Name != "." &&
			importSpec.Name.Name != "_" {
			aliasMap[pkgPath] = importSpec.Name.Name
		}
	}

	return aliasMap
}

// extractPackageName extracts the package name from a types.Type for a given package path.
// It unwraps pointer types to find the underlying named type.
func extractPackageName(t types.Type, targetPath string) string {
	switch typ := t.(type) {
	case *types.Named:
		if obj := typ.Obj(); obj != nil {
			if pkg := obj.Pkg(); pkg != nil && pkg.Path() == targetPath {
				return pkg.Name()
			}
		}
	case *types.Pointer:
		return extractPackageName(typ.Elem(), targetPath)
	case *types.Interface:
		for i := 0; i < typ.NumEmbeddeds(); i++ {
			if name := extractPackageName(typ.EmbeddedType(i), targetPath); name != "" {
				return name
			}
		}
	}
	return ""
}

// detectPackageCollisions identifies packages with duplicate names.
// Returns map[packagePath]packageName for packages involved in collisions.
func detectPackageCollisions(g *graph.Graph) map[string]string {
	if g == nil {
		return make(map[string]string)
	}

	nameToPathsMap := make(map[string][]string)
	pathToNameMap := make(map[string]string)

	// addPackage registers a package path under its name, avoiding duplicates.
	addPackage := func(pkgName, pkgPath string) {
		if pkgName == "" || pkgPath == "" {
			return
		}
		// Check for duplicate path in this name's list
		if existingName, exists := pathToNameMap[pkgPath]; exists && existingName == pkgName {
			return
		}
		nameToPathsMap[pkgName] = append(nameToPathsMap[pkgName], pkgPath)
		pathToNameMap[pkgPath] = pkgName
	}

	// Build mappings from node packages
	for _, node := range g.Nodes {
		addPackage(node.PackageName, node.PackagePath)

		// Also include constructor function's package (may differ from type package for Provide)
		if node.ConstructorPkgName != "" && node.ConstructorPkgPath != "" {
			addPackage(node.ConstructorPkgName, node.ConstructorPkgPath)
		}

		// Also include packages from RegisteredType (for Typed[I] interface types)
		if node.RegisteredType != nil {
			for _, regPkgPath := range extractPackagePaths(node.RegisteredType) {
				pkgName := extractPackageName(node.RegisteredType, regPkgPath)
				addPackage(pkgName, regPkgPath)
			}
		}

		// Also include packages from expression (for Variable nodes)
		for i, exprPkgPath := range node.ExpressionPkgs {
			if i < len(node.ExpressionPkgNames) {
				addPackage(node.ExpressionPkgNames[i], exprPkgPath)
			}
		}
	}

	// Extract only colliding packages (2+ paths with same name)
	collisions := make(map[string]string)
	for pkgName, paths := range nameToPathsMap {
		if len(paths) >= 2 {
			// Deduplicate paths
			uniquePaths := make(map[string]bool)
			for _, p := range paths {
				uniquePaths[p] = true
			}
			if len(uniquePaths) >= 2 {
				for path := range uniquePaths {
					collisions[path] = pkgName
				}
			}
		}
	}

	return collisions
}

// generateAliases creates unique aliases for colliding packages.
// Strategy: 1) Preserve existing, 2) Version-based, 3) Numbered
func generateAliases(
	collisions map[string]string,
	existingAliases map[string]string,
) map[string]string {
	aliasMap := make(map[string]string)
	usedAliases := make(map[string]bool)

	// First pass: preserve existing user-defined aliases
	for pkgPath := range collisions {
		if alias, exists := existingAliases[pkgPath]; exists {
			aliasMap[pkgPath] = alias
			usedAliases[alias] = true
		}
	}

	// Second pass: generate aliases for remaining collisions
	nameToPathsMap := make(map[string][]string)
	for pkgPath, pkgName := range collisions {
		if _, hasAlias := aliasMap[pkgPath]; !hasAlias {
			nameToPathsMap[pkgName] = append(nameToPathsMap[pkgName], pkgPath)
		}
	}

	for pkgName, paths := range nameToPathsMap {
		sort.Strings(paths) // Deterministic ordering

		for i, pkgPath := range paths {
			// Try version extraction first
			version := extractVersion(pkgPath)
			if version != "" {
				candidate := version + pkgName
				if !usedAliases[candidate] && !isReservedKeyword(candidate) {
					aliasMap[pkgPath] = candidate
					usedAliases[candidate] = true
					continue
				}
			}

			// First occurrence gets no alias (backward compatible)
			if i == 0 {
				aliasMap[pkgPath] = ""
				continue
			}

			// Fallback to numbered aliases
			for j := 2; ; j++ {
				candidate := pkgName + strconv.Itoa(j)
				if !usedAliases[candidate] && !isReservedKeyword(candidate) {
					aliasMap[pkgPath] = candidate
					usedAliases[candidate] = true
					break
				}
			}
		}
	}

	return aliasMap
}

// extractVersion extracts version identifier from package path.
// Examples: "example.com/v1/user" -> "v1", "example.com/user" -> ""
func extractVersion(pkgPath string) string {
	re := regexp.MustCompile(`/v(\d+)(?:/|$)`)
	if matches := re.FindStringSubmatch(pkgPath); len(matches) > 1 {
		return "v" + matches[1]
	}
	return ""
}

// isReservedKeyword checks if a name is a Go reserved keyword.
func isReservedKeyword(name string) bool {
	return goReservedKeywords[name]
}
