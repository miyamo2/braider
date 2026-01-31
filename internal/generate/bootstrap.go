package generate

import (
	"fmt"
	"go/ast"
	"go/token"
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
	DependencyVar string   // IIFE code defining the dependency variable
	MainReference string   // Optional "_ = dependency" statement for main
	Imports       []string // Required import paths
	Hash          string   // Hash marker for idempotency
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

	// Collect imports
	currentPackage := pass.Pkg.Path()
	imports := CollectImports(g, currentPackage)

	// Build struct fields and initialization code
	// Note: If all nodes are Provide (IsField=false), structFields will be empty
	var structFields []string
	var inits []string
	var returnFields []string

	// Track field names and their types
	fieldNames := make(map[string]string) // typeName -> fieldName

	// Phase 1: Build struct fields for Inject structs only
	// Inject structs (IsField=true) become fields in the dependency struct that is returned
	// to the caller. These are the components that the application can access and use.
	// Provide structs (IsField=false) are local variables within the IIFE and are not
	// exposed to the caller - they exist only to satisfy dependencies of Inject structs.
	for _, typeName := range sortedTypes {
		node := g.Nodes[typeName]
		if node == nil {
			continue
		}

		// Only include Inject structs as fields (IsField = true)
		if !node.IsField {
			continue
		}

		fieldName := DeriveFieldName(typeName)
		fieldNames[typeName] = fieldName

		// Add struct field
		structFields = append(structFields, fmt.Sprintf("\t%s %s", fieldName, node.LocalName))
		returnFields = append(returnFields, fmt.Sprintf("\t\t%s: %s,", fieldName, fieldName))
	}

	// Handle empty struct fields case (all Provide, no Inject)
	if len(structFields) == 0 {
		structFields = []string{} // Explicitly empty for clarity
	}

	// Phase 2: Generate initialization code for ALL types (both Inject and Provide)
	// Even though Provide structs are not included in the returned dependency struct,
	// they must still be initialized because Inject structs may depend on them.
	// The topological sort ensures that all dependencies are initialized before
	// the types that depend on them.
	//
	// Example: If UserService (Inject) depends on UserRepository (Provide),
	// the IIFE will initialize UserRepository first, then pass it to UserService's
	// constructor. UserRepository exists only as a local variable and is not exposed.
	for _, typeName := range sortedTypes {
		node := g.Nodes[typeName]
		if node == nil {
			continue
		}

		// Determine variable name
		varName := ""
		if node.IsField {
			varName = fieldNames[typeName]
		} else {
			// Provide types use local variables
			varName = DeriveFieldName(typeName)
		}

		// Build constructor call with dependencies
		var args []string
		for _, depTypeName := range node.Dependencies {
			depNode := g.Nodes[depTypeName]
			if depNode == nil {
				continue
			}

			depVarName := ""
			if depNode.IsField {
				depVarName = fieldNames[depTypeName]
			} else {
				depVarName = DeriveFieldName(depTypeName)
			}
			args = append(args, depVarName)
		}

		constructorCall := fmt.Sprintf("%s(%s)", node.ConstructorName, strings.Join(args, ", "))
		inits = append(inits, fmt.Sprintf("\t%s := %s", varName, constructorCall))
	}

	// Phase 3: Build IIFE code
	// The IIFE (Immediately-Invoked Function Expression) pattern allows us to:
	// 1. Initialize all dependencies in the correct order (via inits)
	// 2. Keep Provide structs as local variables (not exposed)
	// 3. Return only Inject structs as fields in the dependency struct
	//
	// Structure:
	//   var dependency = func() struct { <Inject fields> } {
	//     <initialization code for all types>
	//     return struct { <Inject fields> } { <Inject values> }
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
