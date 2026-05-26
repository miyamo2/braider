package lsp

import (
	"encoding/json"
	"fmt"
	"go/token"
	"sort"
)

// handleCompletion processes a textDocument/completion request.
// It provides type completions for the T argument in Provide[T] and Injectable[T].
func (s *Server) handleCompletion(id any, rawParams json.RawMessage) error {
	var params CompletionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return s.sendError(id, ErrInvalidParams, "invalid completion params")
	}

	filePath := uriToFilePath(params.TextDocument.URI)
	fset := token.NewFileSet()
	overlay := s.overlayForFile(filePath)

	pkgs, err := loadPackageForFile(fset, filePath, overlay)
	if err != nil || len(pkgs) == 0 {
		// Return empty list gracefully – analysis failure should not crash the server.
		return s.sendResult(id, &CompletionList{IsIncomplete: false, Items: nil})
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 && pkg.Syntax == nil {
		return s.sendResult(id, &CompletionList{IsIncomplete: false, Items: nil})
	}

	// Find the syntax file matching the request URI.
	var targetFile = findSyntaxFile(pkg, filePath)
	if targetFile == nil {
		return s.sendResult(id, &CompletionList{IsIncomplete: false, Items: nil})
	}

	ctx := findCursorContext(fset, targetFile, params.Position.Line, params.Position.Character)
	if ctx.Kind == contextNone {
		return s.sendResult(id, &CompletionList{IsIncomplete: false, Items: nil})
	}

	// Collect all types visible from this package.
	allTypes := collectExportedTypes(pkg)

	// Build a set of already-registered type names (from existing registrations in the
	// workspace) so we can surface them first.
	registeredTypes := s.collectRegisteredTypeNames()

	items := make([]CompletionItem, 0, len(allTypes))
	for _, et := range allTypes {
		qualName := et.QualifiedName(pkg.PkgPath)
		isRegistered := registeredTypes[et.PackagePath+"."+et.LocalName]

		kind := CompletionItemKindStruct
		if et.IsInterface {
			kind = CompletionItemKindInterface
		}

		label := qualName
		sortPrefix := "b_" // unregistered second
		if isRegistered {
			sortPrefix = "a_" // registered types first
		}

		detail := buildCompletionDetail(ctx.Kind, et, isRegistered)

		items = append(items, CompletionItem{
			Label:         label,
			Kind:          kind,
			Detail:        detail,
			Documentation: fmt.Sprintf("Package: %s", et.PackagePath),
			InsertText:    qualName,
			SortText:      sortPrefix + label,
		})
	}

	// Stable sort by SortText so registered entries appear first.
	sort.Slice(items, func(i, j int) bool {
		return items[i].SortText < items[j].SortText
	})

	return s.sendResult(id, &CompletionList{
		IsIncomplete: false,
		Items:        items,
	})
}

// buildCompletionDetail builds a human-readable detail line for a completion item.
func buildCompletionDetail(kind typeContext, et exportedType, isRegistered bool) string {
	switch kind {
	case contextProvide:
		if isRegistered {
			return fmt.Sprintf("Provide[%s] — already registered", et.LocalName)
		}
		return fmt.Sprintf("Register %s as a provider", et.LocalName)
	case contextInject:
		if isRegistered {
			return fmt.Sprintf("Injectable[%s] — already registered", et.LocalName)
		}
		return fmt.Sprintf("Register %s as injectable", et.LocalName)
	case contextVariable:
		if isRegistered {
			return fmt.Sprintf("Variable[%s] — already registered", et.LocalName)
		}
		return fmt.Sprintf("Register %s as a variable", et.LocalName)
	}
	return et.LocalName
}

// collectRegisteredTypeNames returns a set of fully qualified type names that
// appear in the server's cached registries.
func (s *Server) collectRegisteredTypeNames() map[string]bool {
	result := make(map[string]bool)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for typeName := range s.providers {
		result[typeName] = true
	}
	for typeName := range s.injectors {
		result[typeName] = true
	}
	for typeName := range s.variables {
		result[typeName] = true
	}
	return result
}
