package generate

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/miyamo2/braider/internal/graph"
	"golang.org/x/tools/go/analysis"
)

// hashCommentRe is a compiled regular expression for extracting hash markers from comments.
// Cached at package level to avoid repeated compilation.
var hashCommentRe = regexp.MustCompile(`//\s*braider:hash:\s*([a-f0-9]+)`)

// GeneratedBootstrap represents the generated bootstrap code and metadata.
type GeneratedBootstrap struct {
	DependencyVar string       // IIFE code defining the dependency variable
	MainReference string       // Optional "_ = dependency" statement for main
	Imports       []ImportInfo // Required import paths with aliases
	Hash          string       // Hash marker for idempotency
}

// BootstrapGenerator generates bootstrap code for the DI system.
type BootstrapGenerator interface {
	GenerateBootstrap(pass *analysis.Pass, g *graph.Graph, sortedTypes []string) (*GeneratedBootstrap, error)
	CheckBootstrapCurrent(pass *analysis.Pass, existing *ast.GenDecl, g *graph.Graph) bool
	DetectExistingBootstrap(pass *analysis.Pass) *ast.GenDecl
}

// bootstrapGenerator is the default implementation.
type bootstrapGenerator struct {
	formatter CodeFormatter
}

// NewBootstrapGenerator creates a new bootstrap generator.
func NewBootstrapGenerator() BootstrapGenerator {
	return &bootstrapGenerator{
		formatter: NewCodeFormatter(),
	}
}

