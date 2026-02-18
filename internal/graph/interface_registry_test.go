package graph

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/miyamo2/braider/internal/registry"
	"golang.org/x/tools/go/analysis"
)

// TestInterfaceRegistry_Build tests the Build method for constructing the interface registry.
func TestInterfaceRegistry_Build(t *testing.T) {
	tests := []struct {
		name           string
		providers      []*registry.ProviderInfo
		injectors      []*registry.InjectorInfo
		src            string
		wantInterfaces map[string]string // interface type -> implementation type
		wantErr        bool
	}{
		{
			name: "single provider implements interface",
			providers: []*registry.ProviderInfo{
				{
					TypeName:    "example.com/repo.UserRepository",
					PackagePath: "example.com/repo",
					LocalName:   "UserRepository",
					Implements:  []string{"example.com/domain.IUserRepository"},
				},
			},
			injectors: nil,
			src: `
package test

type IUserRepository interface {
	FindByID(string) string
}

type UserRepository struct{}

func (r *UserRepository) FindByID(id string) string {
	return id
}
`,
			wantInterfaces: map[string]string{
				"example.com/domain.IUserRepository": "example.com/repo.UserRepository",
			},
			wantErr: false,
		},
		{
			name:      "single injector implements interface",
			providers: nil,
			injectors: []*registry.InjectorInfo{
				{
					TypeName:    "example.com/service.UserService",
					PackagePath: "example.com/service",
					LocalName:   "UserService",
					Implements:  []string{"example.com/domain.IUserService"},
				},
			},
			src: `
package test

type IUserService interface {
	Run()
}

type UserService struct{}

func (s *UserService) Run() {}
`,
			wantInterfaces: map[string]string{
				"example.com/domain.IUserService": "example.com/service.UserService",
			},
			wantErr: false,
		},
		{
			name: "no implementations",
			providers: []*registry.ProviderInfo{
				{
					TypeName:    "example.com/repo.UserRepository",
					PackagePath: "example.com/repo",
					LocalName:   "UserRepository",
					Implements:  []string{},
				},
			},
			injectors: nil,
			src: `
package test

type UserRepository struct{}
`,
			wantInterfaces: map[string]string{},
			wantErr:        false,
		},
		{
			name: "provider and injector implement different interfaces",
			providers: []*registry.ProviderInfo{
				{
					TypeName:    "example.com/repo.UserRepository",
					PackagePath: "example.com/repo",
					LocalName:   "UserRepository",
					Implements:  []string{"example.com/domain.IUserRepository"},
				},
			},
			injectors: []*registry.InjectorInfo{
				{
					TypeName:    "example.com/service.UserService",
					PackagePath: "example.com/service",
					LocalName:   "UserService",
					Implements:  []string{"example.com/domain.IUserService"},
				},
			},
			src: `
package test

type IUserRepository interface {
	FindByID(string) string
}

type IUserService interface {
	Run()
}

type UserRepository struct{}

func (r *UserRepository) FindByID(id string) string {
	return id
}

type UserService struct{}

func (s *UserService) Run() {}
`,
			wantInterfaces: map[string]string{
				"example.com/domain.IUserRepository": "example.com/repo.UserRepository",
				"example.com/domain.IUserService":    "example.com/service.UserService",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock analysis.Pass
			pass := createMockPass(t, tt.src)

			// Create registry
			reg := NewInterfaceRegistry()

			// Build registry
			err := reg.Build(pass, tt.providers, tt.injectors, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check registered interfaces
			for iface, want := range tt.wantInterfaces {
				got, err := reg.Resolve(iface)
				if err != nil {
					t.Errorf("Resolve(%s) error = %v", iface, err)
					continue
				}
				if got != want {
					t.Errorf("Resolve(%s) = %s, want %s", iface, got, want)
				}
			}
		})
	}
}

// TestInterfaceRegistry_Resolve tests the Resolve method for interface resolution.
func TestInterfaceRegistry_Resolve(t *testing.T) {
	tests := []struct {
		name       string
		interfaces map[string][]string // interface -> implementations
		queryIface string
		want       string
		wantErr    bool
		errType    string // "ambiguous" or "unresolved"
	}{
		{
			name: "single implementation found",
			interfaces: map[string][]string{
				"example.com/domain.IUserRepository": {"example.com/repo.UserRepository"},
			},
			queryIface: "example.com/domain.IUserRepository",
			want:       "example.com/repo.UserRepository",
			wantErr:    false,
		},
		{
			name: "interface not found",
			interfaces: map[string][]string{
				"example.com/domain.IUserRepository": {"example.com/repo.UserRepository"},
			},
			queryIface: "example.com/domain.IOrderRepository",
			want:       "",
			wantErr:    true,
			errType:    "unresolved",
		},
		{
			name: "ambiguous implementation",
			interfaces: map[string][]string{
				"example.com/domain.IUserRepository": {
					"example.com/repo.UserRepositoryA",
					"example.com/repo.UserRepositoryB",
				},
			},
			queryIface: "example.com/domain.IUserRepository",
			want:       "",
			wantErr:    true,
			errType:    "ambiguous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &InterfaceRegistry{
				interfaces: tt.interfaces,
			}

			got, err := reg.Resolve(tt.queryIface)

			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				switch tt.errType {
				case "ambiguous":
					if _, ok := err.(*AmbiguousImplementationError); !ok {
						t.Errorf("Resolve() error type = %T, want *AmbiguousImplementationError", err)
					}
				case "unresolved":
					if _, ok := err.(*UnresolvedInterfaceError); !ok {
						t.Errorf("Resolve() error type = %T, want *UnresolvedInterfaceError", err)
					}
				}
			}

			if got != tt.want {
				t.Errorf("Resolve() = %s, want %s", got, tt.want)
			}
		})
	}
}

