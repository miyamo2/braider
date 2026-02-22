// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package driverutil defines implementation helper functions for
// analysis drivers such as unitchecker, {single,multi}checker, and
// analysistest.
package driverutil

// This file defines the -fix logic common to unitchecker and
// {single,multi}checker.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"maps"
	"os"
	"sort"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/miyamo2/braider/internal/x/tools/diff"
	"github.com/miyamo2/braider/internal/x/tools/free"
)

// FixAction abstracts a checker action (running one analyzer on one
// package) for the purposes of applying its diagnostics' fixes.
type FixAction struct {
	Name         string         // e.g. "analyzer@package"
	Pkg          *types.Package // (for import removal)
	Files        []*ast.File
	FileSet      *token.FileSet
	ReadFileFunc ReadFileFunc
	Diagnostics  []analysis.Diagnostic
}

// ApplyFixes attempts to apply the first suggested fix associated
// with each diagnostic reported by the specified actions.
// All fixes must have been validated by [ValidateFixes].
//
// Each fix is treated as an independent change; fixes are merged in
// an arbitrary deterministic order as if by a three-way diff tool
// such as the UNIX diff3 command or 'git merge'. Any fix that cannot be
// cleanly merged is discarded, in which case the final summary tells
// the user to re-run the tool.
//
// applyFixes returns success if all fixes are valid, could be cleanly
// merged, and the corresponding files were successfully updated.
//
// If printDiff (from the -diff flag) is set, instead of updating the
// files it display the final patch composed of all the cleanly merged
// fixes.
func ApplyFixes(actions []FixAction, writeFile func(filename string, content []byte) error, printDiff, verbose bool) error {
	generated := make(map[*token.File]bool)

	// Select fixes to apply.
	type fixact struct {
		fix *analysis.SuggestedFix
		act FixAction
	}
	var fixes []*fixact
	for _, act := range actions {
		for _, file := range act.Files {
			tokFile := act.FileSet.File(file.FileStart)
			if _, seen := generated[tokFile]; !seen {
				generated[tokFile] = ast.IsGenerated(file)
			}
		}

		for _, diag := range act.Diagnostics {
			for i := range diag.SuggestedFixes {
				fix := &diag.SuggestedFixes[i]
				if i == 0 {
					fixes = append(fixes, &fixact{fix, act})
				} else {
					log.Printf("%s: ignoring alternative fix %q", act.Name, fix.Message)
				}
			}
		}
	}

	// Read file content on demand, from the virtual
	// file system that fed the analyzer (see #62292).
	baselineContent := make(map[string][]byte)
	getBaseline := func(readFile ReadFileFunc, filename string) ([]byte, error) {
		content, ok := baselineContent[filename]
		if !ok {
			var err error
			content, err = readFile(filename)
			if err != nil {
				return nil, err
			}
			baselineContent[filename] = content
		}
		return content, nil
	}

	// Apply each fix, updating the current state
	// only if the entire fix can be cleanly merged.
	var (
		accumulatedEdits = make(map[string][]diff.Edit)
		filePkgs         = make(map[string]*types.Package)

		goodFixes    = 0
		skippedFixes = 0
	)
fixloop:
	for _, fixact := range fixes {
		// Skip a fix if any of its edits touch a generated file.
		for _, edit := range fixact.fix.TextEdits {
			file := fixact.act.FileSet.File(edit.Pos)
			if generated[file] {
				skippedFixes++
				continue fixloop
			}
		}

		// Convert analysis.TextEdits to diff.Edits, grouped by file.
		fileEdits := make(map[string][]diff.Edit)
		for _, edit := range fixact.fix.TextEdits {
			file := fixact.act.FileSet.File(edit.Pos)

			filePkgs[file.Name()] = fixact.act.Pkg

			baseline, err := getBaseline(fixact.act.ReadFileFunc, file.Name())
			if err != nil {
				log.Printf("skipping fix to file %s: %v", file.Name(), err)
				continue fixloop
			}

			if file.Size() != len(baseline) {
				return fmt.Errorf("concurrent file modification detected in file %s (size changed from %d -> %d bytes); aborting fix",
					file.Name(), file.Size(), len(baseline))
			}

			fileEdits[file.Name()] = append(fileEdits[file.Name()], diff.Edit{
				Start: file.Offset(edit.Pos),
				End:   file.Offset(edit.End),
				New:   string(edit.NewText),
			})
		}

		// Apply each set of edits by merging atop
		// the previous accumulated state.
		after := make(map[string][]diff.Edit)
		for file, edits := range fileEdits {
			if prev := accumulatedEdits[file]; len(prev) > 0 {
				merged, ok := diff.Merge(prev, edits)
				if !ok {
					continue fixloop // conflict
				}
				edits = merged
			}
			after[file] = edits
		}

		// The entire fix applied cleanly; commit it.
		goodFixes++
		maps.Copy(accumulatedEdits, after)
	}
	badFixes := len(fixes) - goodFixes - skippedFixes

	// Show diff or update files to final state.
	var files []string
	for file := range accumulatedEdits {
		files = append(files, file)
	}
	sort.Strings(files)
	var filesUpdated, totalFiles int
	for _, file := range files {
		edits := accumulatedEdits[file]
		if len(edits) == 0 {
			continue
		}

		// Apply accumulated fixes.
		baseline := baselineContent[file]
		final, err := diff.ApplyBytes(baseline, edits)
		if err != nil {
			log.Fatalf("internal error in diff.ApplyBytes: %v", err)
		}

		// Attempt to format each file.
		if formatted, err := FormatSourceRemoveImports(filePkgs[file], final); err == nil {
			final = formatted
		}

		if printDiff {
			unified := diff.Unified(file+" (old)", file+" (new)", string(baseline), string(final))
			os.Stdout.WriteString(unified)

		} else {
			totalFiles++
			if err := writeFile(file, final); err != nil {
				log.Println(err)
				continue
			}
			filesUpdated++
		}
	}

	if badFixes > 0 || filesUpdated < totalFiles {
		if printDiff {
			return fmt.Errorf("%d of %s skipped (e.g. due to conflicts)",
				badFixes,
				plural(len(fixes), "fix", "fixes"))
		} else {
			return fmt.Errorf("applied %d of %s; %s updated. (Re-run the command to apply more.)",
				goodFixes,
				plural(len(fixes), "fix", "fixes"),
				plural(filesUpdated, "file", "files"))
		}
	}

	if verbose {
		if skippedFixes > 0 {
			log.Printf("skipped %s that would edit generated files",
				plural(skippedFixes, "fix", "fixes"))
		}
		log.Printf("applied %s, updated %s",
			plural(len(fixes), "fix", "fixes"),
			plural(filesUpdated, "file", "files"))
	}

	return nil
}

