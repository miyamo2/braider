package detect_test

import (
	"go/ast"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
)

func TestConstructorAnalyzer_ExtractDependencies(t *testing.T) {
	tests := []struct {
		name         string
		src          string
		funcName     string
		expectedDeps []string
	}{
		{
			name: "single pointer parameter",
			src: `package test

type Repository struct{}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type Service struct {
	repo *Repository
}
`,
			funcName:     "NewService",
			expectedDeps: []string{"*test.Repository"},
		},
		{
			name: "multiple parameters",
			src: `package test

type Repository struct{}
type Logger struct{}

func NewService(repo *Repository, logger *Logger) *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{"*test.Repository", "*test.Logger"},
		},
		{
			name: "no parameters",
			src: `package test

func NewService() *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{},
		},
		{
			name: "basic type parameters",
			src: `package test

func NewService(name string, count int) *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{"string", "int"},
		},
		{
			name: "multiple names in single param",
			src: `package test

func NewService(a, b int) *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{"int", "int"},
		},
		{
			name: "slice parameter",
			src: `package test

func NewService(items []string) *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{"[]string"},
		},
		{
			name: "map parameter",
			src: `package test

func NewService(data map[string]int) *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{"map[string]int"},
		},
		{
			name: "channel parameter",
			src: `package test

func NewService(ch chan int) *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{"chan int"},
		},
		{
			name: "variadic parameter",
			src: `package test

type Handler interface{}

func NewLogger(prefix string, handlers ...Handler) *Logger {
	return &Logger{}
}

type Logger struct{}
`,
			funcName:     "NewLogger",
			expectedDeps: []string{"string", "[]test.Handler"}, // variadic is represented as slice
		},
		{
			name: "named type parameter",
			src: `package test

type Config struct {
	Name string
}

func NewService(cfg Config) *Service {
	return &Service{}
}

type Service struct{}
`,
			funcName:     "NewService",
			expectedDeps: []string{"test.Config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, file := mockPass(t, tt.src, nil)

			// Find the function declaration
			var funcDecl *ast.FuncDecl
			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if fn.Name.Name == tt.funcName {
						funcDecl = fn
						return false
					}
				}
				return true
			})

			if funcDecl == nil {
				t.Fatalf("function %s not found", tt.funcName)
			}

			analyzer := detect.NewConstructorAnalyzer()
			deps := analyzer.ExtractDependencies(pass, funcDecl)

			if len(deps) != len(tt.expectedDeps) {
				t.Errorf("ExtractDependencies() returned %d deps, want %d", len(deps), len(tt.expectedDeps))
				t.Logf("got: %v", deps)
				t.Logf("want: %v", tt.expectedDeps)
				return
			}

			for i, expected := range tt.expectedDeps {
				if deps[i] != expected {
					t.Errorf("deps[%d] = %q, want %q", i, deps[i], expected)
				}
			}
		})
	}
}

func TestConstructorAnalyzer_ExtractDependencies_NilCtor(t *testing.T) {
	pass, _ := mockPass(t, "package test", nil)

	analyzer := detect.NewConstructorAnalyzer()
	deps := analyzer.ExtractDependencies(pass, nil)

	if deps != nil {
		t.Errorf("ExtractDependencies(nil) = %v, want nil", deps)
	}
}

func TestConstructorAnalyzer_ExtractDependencies_NilParams(t *testing.T) {
	src := `package test

func NoParams() {}
`
	pass, file := mockPass(t, src, nil)

	var funcDecl *ast.FuncDecl
	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			funcDecl = fn
			return false
		}
		return true
	})

	analyzer := detect.NewConstructorAnalyzer()
	deps := analyzer.ExtractDependencies(pass, funcDecl)

	if len(deps) != 0 {
		t.Errorf("ExtractDependencies() = %v, want empty slice", deps)
	}
}
