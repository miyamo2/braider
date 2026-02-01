package generate_test

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/miyamo2/braider/internal/detect"
	"github.com/miyamo2/braider/internal/generate"
)

func TestConstructorGenerator_GenerateConstructor(t *testing.T) {
	tests := []struct {
		name             string
		structName       string
		fields           []detect.FieldInfo
		expectedFuncName string
		expectedContains []string
	}{
		{
			name:       "single field constructor",
			structName: "UserService",
			fields: []detect.FieldInfo{
				{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
			},
			expectedFuncName: "NewUserService",
			expectedContains: []string{
				"func NewUserService(repo Repository) *UserService",
				"return &UserService{",
				"repo: repo,",
			},
		},
		{
			name:       "multiple fields constructor",
			structName: "OrderService",
			fields: []detect.FieldInfo{
				{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
				{Name: "logger", TypeExpr: &ast.Ident{Name: "Logger"}},
				{Name: "config", TypeExpr: &ast.Ident{Name: "Config"}},
			},
			expectedFuncName: "NewOrderService",
			expectedContains: []string{
				"func NewOrderService(repo Repository, logger Logger, config Config) *OrderService",
				"return &OrderService{",
				"repo:   repo,",
				"logger: logger,",
				"config: config,",
			},
		},
		{
			name:       "pointer type field",
			structName: "Service",
			fields: []detect.FieldInfo{
				{Name: "db", TypeExpr: &ast.StarExpr{X: &ast.Ident{Name: "sql.DB"}}, IsPointer: true},
			},
			expectedFuncName: "NewService",
			expectedContains: []string{
				"func NewService(db *sql.DB) *Service",
				"db: db,",
			},
		},
		{
			name:       "exported field",
			structName: "Config",
			fields: []detect.FieldInfo{
				{Name: "Debug", TypeExpr: &ast.Ident{Name: "bool"}, IsExported: true},
			},
			expectedFuncName: "NewConfig",
			expectedContains: []string{
				"func NewConfig(debug bool) *Config",
				"Debug: debug,",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := generate.NewConstructorGenerator()
			candidate := detect.ConstructorCandidate{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: tt.structName},
				},
			}

			result, err := gen.GenerateConstructor(candidate, tt.fields)
			if err != nil {
				t.Fatalf("GenerateConstructor() error = %v", err)
			}

			if result.FuncName != tt.expectedFuncName {
				t.Errorf("FuncName = %s, want %s", result.FuncName, tt.expectedFuncName)
			}

			if result.StructName != tt.structName {
				t.Errorf("StructName = %s, want %s", result.StructName, tt.structName)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.Code, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.Code)
				}
			}
		})
	}
}