// FormatSourceRemoveImports is a variant of [format.Source] that
// removes imports that became redundant when fixes were applied.
func FormatSourceRemoveImports(pkg *types.Package, src []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixed.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	ast.SortImports(fset, file)

	removeUnneededImports(fset, pkg, file)

	const printerNormalizeNumbers = 1 << 30
	cfg := &printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent | printerNormalizeNumbers,
		Tabwidth: 8,
	}
	var buf bytes.Buffer
	if err := cfg.Fprint(&buf, fset, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// removeUnneededImports removes import specs that are not referenced
// within the fixed file.
func removeUnneededImports(fset *token.FileSet, pkg *types.Package, file *ast.File) {
	packageNames := make(map[string]string)
	for _, imp := range pkg.Imports() {
		packageNames[imp.Path()] = imp.Name()
	}

	freenames := make(map[string]bool)
	for _, decl := range file.Decls {
		if decl, ok := decl.(*ast.GenDecl); ok && decl.Tok == token.IMPORT {
			continue
		}

		const includeComplitIdents = false
		maps.Copy(freenames, free.Names(decl, includeComplitIdents))
	}

	var deletions []func()
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		explicit := ""
		if spec.Name != nil {
			explicit = spec.Name.Name
		}
		name := explicit
		if name == "" {
			name = packageNames[path]
		}
		switch name {
		case "":
			continue
		case ".":
			continue
		case "_":
			continue
		}
		if !freenames[name] {
			deletions = append(deletions, func() {
				astutil.DeleteNamedImport(fset, file, explicit, path)
			})
		}
	}

	for _, del := range deletions {
		del()
	}
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	} else {
		return fmt.Sprintf("%d %s", n, plural)
	}
}
