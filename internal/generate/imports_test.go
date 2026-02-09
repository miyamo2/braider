package generate

import (
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"testing"

	"github.com/miyamo2/braider/internal/graph"
)

// createNamedTypeInPackage creates a *types.Named type with the given name in a synthetic package.
// This is useful for testing RegisteredType-related behavior without needing real type-checked packages.
func createNamedTypeInPackage(name, pkgPath, pkgName string) *types.Named {
	pkg := types.NewPackage(pkgPath, pkgName)
	typeName := types.NewTypeName(token.NoPos, pkg, name, nil)
	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()
	return types.NewNamed(typeName, iface, nil)
}

func TestCollectImports(t *testing.T) {
	tests := []struct {
		name           string
		graph          *graph.Graph
		currentPackage string
		currentPkgName string
		want           []ImportInfo
	}{
		{
			name:           "nil graph",
			graph:          nil,
			currentPackage: "main",
			currentPkgName: "main",
			want:           []ImportInfo{},
		},
		{
			name: "empty graph",
			graph: &graph.Graph{
				Nodes: make(map[string]*graph.Node),
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			currentPkgName: "main",
			want:           []ImportInfo{},
		},
		{
			name: "single package",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"github.com/user/repo.Service": {
						TypeName:    "github.com/user/repo.Service",
						PackagePath: "github.com/user/repo",
						PackageName: "repo",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			currentPkgName: "main",
			want:           []ImportInfo{{Path: "github.com/user/repo", Alias: ""}},
		},
		{
			name: "multiple packages",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"github.com/user/repo.Service": {
						TypeName:    "github.com/user/repo.Service",
						PackagePath: "github.com/user/repo",
						PackageName: "repo",
					},
					"github.com/user/repo.Repository": {
						TypeName:    "github.com/user/repo.Repository",
						PackagePath: "github.com/user/repo",
						PackageName: "repo",
					},
					"github.com/user/other.Client": {
						TypeName:    "github.com/user/other.Client",
						PackagePath: "github.com/user/other",
						PackageName: "other",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			currentPkgName: "main",
			want: []ImportInfo{
				{Path: "github.com/user/other", Alias: ""},
				{Path: "github.com/user/repo", Alias: ""},
			},
		},
		{
			name: "exclude current package",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"main.Service": {
						TypeName:    "main.Service",
						PackagePath: "main",
						PackageName: "main",
					},
					"github.com/user/repo.Client": {
						TypeName:    "github.com/user/repo.Client",
						PackagePath: "github.com/user/repo",
						PackageName: "repo",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			currentPkgName: "main",
			want:           []ImportInfo{{Path: "github.com/user/repo", Alias: ""}},
		},
		{
			name: "sorted output with collision resolution",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"github.com/z/pkg.TypeZ": {
						TypeName:    "github.com/z/pkg.TypeZ",
						PackagePath: "github.com/z/pkg",
						PackageName: "pkg",
					},
					"github.com/a/pkg.TypeA": {
						TypeName:    "github.com/a/pkg.TypeA",
						PackagePath: "github.com/a/pkg",
						PackageName: "pkg",
					},
					"github.com/m/pkg.TypeM": {
						TypeName:    "github.com/m/pkg.TypeM",
						PackagePath: "github.com/m/pkg",
						PackageName: "pkg",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			currentPkgName: "main",
			want: []ImportInfo{
				{Path: "github.com/a/pkg", Alias: ""},      // First alphabetically, no alias
				{Path: "github.com/m/pkg", Alias: "pkg2"},  // Second gets numbered alias
				{Path: "github.com/z/pkg", Alias: "pkg3"},  // Third gets numbered alias
			},
		},
		{
			name: "empty package path nodes",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"github.com/user/repo.Service": {
						TypeName:    "github.com/user/repo.Service",
						PackagePath: "github.com/user/repo",
						PackageName: "repo",
					},
					"EmptyPath.Type": {
						TypeName:    "EmptyPath.Type",
						PackagePath: "", // Empty path
						PackageName: "empty",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			currentPkgName: "main",
			want:           []ImportInfo{{Path: "github.com/user/repo", Alias: ""}},
		},
		{
			name: "different main packages",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"other/main.Service": {
						TypeName:    "other/main.Service",
						PackagePath: "other/main",
						PackageName: "main",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "current/main",
			currentPkgName: "main",
			want:           []ImportInfo{}, // Different main packages shouldn't import each other
		},
		{
			name: "same main package",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"current/main.Service": {
						TypeName:    "current/main.Service",
						PackagePath: "current/main",
						PackageName: "main",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "current/main",
			currentPkgName: "main",
			want:           []ImportInfo{}, // Same main package
		},
		{
			name: "RegisteredType package collision with node package",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/other/domain.Entity": {
						TypeName:    "example.com/other/domain.Entity",
						PackagePath: "example.com/other/domain",
						PackageName: "domain",
					},
					"example.com/service.MyService": {
						TypeName:       "example.com/service.MyService",
						PackagePath:    "example.com/service",
						PackageName:    "service",
						RegisteredType: createNamedTypeInPackage("IRepository", "example.com/typed/domain", "domain"),
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			currentPkgName: "main",
			want: []ImportInfo{
				{Path: "example.com/other/domain", Alias: ""},
				{Path: "example.com/service", Alias: ""},
				{Path: "example.com/typed/domain", Alias: "domain2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, _ := CollectImports(tt.graph, tt.currentPackage, tt.currentPkgName, nil)
				// Handle nil vs empty slice comparison
				if len(got) == 0 && len(tt.want) == 0 {
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("CollectImports() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestDetectExistingAliases(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want map[string]string
	}{
		{
			name: "no imports",
			src:  `package main`,
			want: map[string]string{},
		},
		{
			name: "no aliases",
			src: `package main

import "fmt"
import "os"`,
			want: map[string]string{},
		},
		{
			name: "single alias",
			src: `package main

import myuser "example.com/v1/user"`,
			want: map[string]string{
				"example.com/v1/user": "myuser",
			},
		},
		{
			name: "multiple aliases",
			src: `package main

import (
	v1user "example.com/v1/user"
	v2user "example.com/v2/user"
	"fmt"
)`,
			want: map[string]string{
				"example.com/v1/user": "v1user",
				"example.com/v2/user": "v2user",
			},
		},
		{
			name: "skip dot and blank imports",
			src: `package main

import (
	. "fmt"
	_ "database/sql"
	myuser "example.com/user"
)`,
			want: map[string]string{
				"example.com/user": "myuser",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			got := detectExistingAliases(file)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("detectExistingAliases() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectPackageCollisions(t *testing.T) {
	tests := []struct {
		name string
		g    *graph.Graph
		want map[string]string
	}{
		{
			name: "nil graph",
			g:    nil,
			want: map[string]string{},
		},
		{
			name: "no collisions",
			g: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/user.User": {
						PackagePath: "example.com/user",
						PackageName: "user",
					},
					"example.com/repo.Repo": {
						PackagePath: "example.com/repo",
						PackageName: "repo",
					},
				},
			},
			want: map[string]string{},
		},
		{
			name: "two packages with same name",
			g: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/v1/user.User": {
						PackagePath: "example.com/v1/user",
						PackageName: "user",
					},
					"example.com/v2/user.User": {
						PackagePath: "example.com/v2/user",
						PackageName: "user",
					},
				},
			},
			want: map[string]string{
				"example.com/v1/user": "user",
				"example.com/v2/user": "user",
			},
		},
		{
			name: "three packages with same name",
			g: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/pkg/user.User": {
						PackagePath: "example.com/pkg/user",
						PackageName: "user",
					},
					"example.com/lib/user.User": {
						PackagePath: "example.com/lib/user",
						PackageName: "user",
					},
					"example.com/util/user.User": {
						PackagePath: "example.com/util/user",
						PackageName: "user",
					},
				},
			},
			want: map[string]string{
				"example.com/pkg/user":  "user",
				"example.com/lib/user":  "user",
				"example.com/util/user": "user",
			},
		},
		{
			name: "collision via RegisteredType package",
			g: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"example.com/other/domain.Entity": {
						PackagePath: "example.com/other/domain",
						PackageName: "domain",
					},
					"example.com/service.MyService": {
						PackagePath:    "example.com/service",
						PackageName:    "service",
						RegisteredType: createNamedTypeInPackage("IRepository", "example.com/typed/domain", "domain"),
					},
				},
			},
			want: map[string]string{
				"example.com/other/domain": "domain",
				"example.com/typed/domain": "domain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectPackageCollisions(tt.g)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("detectPackageCollisions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestImportInfo_HasAlias(t *testing.T) {
	tests := []struct {
		name string
		info ImportInfo
		want bool
	}{
		{
			name: "with alias",
			info: ImportInfo{Path: "example.com/v1/user", Alias: "v1user"},
			want: true,
		},
		{
			name: "without alias",
			info: ImportInfo{Path: "example.com/user", Alias: ""},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.HasAlias()
			if got != tt.want {
				t.Errorf("HasAlias() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name    string
		pkgPath string
		want    string
	}{
		{
			name:    "v1 version",
			pkgPath: "example.com/v1/user",
			want:    "v1",
		},
		{
			name:    "v2 version",
			pkgPath: "example.com/v2/user",
			want:    "v2",
		},
		{
			name:    "v10 version",
			pkgPath: "example.com/v10/user",
			want:    "v10",
		},
		{
			name:    "no version",
			pkgPath: "example.com/user",
			want:    "",
		},
		{
			name:    "version at end",
			pkgPath: "example.com/user/v3",
			want:    "v3",
		},
		{
			name:    "version in middle",
			pkgPath: "example.com/v2/api/user",
			want:    "v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVersion(tt.pkgPath)
			if got != tt.want {
				t.Errorf("extractVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateAliases(t *testing.T) {
	tests := []struct {
		name            string
		collisions      map[string]string
		existingAliases map[string]string
		want            map[string]string
	}{
		{
			name:            "empty collisions",
			collisions:      map[string]string{},
			existingAliases: map[string]string{},
			want:            map[string]string{},
		},
		{
			name: "version-based aliases",
			collisions: map[string]string{
				"example.com/v1/user": "user",
				"example.com/v2/user": "user",
			},
			existingAliases: map[string]string{},
			want: map[string]string{
				"example.com/v1/user": "v1user",
				"example.com/v2/user": "v2user",
			},
		},
		{
			name: "preserve existing aliases",
			collisions: map[string]string{
				"example.com/v1/user": "user",
				"example.com/v2/user": "user",
			},
			existingAliases: map[string]string{
				"example.com/v1/user": "myuser",
			},
			want: map[string]string{
				"example.com/v1/user": "myuser",
				"example.com/v2/user": "v2user",
			},
		},
		{
			name: "numbered fallback",
			collisions: map[string]string{
				"example.com/pkg/user":  "user",
				"example.com/lib/user":  "user",
				"example.com/util/user": "user",
			},
			existingAliases: map[string]string{},
			want: map[string]string{
				"example.com/lib/user":  "",      // First alphabetically gets no alias
				"example.com/pkg/user":  "user2", // Second gets user2
				"example.com/util/user": "user3", // Third gets user3
			},
		},
		{
			name: "mixed version and numbered",
			collisions: map[string]string{
				"example.com/v1/user": "user",
				"example.com/pkg/user": "user",
			},
			existingAliases: map[string]string{},
			want: map[string]string{
				"example.com/pkg/user": "",
				"example.com/v1/user":  "v1user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateAliases(tt.collisions, tt.existingAliases)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateAliases() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractPackagePaths(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want []string
	}{
		{
			name: "named type",
			typ:  createNamedTypeInPackage("Foo", "example.com/foo", "foo"),
			want: []string{"example.com/foo"},
		},
		{
			name: "pointer to named type",
			typ:  types.NewPointer(createNamedTypeInPackage("Bar", "example.com/bar", "bar")),
			want: []string{"example.com/bar"},
		},
		{
			name: "interface with embedded named type",
			typ: func() types.Type {
				embedded := createNamedTypeInPackage("IRepo", "example.com/domain", "domain")
				iface := types.NewInterfaceType(nil, []types.Type{embedded})
				iface.Complete()
				return iface
			}(),
			want: []string{"example.com/domain"},
		},
		{
			name: "nil-package named type",
			typ: func() types.Type {
				// Built-in type has no package
				typeName := types.NewTypeName(token.NoPos, nil, "error", nil)
				iface := types.NewInterfaceType(nil, nil)
				iface.Complete()
				return types.NewNamed(typeName, iface, nil)
			}(),
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPackagePaths(tt.typ)
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractPackagePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}