// GenerateBootstrap generates the complete bootstrap code.
func (bg *bootstrapGenerator) GenerateBootstrap(
	pass *analysis.Pass,
	g *graph.Graph,
	sortedTypes []string,
) (*GeneratedBootstrap, error) {
	if g == nil {
		return nil, fmt.Errorf("nil dependency graph")
	}

	// Compute hash for idempotency (works even with empty graph)
	hash := ComputeGraphHash(g)

	// Detect existing aliases from the target file
	// Find the file containing App annotation or existing bootstrap
	var targetFile *ast.File
	existingBootstrap := bg.DetectExistingBootstrap(pass)
	if existingBootstrap != nil {
		// Find file containing existing bootstrap
		for _, file := range pass.Files {
			for _, decl := range file.Decls {
				if decl == existingBootstrap {
					targetFile = file
					break
				}
			}
			if targetFile != nil {
				break
			}
		}
	}
	// If no existing bootstrap, use the first file (typically main.go)
	if targetFile == nil && len(pass.Files) > 0 {
		targetFile = pass.Files[0]
	}
	existingAliases := detectExistingAliases(targetFile)

	// Collect imports
	currentPackage := pass.Pkg.Path()
	currentPkgName := pass.Pkg.Name()
	imports, aliasMap := CollectImports(g, currentPackage, currentPkgName, existingAliases)

	// Build struct fields and initialization code
	// Note: If all nodes are Variable (IsField=false), structFields will be empty
	var structFields []string
	var inits []string
	var returnFields []string

	// Track field names and their types
	fieldNames := make(map[string]string) // typeName -> fieldName

	// Phase 1: Build struct fields for Inject and Provide nodes
	// Inject and Provide nodes (IsField=true) become fields in the dependency struct
	// that is returned to the caller. These are the components that the application can
	// access and use. Variable nodes (IsField=false) are local variables within the IIFE
	// and are not exposed to the caller.
	for _, typeName := range sortedTypes {
		node := g.Nodes[typeName]
		if node == nil {
			continue
		}

		// Only include Inject structs as fields (IsField = true)
		if !node.IsField {
			continue
		}

		// Use custom name if provided (Named[N] option), otherwise derive from type name
		fieldName := node.Name
		if fieldName == "" {
			fieldName = DeriveFieldName(typeName)
		}
		fieldNames[typeName] = fieldName

		// Determine qualifier (alias takes precedence)
		qualifier := node.PackageAlias
		if qualifier == "" {
			qualifier = node.PackageName
		}

		// Add struct field with qualified type name
		// Use RegisteredType if available (for Typed[I] option), otherwise use concrete type
		var qualifiedType string
		if node.RegisteredType != nil {
			// Use RegisteredType for interface-typed dependencies
			qualifiedType = types.TypeString(node.RegisteredType, func(p *types.Package) string {
				if p == nil {
					return ""
				}
				if p.Path() == currentPackage {
					return "" // Same package, no qualifier
				}
				// Look up alias for the RegisteredType's package from the alias map
				if alias, ok := aliasMap[p.Path()]; ok && alias != "" {
					return alias
				}
				return p.Name()
			})
		} else {
			// Use concrete type (default behavior)
			// For main packages, also check PackagePath to distinguish multiple entry points
			if node.PackageName == "main" && currentPkgName == "main" {
				// Both are main - check if same entry point
				if node.PackagePath != currentPackage {
					// Different main packages - this shouldn't happen in same analysis,
					// but if it does, use unqualified (they're in different binaries)
					qualifiedType = node.LocalName
				} else {
					qualifiedType = node.LocalName
				}
			} else if node.PackageName != currentPkgName {
				// Different package - use alias if available, otherwise package name
				qualifiedType = qualifier + "." + node.LocalName
			} else {
				// Same package - no qualification
				qualifiedType = node.LocalName
			}
		}
		structFields = append(structFields, fmt.Sprintf("\t%s %s", fieldName, qualifiedType))
		returnFields = append(returnFields, fmt.Sprintf("\t\t%s: %s,", fieldName, fieldName))
	}

	// Handle empty struct fields case (all Provide, no Inject)
	if len(structFields) == 0 {
		structFields = []string{} // Explicitly empty for clarity
	}

	// Build a set of node keys that are depended upon by other nodes.
	// Used to determine whether a Variable node needs a named assignment (depended upon)
	// or a blank assignment (not depended upon).
	dependedUpon := make(map[string]struct{})
	for _, node := range g.Nodes {
		for _, dep := range node.Dependencies {
			dependedUpon[dep] = struct{}{}
		}
	}

	// Phase 2: Generate initialization code for ALL types (Inject, Provide, and Variable)
	// All types must be initialized in topological order to ensure dependencies are
	// available when needed. Inject and Provide nodes are both exposed as struct fields,
	// while Variable nodes remain as local variables only.
	for _, typeName := range sortedTypes {
		node := g.Nodes[typeName]
		if node == nil {
			continue
		}

		// Determine variable name
		// Use custom name if provided (Named[N] option), otherwise derive from type name
		varName := ""
		if node.IsField {
			varName = fieldNames[typeName]
		} else {
			// Variable types use local variables
			// Use custom name if provided, otherwise derive
			varName = node.Name
			if varName == "" {
				varName = DeriveFieldName(typeName)
			}
		}

		// Variable node: emit expression assignment
		// MUST be checked BEFORE ConstructorName validation to avoid
		// triggering "requires a constructor" error for Variable nodes.
		if node.ExpressionText != "" {
			expressionText := node.ExpressionText
			if !node.IsQualified && node.PackagePath != currentPackage {
				// Local reference from another package — add package qualifier
				qualifier := node.PackageAlias
				if qualifier == "" {
					qualifier = node.PackageName
				}
				expressionText = qualifier + "." + expressionText
			} else if node.IsQualified {
				// Rewrite package qualifiers using collision aliases.
				// ExpressionText is normalized to declared package names (e.g., "os.Stdout"),
				// but when a collision alias is generated (e.g., "os2"), we must rewrite
				// the qualifier in ExpressionText to match.
				expressionText = rewriteExpressionAliases(expressionText, node.ExpressionPkgs, node.ExpressionPkgNames, aliasMap)
			}
			// If no other node depends on this Variable, use blank assignment (_ =)
			// to avoid "declared and not used" errors. Otherwise, use named assignment.
			if _, ok := dependedUpon[typeName]; ok {
				inits = append(inits, fmt.Sprintf("\t%s := %s", varName, expressionText))
			} else {
				inits = append(inits, fmt.Sprintf("\t_ = %s", expressionText))
			}
			continue
		}

		// Provider/Injector node: constructor call (existing logic)
		if len(node.ConstructorName) == 0 {
			return nil, fmt.Errorf("injectable struct %s requires a constructor", typeName)
		}

		// Build constructor call with dependencies
		var args []string
		for _, depTypeName := range node.Dependencies {
			depNode := g.Nodes[depTypeName]
			if depNode == nil {
				continue
			}

			// Resolve dependency variable name
			// Use custom name if provided, otherwise derive
			depVarName := ""
			if depNode.IsField {
				depVarName = fieldNames[depTypeName]
			} else {
				// For Provide types, use custom name if provided
				depVarName = depNode.Name
				if depVarName == "" {
					depVarName = DeriveFieldName(depTypeName)
				}
			}
			args = append(args, depVarName)
		}

		// Determine qualifier (alias takes precedence)
		qualifier := node.PackageAlias
		if qualifier == "" {
			qualifier = node.PackageName
		}

		// Determine package qualifier for constructor call
		var pkgQualifier string
		if node.PackageName == "main" && currentPkgName == "main" {
			// Same or different main - no qualifier (different binaries anyway)
			pkgQualifier = ""
		} else if node.PackageName != currentPkgName {
			pkgQualifier = qualifier + "."
		}
		constructorCall := fmt.Sprintf("%s%s(%s)", pkgQualifier, node.ConstructorName, strings.Join(args, ", "))
		inits = append(inits, fmt.Sprintf("\t%s := %s", varName, constructorCall))
	}

	// Phase 3: Build IIFE code
	// The IIFE (Immediately-Invoked Function Expression) pattern allows us to:
	// 1. Initialize all dependencies in the correct order (via inits)
	// 2. Keep Variable nodes as local variables (not exposed)
	// 3. Return Inject and Provide nodes as fields in the dependency struct
	//
	// Structure:
	//   var dependency = func() struct { <Inject/Provide fields> } {
	//     <initialization code for all types>
	//     return struct { <Inject/Provide fields> } { <Inject/Provide values> }
	//   }()
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// braider:hash:%s\n", hash))
	sb.WriteString("var dependency = func() struct {\n")
	if len(structFields) > 0 {
		sb.WriteString(strings.Join(structFields, "\n"))
		sb.WriteString("\n")
	}
	sb.WriteString("} {\n")
	sb.WriteString(strings.Join(inits, "\n"))
	sb.WriteString("\n\treturn struct {\n")
	sb.WriteString(strings.Join(structFields, "\n"))
	sb.WriteString("\n\t}{\n")
	sb.WriteString(strings.Join(returnFields, "\n"))
	sb.WriteString("\n\t}\n")
	sb.WriteString("}()")

	dependencyVar := sb.String()

	// Format the code
	formatted, err := bg.formatter.FormatCode(dependencyVar)
	if err != nil {
		return nil, fmt.Errorf("failed to format bootstrap code: %w", err)
	}

	return &GeneratedBootstrap{
		DependencyVar: formatted,
		MainReference: "_ = dependency",
		Imports:       imports,
		Hash:          hash,
	}, nil
}

