package lsp

import (
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

// handleHover processes a textDocument/hover request.
// When the cursor is on the T argument of Provide[T] / Injectable[T] / Variable[T]
// it shows which provider/injector/variable binding wins for that type.
func (s *Server) handleHover(id any, rawParams json.RawMessage) error {
	var params HoverParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return s.sendError(id, ErrInvalidParams, "invalid hover params")
	}

	filePath := uriToFilePath(params.TextDocument.URI)
	fset := token.NewFileSet()
	overlay := s.overlayForFile(filePath)

	pkgs, err := loadPackageForFile(fset, filePath, overlay)
	if err != nil || len(pkgs) == 0 {
		return s.sendResult(id, nil)
	}

	pkg := pkgs[0]
	if pkg.Syntax == nil || pkg.TypesInfo == nil {
		return s.sendResult(id, nil)
	}

	targetFile := findSyntaxFile(pkg, filePath)
	if targetFile == nil {
		return s.sendResult(id, nil)
	}

	// Find the type at the cursor position via type info.
	t, ok := typeAtPosition(fset, targetFile, pkg.TypesInfo, params.Position.Line, params.Position.Character)
	if !ok {
		return s.sendResult(id, nil)
	}

	// Dereference pointer types for lookup.
	baseType := t
	if ptr, ok2 := baseType.(*types.Pointer); ok2 {
		baseType = ptr.Elem()
	}

	named, ok := baseType.(*types.Named)
	if !ok {
		return s.sendResult(id, nil)
	}

	typeName := fullyQualifiedTypeName(named)
	binding := s.lookupBinding(typeName)
	if binding == nil {
		return s.sendResult(id, &HoverResult{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: fmt.Sprintf("**%s**\n\nNo braider binding registered for this type.", typeName),
			},
		})
	}

	return s.sendResult(id, &HoverResult{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: buildHoverContent(binding),
		},
	})
}

// buildHoverContent renders a Markdown description of a resolved binding.
func buildHoverContent(b *resolvedBinding) string {
	var sb strings.Builder
	switch b.Kind {
	case "provide":
		sb.WriteString(fmt.Sprintf("**Provide** binding\n\n"))
		sb.WriteString(fmt.Sprintf("- Type: `%s`\n", b.TypeName))
		sb.WriteString(fmt.Sprintf("- Constructor: `%s`\n", b.ConstructorFn))
		sb.WriteString(fmt.Sprintf("- Package: `%s`\n", b.PackagePath))
	case "inject":
		sb.WriteString(fmt.Sprintf("**Injectable** binding\n\n"))
		sb.WriteString(fmt.Sprintf("- Type: `%s`\n", b.TypeName))
		sb.WriteString(fmt.Sprintf("- Constructor: `%s`\n", b.ConstructorFn))
		sb.WriteString(fmt.Sprintf("- Package: `%s`\n", b.PackagePath))
	case "variable":
		sb.WriteString(fmt.Sprintf("**Variable** binding\n\n"))
		sb.WriteString(fmt.Sprintf("- Type: `%s`\n", b.TypeName))
		sb.WriteString(fmt.Sprintf("- Package: `%s`\n", b.PackagePath))
	}
	if b.Name != "" {
		sb.WriteString(fmt.Sprintf("- Named: `%s`\n", b.Name))
	}
	return sb.String()
}

// lookupBinding searches the server's cached registries for the given type name.
// Returns nil when no binding exists.
func (s *Server) lookupBinding(typeName string) *resolvedBinding {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if inner, ok := s.providers[typeName]; ok {
		// Prefer the unnamed (default) binding.
		if info, exists := inner[""]; exists {
			return &resolvedBinding{
				Kind:          "provide",
				ConstructorFn: info.ConstructorName,
				PackagePath:   info.PackagePath,
				TypeName:      typeName,
				Name:          info.Name,
			}
		}
		// Fall back to any named binding.
		for _, info := range inner {
			return &resolvedBinding{
				Kind:          "provide",
				ConstructorFn: info.ConstructorName,
				PackagePath:   info.PackagePath,
				TypeName:      typeName,
				Name:          info.Name,
			}
		}
	}

	if inner, ok := s.injectors[typeName]; ok {
		if info, exists := inner[""]; exists {
			return &resolvedBinding{
				Kind:          "inject",
				ConstructorFn: info.ConstructorName,
				PackagePath:   info.PackagePath,
				TypeName:      typeName,
				Name:          info.Name,
			}
		}
		for _, info := range inner {
			return &resolvedBinding{
				Kind:          "inject",
				ConstructorFn: info.ConstructorName,
				PackagePath:   info.PackagePath,
				TypeName:      typeName,
				Name:          info.Name,
			}
		}
	}

	if inner, ok := s.variables[typeName]; ok {
		if info, exists := inner[""]; exists {
			return &resolvedBinding{
				Kind:        "variable",
				PackagePath: info.PackagePath,
				TypeName:    typeName,
				Name:        info.Name,
			}
		}
		for _, info := range inner {
			return &resolvedBinding{
				Kind:        "variable",
				PackagePath: info.PackagePath,
				TypeName:    typeName,
				Name:        info.Name,
			}
		}
	}

	return nil
}

// fullyQualifiedTypeName builds the braider-style "pkgpath.LocalName" key.
func fullyQualifiedTypeName(named *types.Named) string {
	obj := named.Obj()
	if obj.Pkg() == nil {
		return obj.Name()
	}
	return obj.Pkg().Path() + "." + obj.Name()
}
