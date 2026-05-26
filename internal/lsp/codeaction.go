package lsp

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strings"
)

// handleCodeAction processes a textDocument/codeAction request.
// When the cursor is on an exported constructor function (no receiver, returns a named type)
// it offers a "Register with annotation.Provide" code action that inserts the
// appropriate `var _ = annotation.Provide[provide.Default](NewFoo)` declaration.
func (s *Server) handleCodeAction(id any, rawParams json.RawMessage) error {
	var params CodeActionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return s.sendError(id, ErrInvalidParams, "invalid codeAction params")
	}

	filePath := uriToFilePath(params.TextDocument.URI)
	fset := token.NewFileSet()
	overlay := s.overlayForFile(filePath)

	pkgs, err := loadPackageForFile(fset, filePath, overlay)
	if err != nil || len(pkgs) == 0 {
		return s.sendResult(id, []CodeAction{})
	}

	pkg := pkgs[0]
	if pkg.Syntax == nil || pkg.TypesInfo == nil {
		return s.sendResult(id, []CodeAction{})
	}

	targetFile := findSyntaxFile(pkg, filePath)
	if targetFile == nil {
		return s.sendResult(id, []CodeAction{})
	}

	fnDecl, returnTypeName, ok := constructorAtPosition(
		fset, targetFile, pkg.TypesInfo,
		params.Range.Start.Line, params.Range.Start.Character,
	)
	if !ok {
		return s.sendResult(id, []CodeAction{})
	}

	fnName := fnDecl.Name.Name

	// Read the current file content (from overlay or disk) to check existing imports.
	var fileContent string
	s.mu.RLock()
	uri := filePathToURI(filePath)
	fileContent = s.openFiles[uri]
	s.mu.RUnlock()
	if fileContent == "" {
		// Fall back to reading from disk.
		if content, err := readFile(filePath); err == nil {
			fileContent = content
		}
	}

	// Determine package aliases used in the file.
	annotationAlias, provideAlias := effectiveImportAliases(targetFile)

	// Build the Provide registration snippet.
	provideSnippet := fmt.Sprintf("var _ = %s.Provide[%s.Default](%s)\n",
		annotationAlias, provideAlias, fnName)

	// Insert the snippet just before the constructor declaration.
	insertPos := fset.Position(fnDecl.Pos())
	insertLine := insertPos.Line - 1 // 0-based
	insertRange := Range{
		Start: Position{Line: insertLine, Character: 0},
		End:   Position{Line: insertLine, Character: 0},
	}

	var allEdits []TextEdit

	// Add missing import statements if needed.
	importEdit := buildMissingImportEdit(fset, targetFile, fileContent, annotationAlias, provideAlias)
	if importEdit != nil {
		allEdits = append(allEdits, *importEdit)
	}

	allEdits = append(allEdits, TextEdit{
		Range:   insertRange,
		NewText: provideSnippet,
	})

	action := CodeAction{
		Title: fmt.Sprintf("Register %s with annotation.Provide[%s]", fnName, returnTypeName),
		Kind:  "refactor.rewrite",
		Edit: &WorkspaceEdit{
			Changes: map[string][]TextEdit{
				params.TextDocument.URI: allEdits,
			},
		},
	}

	return s.sendResult(id, []CodeAction{action})
}

// effectiveImportAliases returns the local aliases used in the file for the
// annotation and provide packages. Falls back to "annotation" / "provide".
func effectiveImportAliases(f *ast.File) (annotationAlias, provideAlias string) {
	annotationAlias = "annotation"
	provideAlias = "provide"
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			// Derive last path component.
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		switch path {
		case annotationProvidePkg:
			annotationAlias = alias
		case "github.com/miyamo2/braider/pkg/annotation/provide":
			provideAlias = alias
		}
	}
	return
}

// buildMissingImportEdit returns a TextEdit that adds missing import lines for
// the annotation and/or provide packages, or nil if both are already present.
func buildMissingImportEdit(fset *token.FileSet, f *ast.File, content, annotationAlias, provideAlias string) *TextEdit {
	needAnnotation := !importedInFile(f, annotationProvidePkg)
	needProvide := !importedInFile(f, "github.com/miyamo2/braider/pkg/annotation/provide")

	if !needAnnotation && !needProvide {
		return nil
	}

	var lines []string
	if needAnnotation {
		lines = append(lines, fmt.Sprintf("\t%q", annotationProvidePkg))
	}
	if needProvide {
		lines = append(lines, fmt.Sprintf("\t%q", "github.com/miyamo2/braider/pkg/annotation/provide"))
	}

	// Try to append inside the existing import block.
	if f.Imports != nil && len(f.Imports) > 0 {
		lastImport := f.Imports[len(f.Imports)-1]
		pos := fset.Position(lastImport.End())
		insertLine := pos.Line // 0-based: insert after last import line
		return &TextEdit{
			Range: Range{
				Start: Position{Line: insertLine, Character: 0},
				End:   Position{Line: insertLine, Character: 0},
			},
			NewText: strings.Join(lines, "\n") + "\n",
		}
	}

	// No existing imports: insert a new import block after the package clause.
	pkgEnd := fset.Position(f.Name.End())
	insertLine := pkgEnd.Line // 0-based: insert on the line after the package declaration
	newText := "\nimport (\n" + strings.Join(lines, "\n") + "\n)\n"
	return &TextEdit{
		Range: Range{
			Start: Position{Line: insertLine, Character: 0},
			End:   Position{Line: insertLine, Character: 0},
		},
		NewText: newText,
	}
}

// importedInFile reports whether importPath is already imported in f.
func importedInFile(f *ast.File, importPath string) bool {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == importPath {
			return true
		}
	}
	return false
}

// readFile reads a file from disk and returns its content as a string.
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}
