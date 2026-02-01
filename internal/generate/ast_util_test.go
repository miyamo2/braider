package generate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestIsDependencyReferenced(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name:   "nil function",
			source: "",
			want:   false,
		},
		{
			name: "empty function body",
			source: `package main
func main() {}`,
			want: false,
		},
		{
			name: "dependency not referenced",
			source: `package main
import "fmt"
func main() {
	fmt.Println("hello")
}`,
			want: false,
		},
		{
			name: "dependency referenced via selector",
			source: `package main
func main() {
	_ = dependency.field
}`,
			want: true,
		},
		{
			name: "dependency passed to function",
			source: `package main
func main() {
	process(dependency)
}`,
			want: true,
		},
		{
			name: "dependency assigned to variable",
			source: `package main
func main() {
	dep := dependency
	_ = dep
}`,
			want: true,
		},
		{
			name: "dependency in expression",
			source: `package main
func main() {
	if dependency != nil {
		// do something
	}
}`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mainFunc *ast.FuncDecl
			if tt.source != "" {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "", tt.source, 0)
				if err != nil {
					t.Fatalf("Failed to parse function: %v", err)
				}
				for _, decl := range file.Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok {
						mainFunc = fn
						break
					}
				}
			}

			got := IsDependencyReferenced(mainFunc)
			if got != tt.want {
				t.Errorf("IsDependencyReferenced() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDependencyReferenced_BlankAssignment(t *testing.T) {
	// The implementation now correctly distinguishes between
	// "_ = dependency" (blank assignment) and actual usage.
	// Blank assignments should not be considered as references.
	source := `package main
func main() {
	_ = dependency
}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse function: %v", err)
	}

	var mainFunc *ast.FuncDecl
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			mainFunc = fn
			break
		}
	}

	got := IsDependencyReferenced(mainFunc)
	// Should return false for blank assignment
	if got != false {
		t.Errorf("IsDependencyReferenced() = %v, want false (blank assignment excluded)", got)
	}
}