func TestDeriveParamName(t *testing.T) {
	tests := []struct {
		fieldName string
		expected  string
	}{
		{"Repo", "repo"},
		{"Logger", "logger"},
		{"UserService", "userService"},
		{"DB", "db"},
		{"A", "a"},
		// Keywords
		{"Type", "type_"},
		{"Range", "range_"},
		{"Map", "map_"},
		{"Chan", "chan_"},
		{"Func", "func_"},
		{"Interface", "interface_"},
		{"Select", "select_"},
		{"Case", "case_"},
		{"Default", "default_"},
		{"Defer", "defer_"},
		{"Go", "go_"},
		{"Package", "package_"},
		{"Return", "return_"},
		{"Struct", "struct_"},
		{"Switch", "switch_"},
		{"Var", "var_"},
		{"Const", "const_"},
		{"If", "if_"},
		{"Else", "else_"},
		{"For", "for_"},
		{"Break", "break_"},
		{"Continue", "continue_"},
		{"Fallthrough", "fallthrough_"},
		{"Goto", "goto_"},
		{"Import", "import_"},
		// Builtins
		{"Len", "lenParam"},
		{"Cap", "capParam"},
		{"Make", "makeParam"},
		{"New", "newParam"},
		{"Append", "appendParam"},
		{"Copy", "copyParam"},
		{"Delete", "deleteParam"},
		{"Close", "closeParam"},
		{"Panic", "panicParam"},
		{"Recover", "recoverParam"},
		{"Print", "printParam"},
		{"Println", "printlnParam"},
		{"Real", "realParam"},
		{"Imag", "imagParam"},
		{"Complex", "complexParam"},
		{"Error", "errorParam"},
		{"True", "trueParam"},
		{"False", "falseParam"},
		{"Nil", "nilParam"},
		{"Iota", "iotaParam"},
		// Builtin types
		{"String", "stringParam"},
		{"Int", "intParam"},
		{"Int8", "int8Param"},
		{"Int16", "int16Param"},
		{"Int32", "int32Param"},
		{"Int64", "int64Param"},
		{"Uint", "uintParam"},
		{"Uint8", "uint8Param"},
		{"Uint16", "uint16Param"},
		{"Uint32", "uint32Param"},
		{"Uint64", "uint64Param"},
		{"Uintptr", "uintptrParam"},
		{"Float32", "float32Param"},
		{"Float64", "float64Param"},
		{"Complex64", "complex64Param"},
		{"Complex128", "complex128Param"},
		{"Bool", "boolParam"},
		{"Byte", "byteParam"},
		{"Rune", "runeParam"},
		// Already lowercase
		{"repo", "repo"},
		{"logger", "logger"},
		{"config", "config"},
		// Edge cases
		{"", "param"},
		{"STRING", "stringParam"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			result := generate.DeriveParamName(tt.fieldName)
			if result != tt.expected {
				t.Errorf("DeriveParamName(%q) = %q, want %q", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestConstructorGenerator_NilTypeExpr(t *testing.T) {
	gen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "Service"},
		},
	}

	// Field with nil TypeExpr should be handled gracefully
	fields := []detect.FieldInfo{
		{Name: "field", TypeExpr: nil},
	}

	result, err := gen.GenerateConstructor(candidate, fields)
	if err != nil {
		t.Fatalf("GenerateConstructor() error = %v", err)
	}

	// Should generate constructor even with nil type expr
	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestConstructorGenerator_GeneratesGoDoc(t *testing.T) {
	gen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "UserService"},
		},
	}
	fields := []detect.FieldInfo{
		{Name: "repo", TypeExpr: &ast.Ident{Name: "Repository"}},
	}

	result, err := gen.GenerateConstructor(candidate, fields)
	if err != nil {
		t.Fatalf("GenerateConstructor() error = %v", err)
	}

	expectedDoc := "// NewUserService is a constructor for UserService."
	expectedGenMarker := "// Generated by braider. DO NOT EDIT."

	if !strings.Contains(result.Code, expectedDoc) {
		t.Errorf("Code missing GoDoc: %q\nGot:\n%s", expectedDoc, result.Code)
	}

	if !strings.Contains(result.Code, expectedGenMarker) {
		t.Errorf("Code missing generation marker: %q\nGot:\n%s", expectedGenMarker, result.Code)
	}
}

func TestConstructorGenerator_SelectorExpr(t *testing.T) {
	gen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "Service"},
		},
	}
	// Simulate a field with selector expression like "sql.DB"
	fields := []detect.FieldInfo{
		{
			Name: "db",
			TypeExpr: &ast.StarExpr{
				X: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "sql"},
					Sel: &ast.Ident{Name: "DB"},
				},
			},
			IsPointer: true,
		},
	}

	result, err := gen.GenerateConstructor(candidate, fields)
	if err != nil {
		t.Fatalf("GenerateConstructor() error = %v", err)
	}

	if !strings.Contains(result.Code, "*sql.DB") {
		t.Errorf("Code should contain '*sql.DB'\nGot:\n%s", result.Code)
	}
}

func TestConstructorGenerator_EmptyFields(t *testing.T) {
	gen := generate.NewConstructorGenerator()
	candidate := detect.ConstructorCandidate{
		TypeSpec: &ast.TypeSpec{
			Name: &ast.Ident{Name: "Empty"},
		},
	}

	result, err := gen.GenerateConstructor(candidate, []detect.FieldInfo{})
	if err != nil {
		t.Fatalf("GenerateConstructor() error = %v", err)
	}

	// Should generate constructor with no parameters
	if !strings.Contains(result.Code, "func NewEmpty() *Empty") {
		t.Errorf("Code should contain parameterless constructor\nGot:\n%s", result.Code)
	}

	if !strings.Contains(result.Code, "return &Empty{}") {
		t.Errorf("Code should contain empty struct literal\nGot:\n%s", result.Code)
	}
}

