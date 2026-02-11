// Package detect_test contains tests for the detect package.
//
// OptionExtractor tests use file-based test fixtures from testdata/option_extractor/
// rather than in-memory package construction. This provides:
//   - Realistic type checking via the Go toolchain
//   - Actual dependency resolution (github.com/miyamo2/braider/pkg)
//   - Simpler test maintenance
//
// To add a new test case:
//  1. Create directory: testdata/option_extractor/<case_name>/
//  2. Add main.go with struct using annotation.Injectable[T]
//  3. Add go.mod with replace directive (see existing examples)
//  4. Add test function using testdata.LoadTestPackage()
package detect

import (
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestOptionExtractor_ExtractInjectOptions_Default(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/default")
	injectableExpr, _, structName := FindInjectableField(t, pkg)

	// Extract concrete type from the struct
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		t.Fatalf("Struct %s not found in package scope", structName)
	}
	concreteType := types.NewPointer(obj.Type())

	mockValidator := &MockNamerValidator{}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractInjectOptions(pass, injectableExpr, concreteType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assertions
	if !metadata.IsDefault {
		t.Error("Expected IsDefault=true")
	}
	if metadata.WithoutConstructor {
		t.Error("Expected WithoutConstructor=false")
	}
	if metadata.TypedInterface != nil {
		t.Errorf("Expected TypedInterface=nil, got %v", metadata.TypedInterface)
	}
	if metadata.Name != "" {
		t.Errorf("Expected Name=\"\", got %q", metadata.Name)
	}
}

func TestOptionExtractor_ExtractInjectOptions_Typed(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/typed")
	injectableExpr, _, structName := FindInjectableField(t, pkg)

	// Extract concrete type from the struct
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		t.Fatalf("Struct %s not found in package scope", structName)
	}
	concreteType := types.NewPointer(obj.Type())

	// Find MyInterface type
	ifaceObj := pkg.Types.Scope().Lookup("MyInterface")
	if ifaceObj == nil {
		t.Fatal("MyInterface not found")
	}
	ifaceType := ifaceObj.Type()

	mockValidator := &MockNamerValidator{}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractInjectOptions(pass, injectableExpr, concreteType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if metadata.TypedInterface == nil {
		t.Error("Expected TypedInterface to be set, got nil")
	} else if metadata.TypedInterface != ifaceType {
		t.Errorf("Expected TypedInterface=%v, got %v", ifaceType, metadata.TypedInterface)
	}

	// Should not be default when typed
	if metadata.IsDefault {
		t.Error("Expected IsDefault=false for Typed option, got true")
	}
}

func TestOptionExtractor_ExtractInjectOptions_Named(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/named")
	injectableExpr, _, structName := FindInjectableField(t, pkg)

	// Extract concrete type from the struct
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		t.Fatalf("Struct %s not found in package scope", structName)
	}
	concreteType := types.NewPointer(obj.Type())

	mockValidator := &MockNamerValidator{
		ExtractNameFn: func(pass *analysis.Pass, nt types.Type) (string, error) {
			return "database", nil
		},
	}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractInjectOptions(pass, injectableExpr, concreteType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if metadata.Name != "database" {
		t.Errorf("Expected Name=\"database\", got %q", metadata.Name)
	}
}

func TestOptionExtractor_ExtractInjectOptions_WithoutConstructor(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/without_constructor")
	injectableExpr, _, structName := FindInjectableField(t, pkg)

	// Extract concrete type from the struct
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		t.Fatalf("Struct %s not found in package scope", structName)
	}
	concreteType := types.NewPointer(obj.Type())

	mockValidator := &MockNamerValidator{}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractInjectOptions(pass, injectableExpr, concreteType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !metadata.WithoutConstructor {
		t.Error("Expected WithoutConstructor=true, got false")
	}
	if metadata.IsDefault {
		t.Error("Expected IsDefault=false when WithoutConstructor is set, got true")
	}
}

func TestOptionExtractor_ExtractInjectOptions_TypedNonInterface(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/typed_non_interface")
	injectableExpr, _, structName := FindInjectableField(t, pkg)

	// Extract concrete type from the struct
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		t.Fatalf("Struct %s not found in package scope", structName)
	}
	concreteType := types.NewPointer(obj.Type())

	mockValidator := &MockNamerValidator{}
	extractor := NewOptionExtractor(mockValidator)

	_, err := extractor.ExtractInjectOptions(pass, injectableExpr, concreteType)

	if err == nil {
		t.Error("Expected error for Typed[I] with non-interface type, got nil")
		return
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Typed[I] requires an interface type") {
		t.Errorf("Expected error to contain 'Typed[I] requires an interface type', got: %v", err)
	}
}