// TestInterfaceRegistry_ErrorMessages tests error message formatting.
func TestInterfaceRegistry_ErrorMessages(t *testing.T) {
	t.Run("AmbiguousImplementationError", func(t *testing.T) {
		err := &AmbiguousImplementationError{
			InterfaceType:   "example.com/domain.IUserRepository",
			Implementations: []string{"example.com/repo.UserRepositoryA", "example.com/repo.UserRepositoryB"},
		}
		msg := err.Error()
		want := "multiple injectable structs implement interface example.com/domain.IUserRepository: example.com/repo.UserRepositoryA, example.com/repo.UserRepositoryB"
		if msg != want {
			t.Errorf("AmbiguousImplementationError.Error() = %q, want %q", msg, want)
		}
	})

	t.Run("UnresolvedInterfaceError", func(t *testing.T) {
		err := &UnresolvedInterfaceError{
			InterfaceType: "io.Reader",
		}
		msg := err.Error()
		want := "no injectable struct implements interface io.Reader; add annotation.Provide or annotation.Injectable to an implementing struct or change parameter to concrete type"
		if msg != want {
			t.Errorf("UnresolvedInterfaceError.Error() = %q, want %q", msg, want)
		}
	})
}

// TestInterfaceRegistry_Build_WithVariables tests the Build method including Variable entries.
func TestInterfaceRegistry_Build_WithVariables(t *testing.T) {
	tests := []struct {
		name           string
		providers      []*registry.ProviderInfo
		injectors      []*registry.InjectorInfo
		variables      []*registry.VariableInfo
		wantInterfaces map[string]string // interface type -> implementation type
		wantErr        bool
	}{
		{
			name:      "variable implements interface via Typed[I]",
			providers: nil,
			injectors: nil,
			variables: []*registry.VariableInfo{
				{
					TypeName:       "os.File",
					PackagePath:    "os",
					PackageName:    "os",
					LocalName:      "File",
					ExpressionText: "os.Stdout",
					Dependencies:   []string{},
					Implements:     []string{"io.Writer"},
				},
			},
			wantInterfaces: map[string]string{
				"io.Writer": "os.File",
			},
			wantErr: false,
		},
		{
			name: "variable and provider implement different interfaces",
			providers: []*registry.ProviderInfo{
				{
					TypeName:    "example.com/repo.UserRepository",
					PackagePath: "example.com/repo",
					LocalName:   "UserRepository",
					Implements:  []string{"example.com/domain.IUserRepository"},
				},
			},
			injectors: nil,
			variables: []*registry.VariableInfo{
				{
					TypeName:       "os.File",
					PackagePath:    "os",
					PackageName:    "os",
					LocalName:      "File",
					ExpressionText: "os.Stdout",
					Dependencies:   []string{},
					Implements:     []string{"io.Writer"},
				},
			},
			wantInterfaces: map[string]string{
				"example.com/domain.IUserRepository": "example.com/repo.UserRepository",
				"io.Writer":                          "os.File",
			},
			wantErr: false,
		},
		{
			name:      "variable with no interface implementations",
			providers: nil,
			injectors: nil,
			variables: []*registry.VariableInfo{
				{
					TypeName:       "os.File",
					PackagePath:    "os",
					PackageName:    "os",
					LocalName:      "File",
					ExpressionText: "os.Stdout",
					Dependencies:   []string{},
					Implements:     []string{},
				},
			},
			wantInterfaces: map[string]string{},
			wantErr:        false,
		},
		{
			name:      "variable with multiple interface implementations",
			providers: nil,
			injectors: nil,
			variables: []*registry.VariableInfo{
				{
					TypeName:       "os.File",
					PackagePath:    "os",
					PackageName:    "os",
					LocalName:      "File",
					ExpressionText: "os.Stdout",
					Dependencies:   []string{},
					Implements:     []string{"io.Writer", "io.ReadCloser"},
				},
			},
			wantInterfaces: map[string]string{
				"io.Writer":     "os.File",
				"io.ReadCloser": "os.File",
			},
			wantErr: false,
		},
		{
			name:      "nil variables parameter works",
			providers: nil,
			injectors: nil,
			variables: nil,
			wantInterfaces: map[string]string{},
			wantErr:        false,
		},
		{
			name: "variable and injector implement same interface causes ambiguity",
			providers: nil,
			injectors: []*registry.InjectorInfo{
				{
					TypeName:    "example.com/service.MyWriter",
					PackagePath: "example.com/service",
					LocalName:   "MyWriter",
					Implements:  []string{"io.Writer"},
				},
			},
			variables: []*registry.VariableInfo{
				{
					TypeName:       "os.File",
					PackagePath:    "os",
					PackageName:    "os",
					LocalName:      "File",
					ExpressionText: "os.Stdout",
					Dependencies:   []string{},
					Implements:     []string{"io.Writer"},
				},
			},
			// Both implement io.Writer - should result in ambiguous error on Resolve
			wantInterfaces: map[string]string{}, // We won't check resolve here
			wantErr:        false,                // Build itself doesn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := `package test`
			pass := createMockPass(t, src)

			reg := NewInterfaceRegistry()
			err := reg.Build(pass, tt.providers, tt.injectors, tt.variables)

			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for iface, want := range tt.wantInterfaces {
				got, err := reg.Resolve(iface)
				if err != nil {
					t.Errorf("Resolve(%s) error = %v", iface, err)
					continue
				}
				if got != want {
					t.Errorf("Resolve(%s) = %s, want %s", iface, got, want)
				}
			}
		})
	}

	t.Run("ambiguous resolution when variable and injector implement same interface", func(t *testing.T) {
		src := `package test`
		pass := createMockPass(t, src)

		injectors := []*registry.InjectorInfo{
			{
				TypeName:    "example.com/service.MyWriter",
				PackagePath: "example.com/service",
				LocalName:   "MyWriter",
				Implements:  []string{"io.Writer"},
			},
		}
		variables := []*registry.VariableInfo{
			{
				TypeName:       "os.File",
				PackagePath:    "os",
				PackageName:    "os",
				LocalName:      "File",
				ExpressionText: "os.Stdout",
				Dependencies:   []string{},
				Implements:     []string{"io.Writer"},
			},
		}

		reg := NewInterfaceRegistry()
		if err := reg.Build(pass, nil, injectors, variables); err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		_, err := reg.Resolve("io.Writer")
		if err == nil {
			t.Fatal("Resolve() should return error for ambiguous implementation")
		}
		if _, ok := err.(*AmbiguousImplementationError); !ok {
			t.Errorf("Resolve() error type = %T, want *AmbiguousImplementationError", err)
		}
	})
}

// createMockPass creates a mock analysis.Pass for testing.
func createMockPass(t *testing.T, src string) *analysis.Pass {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	// Create type checker
	conf := types.Config{
		Importer: nil, // Simple test doesn't need imports
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	pkg, err := conf.Check("test", fset, []*ast.File{file}, info)
	if err != nil {
		t.Logf("type check warning (expected in simple tests): %v", err)
	}

	return &analysis.Pass{
		Fset:      fset,
		Files:     []*ast.File{file},
		Pkg:       pkg,
		TypesInfo: info,
	}
}
