package report_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
	"github.com/miyamo2/braider/internal/report"
	"golang.org/x/tools/go/analysis"
)

func TestSuggestedFixBuilder_BuildConstructorFix_Insert(t *testing.T) {
	src := `package test

type MyService struct {
	repo Repository
}

type Repository interface{}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
	}

	// Find struct
	var genDecl *ast.GenDecl
	var typeSpec *ast.TypeSpec
	var structType *ast.StructType
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if ts.Name.Name == "MyService" {
						genDecl = gd
						typeSpec = ts
						structType = ts.Type.(*ast.StructType)
						break
					}
				}
			}
		}
	}

	candidate := detect.ConstructorCandidate{
		TypeSpec:            typeSpec,
		StructType:          structType,
		GenDecl:             genDecl,
		ExistingConstructor: nil, // No existing constructor
	}

	constructor := &generate.GeneratedConstructor{
		FuncName:   "NewMyService",
		StructName: "MyService",
		Code:       "func NewMyService(repo Repository) *MyService {\n\treturn &MyService{repo: repo}\n}\n",
	}

	builder := report.NewSuggestedFixBuilder()
	fix := builder.BuildConstructorFix(pass, candidate, constructor)

	// Verify fix message
	if fix.Message != "generate constructor for MyService" {
		t.Errorf("fix.Message = %q, want %q", fix.Message, "generate constructor for MyService")
	}

	// Verify TextEdit
	if len(fix.TextEdits) != 1 {
		t.Fatalf("expected 1 TextEdit, got %d", len(fix.TextEdits))
	}

	edit := fix.TextEdits[0]

	// For insertion, Pos should equal End
	if edit.Pos != edit.End {
		t.Error("for insertion, TextEdit.Pos should equal TextEdit.End")
	}

	// Verify position is after struct definition
	structEnd := genDecl.End()
	if edit.Pos != structEnd {
		t.Errorf("TextEdit.Pos = %d, want %d (struct end)", edit.Pos, structEnd)
	}

	// Verify NewText contains constructor code with blank line prefix
	newText := string(edit.NewText)
	if !strings.HasPrefix(newText, "\n\n") {
		t.Error("NewText should start with blank line separator")
	}
	if !strings.Contains(newText, "NewMyService") {
		t.Error("NewText should contain constructor function")
	}
}

func TestSuggestedFixBuilder_BuildConstructorFix_Replace(t *testing.T) {
	src := `package test

type MyService struct {
	repo Repository
}

// OldConstructor is old.
func NewMyService(repo Repository) *MyService {
	return &MyService{repo: repo}
}

type Repository interface{}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
	}

	// Find struct and existing constructor
	var genDecl *ast.GenDecl
	var typeSpec *ast.TypeSpec
	var structType *ast.StructType
	var existingCtor *ast.FuncDecl

	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if ts.Name.Name == "MyService" {
						genDecl = gd
						typeSpec = ts
						structType = ts.Type.(*ast.StructType)
					}
				}
			}
		}
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "NewMyService" {
				existingCtor = fn
			}
		}
	}

	candidate := detect.ConstructorCandidate{
		TypeSpec:            typeSpec,
		StructType:          structType,
		GenDecl:             genDecl,
		ExistingConstructor: existingCtor,
	}

	constructor := &generate.GeneratedConstructor{
		FuncName:   "NewMyService",
		StructName: "MyService",
		Code:       "// NewMyService is a constructor for MyService.\nfunc NewMyService(repo Repository, logger Logger) *MyService {\n\treturn &MyService{repo: repo, logger: logger}\n}\n",
	}

	builder := report.NewSuggestedFixBuilder()
	fix := builder.BuildConstructorFix(pass, candidate, constructor)

	// Verify fix message
	if fix.Message != "regenerate constructor for MyService" {
		t.Errorf("fix.Message = %q, want %q", fix.Message, "regenerate constructor for MyService")
	}

	// Verify TextEdit
	if len(fix.TextEdits) != 1 {
		t.Fatalf("expected 1 TextEdit, got %d", len(fix.TextEdits))
	}

	edit := fix.TextEdits[0]

	// For replacement, Pos should be less than End
	if edit.Pos >= edit.End {
		t.Error("for replacement, TextEdit.Pos should be less than TextEdit.End")
	}

	// Verify position includes doc comment
	if existingCtor.Doc != nil {
		if edit.Pos != existingCtor.Doc.Pos() {
			t.Errorf("TextEdit.Pos should include doc comment")
		}
	}

	// Verify NewText contains new constructor code
	newText := string(edit.NewText)
	if !strings.Contains(newText, "NewMyService") {
		t.Error("NewText should contain constructor function")
	}
}

