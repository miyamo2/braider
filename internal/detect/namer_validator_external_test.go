// Package detect contains tests for the NamerValidator with file-based test fixtures.
//
// NamerValidator tests use file-based test fixtures from testdata/namer_validator/
// rather than in-memory package construction. This provides:
//   - Realistic type checking via the Go toolchain
//   - Actual dependency resolution
//   - Simpler test maintenance
//
// To add a new test case:
//   1. Create directory: testdata/namer_validator/<case_name>/
//   2. Add main.go with the test case
//   3. Add go.mod
//   4. Add test function using LoadTestPackage()
package detect

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// TestNamerValidator_ExtractName_InvalidSignature tests rejection of invalid method signatures
func TestNamerValidator_ExtractName_InvalidSignature(t *testing.T) {
	tests := []struct {
		name        string
		relativeDir string
		typeName    string
		errContains string
	}{
		{
			name:        "Name method with parameters",
			relativeDir: "namer_validator/invalid_signature/with_parameters",
			typeName:    "InvalidName",
			errContains: "must return hardcoded string literal",
		},
		{
			name:        "Name method with multiple return values",
			relativeDir: "namer_validator/invalid_signature/multiple_returns",
			typeName:    "MultiReturn",
			errContains: "must return exactly one value",
		},
		{
			name:        "Name method with wrong return type",
			relativeDir: "namer_validator/invalid_signature/wrong_return_type",
			typeName:    "WrongType",
			errContains: "must return string",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				pkg, pass := LoadTestPackage(t, tt.relativeDir)

				validator := NewNamerValidator(nil)

				namedType := FindNamedType(pkg, tt.typeName)
				if namedType == nil {
					t.Fatalf("Type %s not found in package", tt.typeName)
				}

				_, err := validator.ExtractName(pass, namedType)
				if err == nil {
					t.Errorf("ExtractName() expected error, got nil")
					return
				}

				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ExtractName() error = %v, want error containing %q", err, tt.errContains)
				}
			},
		)
	}
}

// TestNamerValidator_ExtractName_ExternalPackage tests validation via PackageLoader
func TestNamerValidator_ExtractName_ExternalPackage(t *testing.T) {
	// Load external package
	externalPkg, _ := LoadTestPackage(t, "namer_validator/external_package/external")

	// Load current package
	_, currentPass := LoadTestPackage(t, "namer_validator/external_package/current")

	// Create mock loader with external package
	loader := &MockPackageLoader{
		Packages: map[string]*packages.Package{
			"example.com/external": externalPkg,
		},
	}

	validator := NewNamerValidator(loader)

	// Get ExternalNamer type from external package
	namedType := FindNamedType(externalPkg, "ExternalNamer")
	if namedType == nil {
		t.Fatal("ExternalNamer not found in external package")
	}

	name, err := validator.ExtractName(currentPass, namedType)
	if err != nil {
		t.Fatalf("ExtractName() unexpected error: %v", err)
	}

	if name != "externalName" {
		t.Errorf("ExtractName() = %q, want %q", name, "externalName")
	}
}

// TestNamerValidator_ExtractName_ExternalPackageNotAvailable tests error when package loader unavailable
func TestNamerValidator_ExtractName_ExternalPackageNotAvailable(t *testing.T) {
	// Load external package
	externalPkg, _ := LoadTestPackage(t, "namer_validator/no_loader/external")

	// Load current package for pass
	_, currentPass := LoadTestPackage(t, "namer_validator/no_loader/current")

	// Create validator without loader
	validator := NewNamerValidator(nil)

	namedType := FindNamedType(externalPkg, "ExternalNamer")
	if namedType == nil {
		t.Fatal("ExternalNamer not found")
	}

	_, err := validator.ExtractName(currentPass, namedType)

	if err == nil {
		t.Error("ExtractName() expected error when loader is nil, got nil")
		return
	}

	if !strings.Contains(err.Error(), "no package loader available") {
		t.Errorf("ExtractName() error = %v, want error containing 'no package loader available'", err)
	}
	if !strings.Contains(err.Error(), "Define Namer in same package") {
		t.Errorf("ExtractName() error = %v, want error containing 'Define Namer in same package'", err)
	}
}

// TestNamerValidator_ExtractName_ExternalPackageLoadError tests error when package loading fails
func TestNamerValidator_ExtractName_ExternalPackageLoadError(t *testing.T) {
	// Load external package
	externalPkg, _ := LoadTestPackage(t, "namer_validator/load_error/external")

	// Load current package for pass
	_, currentPass := LoadTestPackage(t, "namer_validator/load_error/current")

	// Create loader that returns error (empty packages map)
	loader := &MockPackageLoader{
		Packages: map[string]*packages.Package{},
	}

	validator := NewNamerValidator(loader)

	namedType := FindNamedType(externalPkg, "ExternalNamer")
	if namedType == nil {
		t.Fatal("ExternalNamer not found")
	}

	_, err := validator.ExtractName(currentPass, namedType)
	if err == nil {
		t.Error("ExtractName() expected error when package loading fails, got nil")
		return
	}

	if !strings.Contains(err.Error(), "cannot validate external Namer") {
		t.Errorf("ExtractName() error = %v, want error containing 'cannot validate external Namer'", err)
	}
	if !strings.Contains(err.Error(), "Define Namer in same package") {
		t.Errorf("ExtractName() error = %v, want error containing 'Define Namer in same package'", err)
	}
}
