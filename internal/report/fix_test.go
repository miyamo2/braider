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