func TestSuggestedFixBuilder_BlankLineIncluded(t *testing.T) {
	src := `package test

type MyService struct{}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
	}

	// Find struct
	var genDecl *ast.GenDecl
	var typeSpec *ast.TypeSpec
	var structType *ast.StructType
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if ts.Name.Name == "MyService" {
						genDecl = gd
						typeSpec = ts
						structType = ts.Type.(*ast.StructType)
					}
				}
			}
		}
	}

	candidate := detect.ConstructorCandidate{
		TypeSpec:   typeSpec,
		StructType: structType,
		GenDecl:    genDecl,
	}

	constructor := &generate.GeneratedConstructor{
		FuncName:   "NewMyService",
		StructName: "MyService",
		Code:       "func NewMyService() *MyService { return &MyService{} }\n",
	}

	builder := report.NewSuggestedFixBuilder()
	fix := builder.BuildConstructorFix(pass, candidate, constructor)

	edit := fix.TextEdits[0]
	newText := string(edit.NewText)

	// Should have blank line between struct and constructor
	if !strings.HasPrefix(newText, "\n\n") {
		t.Errorf("NewText should start with blank line, got: %q", newText[:min(10, len(newText))])
	}
}

func TestSuggestedFixBuilder_BuildBootstrapFix(t *testing.T) {
	tests := []struct {
		name                 string
		src                  string
		dependencyReferenced bool
		expectedEditCount    int
	}{
		{
			name: "inserts bootstrap and main reference",
			src: `package main

func main() {
}
`,
			dependencyReferenced: false,
			expectedEditCount:    2, // dependency var + main ref
		},
		{
			name: "inserts bootstrap only when dependency already referenced",
			src: `package main

func main() {
	println(dependency)
}
`,
			dependencyReferenced: true,
			expectedEditCount:    1, // only dependency var
		},
		{
			name: "empty main function gets reference",
			src: `package main

func main() {}
`,
			dependencyReferenced: false,
			expectedEditCount:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			pass := &analysis.Pass{
				Fset:  fset,
				Files: []*ast.File{file},
			}

			// Find main function
			var mainFunc *ast.FuncDecl
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
					mainFunc = fn
					break
				}
			}

			app := &detect.AppAnnotation{
				File: file,
				Pos:  token.Pos(1),
			}

			bootstrap := &generate.GeneratedBootstrap{
				DependencyVar: "var dependency = struct{}{}",
				MainReference: "_ = dependency",
				Hash:          "abc123",
				Imports:       []string{},
			}

			builder := report.NewSuggestedFixBuilder()
			fix := builder.BuildBootstrapFix(pass, app, bootstrap, mainFunc)

			if fix.Message != "generate bootstrap code" {
				t.Errorf("Message = %q, want %q", fix.Message, "generate bootstrap code")
			}

			if len(fix.TextEdits) != tt.expectedEditCount {
				t.Errorf("len(TextEdits) = %d, want %d", len(fix.TextEdits), tt.expectedEditCount)
			}

			// Verify dependency variable edit exists
			dependencyVarFound := false
			for _, edit := range fix.TextEdits {
				if strings.Contains(string(edit.NewText), "var dependency") {
					dependencyVarFound = true
					break
				}
			}
			if !dependencyVarFound {
				t.Error("dependency variable edit not found")
			}

			// Verify main reference edit based on expectation
			mainRefFound := false
			for _, edit := range fix.TextEdits {
				if strings.Contains(string(edit.NewText), "_ = dependency") {
					mainRefFound = true
					break
				}
			}
			if !tt.dependencyReferenced && !mainRefFound {
				t.Error("main reference edit not found when expected")
			}
			if tt.dependencyReferenced && mainRefFound {
				t.Error("main reference edit found when not expected")
			}
		})
	}
}

func TestSuggestedFixBuilder_BuildBootstrapReplacementFix(t *testing.T) {
	src := `package main

// braider:hash:abc123
var dependency = struct{}{}

func main() {
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
	}

	// Find existing dependency variable and main function
	var existingDecl *ast.GenDecl
	var mainFunc *ast.FuncDecl
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
			// Check if it's the dependency variable
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range vs.Names {
						if name.Name == "dependency" {
							existingDecl = gd
							break
						}
					}
				}
			}
		}
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "main" {
			mainFunc = fn
		}
	}

	if existingDecl == nil {
		t.Fatal("existing dependency declaration not found")
	}
	if mainFunc == nil {
		t.Fatal("main function not found")
	}

	bootstrap := &generate.GeneratedBootstrap{
		DependencyVar: "// braider:hash:def456\nvar dependency = struct{ NewField string }{}",
		MainReference: "_ = dependency",
		Hash:          "def456",
		Imports:       []string{},
	}

	builder := report.NewSuggestedFixBuilder()
	fix := builder.BuildBootstrapReplacementFix(pass, existingDecl, bootstrap, mainFunc)

	if fix.Message != "update bootstrap code" {
		t.Errorf("Message = %q, want %q", fix.Message, "update bootstrap code")
	}

	// Should have at least 2 edits: replace dependency + add main ref
	if len(fix.TextEdits) < 2 {
		t.Errorf("len(TextEdits) = %d, want at least 2", len(fix.TextEdits))
	}

	// Verify dependency replacement edit (Pos < End for replacement)
	replacementFound := false
	for _, edit := range fix.TextEdits {
		if edit.Pos < edit.End && strings.Contains(string(edit.NewText), "var dependency") {
			replacementFound = true
			break
		}
	}
	if !replacementFound {
		t.Error("dependency replacement edit not found")
	}
}