// DetectExistingBootstrap finds an existing bootstrap variable declaration.
func (bg *bootstrapGenerator) DetectExistingBootstrap(pass *analysis.Pass) *ast.GenDecl {
	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}

			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}

				for _, name := range valueSpec.Names {
					if name.Name == "dependency" {
						return genDecl
					}
				}
			}
		}
	}
	return nil
}

// CheckBootstrapCurrent checks if the existing bootstrap code is up-to-date.
func (bg *bootstrapGenerator) CheckBootstrapCurrent(
	pass *analysis.Pass,
	existing *ast.GenDecl,
	g *graph.Graph,
) bool {
	if existing == nil || g == nil {
		return false
	}

	// Extract hash from comment
	existingHash := extractHashFromComments(existing.Doc)
	if existingHash == "" {
		return false
	}

	// Compute current hash
	currentHash := ComputeGraphHash(g)

	return existingHash == currentHash
}

// rewriteExpressionAliases replaces declared package name prefixes in expressionText
// with their collision aliases from aliasMap. This is needed when a package name collision
// causes an alias to be generated (e.g., "os" -> "os2"), but ExpressionText still uses
// the declared name (e.g., "os.Stdout" must become "os2.Stdout").
func rewriteExpressionAliases(
	expressionText string,
	exprPkgs []string,
	exprPkgNames []string,
	aliasMap map[string]string,
) string {
	for i, pkgPath := range exprPkgs {
		if i >= len(exprPkgNames) {
			break
		}
		alias, exists := aliasMap[pkgPath]
		if !exists || alias == "" {
			continue
		}
		declaredName := exprPkgNames[i]
		prefix := declaredName + "."
		if strings.HasPrefix(expressionText, prefix) {
			expressionText = alias + "." + expressionText[len(prefix):]
		}
	}
	return expressionText
}

// extractHashFromComments extracts the hash marker from comment group.
// Expected format: "// braider:hash:abc12345" (with flexible whitespace)
func extractHashFromComments(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	// Use cached regex for flexible matching
	for _, comment := range doc.List {
		if matches := hashCommentRe.FindStringSubmatch(comment.Text); matches != nil {
			return matches[1]
		}
	}
	return ""
}
