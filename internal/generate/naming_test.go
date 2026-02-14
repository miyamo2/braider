package generate

import (
	"fmt"
	"testing"
)

func TestDeriveFieldName(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		want     string
	}{
		// Basic cases
		{
			name:     "simple type",
			typeName: "Service",
			want:     "service",
		},
		{
			name:     "already lowercase",
			typeName: "service",
			want:     "service",
		},
		// Abbreviations
		{
			name:     "DB prefix",
			typeName: "DBConnection",
			want:     "dbConnection",
		},
		{
			name:     "HTTP prefix",
			typeName: "HTTPClient",
			want:     "httpClient",
		},
		{
			name:     "all caps",
			typeName: "HTTP",
			want:     "http",
		},
		{
			name:     "URL type",
			typeName: "URL",
			want:     "url",
		},
		// Fully qualified names
		{
			name:     "fully qualified",
			typeName: "github.com/user/repo.Service",
			want:     "service",
		},
		{
			name:     "fully qualified with abbreviation",
			typeName: "github.com/user/repo.HTTPClient",
			want:     "httpClient",
		},
		// Keyword conflicts
		{
			name:     "type keyword",
			typeName: "Type",
			want:     "type_",
		},
		{
			name:     "map keyword",
			typeName: "Map",
			want:     "map_",
		},
		{
			name:     "string builtin",
			typeName: "String",
			want:     "string_",
		},
		{
			name:     "error builtin",
			typeName: "Error",
			want:     "error_",
		},
		// Edge cases
		{
			name:     "single character",
			typeName: "A",
			want:     "a",
		},
		{
			name:     "multiple words",
			typeName: "UserService",
			want:     "userService",
		},
		{
			name:     "with numbers",
			typeName: "OAuth2Client",
			want:     "oAuth2Client", // Numbers break uppercase sequence
		},
		// Edge case: empty string
		{
			name:     "empty string",
			typeName: "",
			want:     "field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveFieldName(tt.typeName)
			if got != tt.want {
				t.Errorf("DeriveFieldName(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}

func TestToLowerCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple",
			input: "Service",
			want:  "service",
		},
		{
			name:  "all caps abbreviation",
			input: "HTTP",
			want:  "http",
		},
		{
			name:  "abbreviation with word",
			input: "HTTPClient",
			want:  "httpClient",
		},
		{
			name:  "multiple abbreviations",
			input: "HTTPSClient",
			want:  "httpsClient",
		},
		{
			name:  "DB prefix",
			input: "DBConnection",
			want:  "dbConnection",
		},
		{
			name:  "already camelCase",
			input: "userService",
			want:  "userService",
		},
		{
			name:  "mixed case",
			input: "UserService",
			want:  "userService",
		},
		// Edge case: empty string
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToLowerCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("ToLowerCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsKeywordOrBuiltin(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Keywords
		{
			name:  "type keyword",
			input: "type",
			want:  true,
		},
		{
			name:  "func keyword",
			input: "func",
			want:  true,
		},
		{
			name:  "var keyword",
			input: "var",
			want:  true,
		},
		// Builtins
		{
			name:  "string builtin",
			input: "string",
			want:  true,
		},
		{
			name:  "error builtin",
			input: "error",
			want:  true,
		},
		{
			name:  "append builtin",
			input: "append",
			want:  true,
		},
		// Not keywords/builtins
		{
			name:  "service",
			input: "service",
			want:  false,
		},
		{
			name:  "client",
			input: "client",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsKeywordOrBuiltin(tt.input)
			if got != tt.want {
				t.Errorf("IsKeywordOrBuiltin(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestToUpperCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase start",
			input: "service",
			want:  "Service",
		},
		{
			name:  "already uppercase start",
			input: "Service",
			want:  "Service",
		},
		{
			name:  "all lowercase",
			input: "dbhandler",
			want:  "Dbhandler",
		},
		{
			name:  "lowerCamelCase",
			input: "dbHandler",
			want:  "DbHandler",
		},
		{
			name:  "all caps",
			input: "DB",
			want:  "DB",
		},
		{
			name:  "single lowercase char",
			input: "a",
			want:  "A",
		},
		{
			name:  "single uppercase char",
			input: "A",
			want:  "A",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "unicode lowercase",
			input: "über",
			want:  "Über",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToUpperCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("ToUpperCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Example demonstrates the usage of ToLowerCamelCase function.
func ExampleToLowerCamelCase() {
	// Simple type
	fmt.Println(ToLowerCamelCase("Service"))

	// All-caps abbreviations
	fmt.Println(ToLowerCamelCase("HTTP"))
	fmt.Println(ToLowerCamelCase("HTTPClient"))
	fmt.Println(ToLowerCamelCase("DBConnection"))

	// Already camelCase
	fmt.Println(ToLowerCamelCase("userService"))

	// Output:
	// service
	// http
	// httpClient
	// dbConnection
	// userService
}