func TestOptionExtractor_ExtractVariableOptions_Default(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/variable_default")
	callExpr, argType := FindVariableCall(t, pkg)

	mockValidator := &MockNamerValidator{}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractVariableOptions(pass, callExpr, argType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !metadata.IsDefault {
		t.Error("Expected IsDefault=true")
	}
	if metadata.WithoutConstructor {
		t.Error("Expected WithoutConstructor=false for Variable")
	}
	if metadata.TypedInterface != nil {
		t.Errorf("Expected TypedInterface=nil, got %v", metadata.TypedInterface)
	}
	if metadata.Name != "" {
		t.Errorf("Expected Name=\"\", got %q", metadata.Name)
	}
}

func TestOptionExtractor_ExtractVariableOptions_Typed(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/variable_typed")
	callExpr, argType := FindVariableCall(t, pkg)

	// Find MyWriter type
	ifaceObj := pkg.Types.Scope().Lookup("MyWriter")
	if ifaceObj == nil {
		t.Fatal("MyWriter not found")
	}
	ifaceType := ifaceObj.Type()

	mockValidator := &MockNamerValidator{}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractVariableOptions(pass, callExpr, argType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if metadata.TypedInterface == nil {
		t.Error("Expected TypedInterface to be set, got nil")
	} else if metadata.TypedInterface != ifaceType {
		t.Errorf("Expected TypedInterface=%v, got %v", ifaceType, metadata.TypedInterface)
	}

	if metadata.IsDefault {
		t.Error("Expected IsDefault=false for Typed option, got true")
	}
}

func TestOptionExtractor_ExtractVariableOptions_Named(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/variable_named")
	callExpr, argType := FindVariableCall(t, pkg)

	mockValidator := &MockNamerValidator{
		ExtractNameFn: func(pass *analysis.Pass, nt types.Type) (string, error) {
			return "stdout", nil
		},
	}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractVariableOptions(pass, callExpr, argType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if metadata.Name != "stdout" {
		t.Errorf("Expected Name=\"stdout\", got %q", metadata.Name)
	}
}

func TestOptionExtractor_ExtractVariableOptions_Mixed(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/variable_mixed")
	callExpr, argType := FindVariableCall(t, pkg)

	// Find MyWriter type
	ifaceObj := pkg.Types.Scope().Lookup("MyWriter")
	if ifaceObj == nil {
		t.Fatal("MyWriter not found")
	}
	ifaceType := ifaceObj.Type()

	mockValidator := &MockNamerValidator{
		ExtractNameFn: func(pass *analysis.Pass, nt types.Type) (string, error) {
			return "stdout", nil
		},
	}
	extractor := NewOptionExtractor(mockValidator)

	metadata, err := extractor.ExtractVariableOptions(pass, callExpr, argType)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have both Typed and Named
	if metadata.TypedInterface == nil {
		t.Error("Expected TypedInterface to be set, got nil")
	} else if metadata.TypedInterface != ifaceType {
		t.Errorf("Expected TypedInterface=%v, got %v", ifaceType, metadata.TypedInterface)
	}

	if metadata.Name != "stdout" {
		t.Errorf("Expected Name=\"stdout\", got %q", metadata.Name)
	}

	// WithoutConstructor should never be set for Variable
	if metadata.WithoutConstructor {
		t.Error("Expected WithoutConstructor=false for Variable")
	}
}

func TestOptionExtractor_ExtractInjectOptions_InterfaceImplValidationError(t *testing.T) {
	pkg, pass := LoadTestPackage(t, "option_extractor/interface_validation_error")
	injectableExpr, _, structName := FindInjectableField(t, pkg)

	// Extract concrete type from the struct
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		t.Fatalf("Struct %s not found in package scope", structName)
	}
	concreteType := types.NewPointer(obj.Type())

	mockValidator := &MockNamerValidator{}
	extractor := NewOptionExtractor(mockValidator)

	_, err := extractor.ExtractInjectOptions(pass, injectableExpr, concreteType)

	if err == nil {
		t.Error("Expected error for interface implementation mismatch, got nil")
		return
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "does not implement interface") {
		t.Errorf("Expected error to contain 'does not implement interface', got: %v", err)
	}
}
