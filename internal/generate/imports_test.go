package generate

import (
	"reflect"
	"testing"

	"github.com/miyamo2/braider/internal/graph"
)

func TestCollectImports(t *testing.T) {
	tests := []struct {
		name           string
		graph          *graph.Graph
		currentPackage string
		want           []string
	}{
		{
			name:           "nil graph",
			graph:          nil,
			currentPackage: "main",
			want:           nil,
		},
		{
			name: "empty graph",
			graph: &graph.Graph{
				Nodes: make(map[string]*graph.Node),
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			want:           []string{},
		},
		{
			name: "single package",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"github.com/user/repo.Service": {
						TypeName:    "github.com/user/repo.Service",
						PackagePath: "github.com/user/repo",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			want:           []string{"github.com/user/repo"},
		},
		{
			name: "multiple packages",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"github.com/user/repo.Service": {
						TypeName:    "github.com/user/repo.Service",
						PackagePath: "github.com/user/repo",
					},
					"github.com/user/repo.Repository": {
						TypeName:    "github.com/user/repo.Repository",
						PackagePath: "github.com/user/repo",
					},
					"github.com/user/other.Client": {
						TypeName:    "github.com/user/other.Client",
						PackagePath: "github.com/user/other",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			want: []string{
				"github.com/user/other",
				"github.com/user/repo",
			},
		},
		{
			name: "exclude current package",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"main.Service": {
						TypeName:    "main.Service",
						PackagePath: "main",
					},
					"github.com/user/repo.Client": {
						TypeName:    "github.com/user/repo.Client",
						PackagePath: "github.com/user/repo",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			want:           []string{"github.com/user/repo"},
		},
		{
			name: "sorted output",
			graph: &graph.Graph{
				Nodes: map[string]*graph.Node{
					"github.com/z/pkg.TypeZ": {
						TypeName:    "github.com/z/pkg.TypeZ",
						PackagePath: "github.com/z/pkg",
					},
					"github.com/a/pkg.TypeA": {
						TypeName:    "github.com/a/pkg.TypeA",
						PackagePath: "github.com/a/pkg",
					},
					"github.com/m/pkg.TypeM": {
						TypeName:    "github.com/m/pkg.TypeM",
						PackagePath: "github.com/m/pkg",
					},
				},
				Edges: make(map[string][]string),
			},
			currentPackage: "main",
			want: []string{
				"github.com/a/pkg",
				"github.com/m/pkg",
				"github.com/z/pkg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollectImports(tt.graph, tt.currentPackage)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CollectImports() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractPackagePath(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     string
	}{
		{
			name:     "fully qualified type",
			typeName: "github.com/user/repo.Service",
			want:     "github.com/user/repo",
		},
		{
			name:     "nested package",
			typeName: "github.com/user/repo/internal/service.Handler",
			want:     "github.com/user/repo/internal/service",
		},
		{
			name:     "no package",
			typeName: "Service",
			want:     "",
		},
		{
			name:     "empty string",
			typeName: "",
			want:     "",
		},
		{
			name:     "standard library",
			typeName: "context.Context",
			want:     "context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPackagePath(tt.typeName)
			if got != tt.want {
				t.Errorf("ExtractPackagePath(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}
