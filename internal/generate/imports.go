package generate

import (
	"sort"
	"strings"

	"github.com/miyamo2/braider/internal/graph"
)

// CollectImports extracts unique package paths from the dependency graph.
// It excludes the current package and returns a sorted list of import paths.
func CollectImports(g *graph.Graph, currentPackage string) []string {
	if g == nil {
		return nil
	}

	importSet := make(map[string]bool)

	// Extract package paths from all nodes
	for _, node := range g.Nodes {
		pkgPath := node.PackagePath
		if pkgPath != "" && pkgPath != currentPackage {
			importSet[pkgPath] = true
		}
	}

	// Convert to sorted slice
	imports := make([]string, 0, len(importSet))
	for pkgPath := range importSet {
		imports = append(imports, pkgPath)
	}
	sort.Strings(imports)

	return imports
}

// ExtractPackagePath extracts the package path from a fully qualified type name.
// Example: "github.com/user/repo.Service" -> "github.com/user/repo"
func ExtractPackagePath(typeName string) string {
	lastDot := strings.LastIndex(typeName, ".")
	if lastDot == -1 {
		return ""
	}
	return typeName[:lastDot]
}