func TestConstructorGenerator_ComplexTypes(t *testing.T) {
	tests := []struct {
		name             string
		structName       string
		fields           []detect.FieldInfo
		expectedContains []string
	}{
		{
			name:       "slice type",
			structName: "Service",
			fields: []detect.FieldInfo{
				{
					Name:     "items",
					TypeExpr: &ast.ArrayType{Len: nil, Elt: &ast.Ident{Name: "string"}},
				},
			},
			expectedContains: []string{
				"func NewService(items []string) *Service",
				"items: items,",
			},
		},
		{
			name:       "map type",
			structName: "Cache",
			fields: []detect.FieldInfo{
				{
					Name: "data",
					TypeExpr: &ast.MapType{
						Key:   &ast.Ident{Name: "string"},
						Value: &ast.Ident{Name: "int"},
					},
				},
			},
			expectedContains: []string{
				"func NewCache(data map[string]int) *Cache",
				"data: data,",
			},
		},
		{
			name:       "channel type - send only",
			structName: "Producer",
			fields: []detect.FieldInfo{
				{
					Name: "ch",
					TypeExpr: &ast.ChanType{
						Dir:   ast.SEND,
						Value: &ast.Ident{Name: "string"},
					},
				},
			},
			expectedContains: []string{
				"func NewProducer(ch chan<- string) *Producer",
				"ch: ch,",
			},
		},
		{
			name:       "channel type - receive only",
			structName: "Consumer",
			fields: []detect.FieldInfo{
				{
					Name: "ch",
					TypeExpr: &ast.ChanType{
						Dir:   ast.RECV,
						Value: &ast.Ident{Name: "int"},
					},
				},
			},
			expectedContains: []string{
				"func NewConsumer(ch <-chan int) *Consumer",
				"ch: ch,",
			},
		},
		{
			name:       "channel type - bidirectional",
			structName: "Worker",
			fields: []detect.FieldInfo{
				{
					Name: "ch",
					TypeExpr: &ast.ChanType{
						Dir:   ast.SEND | ast.RECV,
						Value: &ast.Ident{Name: "bool"},
					},
				},
			},
			expectedContains: []string{
				"func NewWorker(ch chan bool) *Worker",
				"ch: ch,",
			},
		},
		{
			name:       "nested complex type",
			structName: "Registry",
			fields: []detect.FieldInfo{
				{
					Name: "mapping",
					TypeExpr: &ast.MapType{
						Key: &ast.Ident{Name: "string"},
						Value: &ast.ArrayType{
							Len: nil,
							Elt: &ast.Ident{Name: "int"},
						},
					},
				},
			},
			expectedContains: []string{
				"func NewRegistry(mapping map[string][]int) *Registry",
				"mapping: mapping,",
			},
		},
		{
			name:       "pointer to slice",
			structName: "Container",
			fields: []detect.FieldInfo{
				{
					Name: "items",
					TypeExpr: &ast.StarExpr{
						X: &ast.ArrayType{
							Len: nil,
							Elt: &ast.Ident{Name: "User"},
						},
					},
					IsPointer: true,
				},
			},
			expectedContains: []string{
				"func NewContainer(items *[]User) *Container",
				"items: items,",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := generate.NewConstructorGenerator()
			candidate := detect.ConstructorCandidate{
				TypeSpec: &ast.TypeSpec{
					Name: &ast.Ident{Name: tt.structName},
				},
			}

			result, err := gen.GenerateConstructor(candidate, tt.fields)
			if err != nil {
				t.Fatalf("GenerateConstructor() error = %v", err)
			}

			for _, contains := range tt.expectedContains {
				if !strings.Contains(result.Code, contains) {
					t.Errorf("Code missing expected content: %q\nGot:\n%s", contains, result.Code)
				}
			}
		})
	}
}

