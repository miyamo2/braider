package detect

import (
	"go/token"
	"go/types"
	"testing"
)

// TestVoidMarkerSeal proves that a void unexported marker method
// (e.g., _IsInjectable()) seals the interface by package path identity.
//
// Even without a named return type, types.Implements rejects types
// whose methods originate from a different package, because
// Named.lookupMethod gates on samePkg (path comparison) for unexported names.
//
// Call chain:
//
//	types.Implements(V, T)                                        // api_predicates.go:53
//	  → (*Checker)(nil).implements(V, T, false, nil)              // api_predicates.go:63
//	    → check.hasAllMethods(V, T, true, Identical, cause)       // instantiate.go:288
//	      → check.missingMethod(V, T, static, equivalent, cause) // lookup.go:533
//	        → lookupFieldOrMethodImpl(V, false, m.pkg, m.name, false) // lookup.go:419
//	          → named.lookupMethod(pkg, name, foldCase)           // lookup.go:195
//	            → samePkg(n.obj.pkg, pkg)                         // named.go:610
//	              → a.path == b.path                              // predicates.go:232
func TestVoidMarkerSeal(t *testing.T) {
	const internalAnnotationPath = "github.com/miyamo2/braider/internal/annotation"
	const externalPkgPath = "example.com/foo"

	// --- Build the marker interface in internal/annotation ---
	markerPkg := types.NewPackage(internalAnnotationPath, "annotation")
	markerSig := types.NewSignatureType(nil, nil, nil, nil, nil, false) // func()
	markerMethod := types.NewFunc(token.NoPos, markerPkg, "_IsInjectable", markerSig)
	markerIface := types.NewInterfaceType([]*types.Func{markerMethod}, nil)
	markerIface.Complete()

	// Also register Injectable as a named interface in the marker package
	injectableTypeName := types.NewTypeName(token.NoPos, markerPkg, "Injectable", nil)
	injectableNamed := types.NewNamed(injectableTypeName, markerIface, nil)
	markerPkg.Scope().Insert(injectableNamed.Obj())
	markerPkg.MarkComplete()

	// --- Case 1: External type with same method name → must NOT implement ---
	// This is the primary seal proof: func (Fake) _IsInjectable() in example.com/foo
	// has method pkg "example.com/foo", while the marker interface requires
	// method pkg "github.com/miyamo2/braider/internal/annotation".
	// samePkg returns false → lookupMethod skips → missingMethod reports not found.
	t.Run("external_package_same_method_rejected", func(t *testing.T) {
		externalPkg := types.NewPackage(externalPkgPath, "foo")

		// type Fake struct{}
		fakeStruct := types.NewStruct(nil, nil)
		fakeTypeName := types.NewTypeName(token.NoPos, externalPkg, "Fake", nil)
		fakeNamed := types.NewNamed(fakeTypeName, fakeStruct, nil)

		// func (Fake) _IsInjectable() — method belongs to externalPkg
		recv := types.NewVar(token.NoPos, externalPkg, "", fakeNamed)
		sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
		method := types.NewFunc(token.NoPos, externalPkg, "_IsInjectable", sig)
		fakeNamed.AddMethod(method)

		if types.Implements(fakeNamed, markerIface) {
			t.Fatal("types.Implements must return false for external type with same void method name")
		}
	})

	// --- Case 2: Type embedding the actual marker interface → must implement ---
	t.Run("embedding_marker_interface_accepted", func(t *testing.T) {
		annotationPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation", "annotation")

		// type Injectable struct { annotation.Injectable }  (embedding the marker interface)
		embeddedField := types.NewField(token.NoPos, nil, "", injectableNamed, true)
		injectStruct := types.NewStruct([]*types.Var{embeddedField}, nil)
		injectTypeName := types.NewTypeName(token.NoPos, annotationPkg, "Injectable", nil)
		injectNamed := types.NewNamed(injectTypeName, injectStruct, nil)

		if !types.Implements(injectNamed, markerIface) {
			t.Fatal("types.Implements must return true for type embedding the marker interface")
		}
	})

	// --- Case 3: User struct embedding through pkg/annotation → must implement ---
	t.Run("user_struct_embedding_annotation_accepted", func(t *testing.T) {
		annotationPkg := types.NewPackage("github.com/miyamo2/braider/pkg/annotation", "annotation")

		// Build pkg/annotation.Injectable that embeds internal/annotation.Injectable
		embeddedField := types.NewField(token.NoPos, nil, "", injectableNamed, true)
		pkgInjectStruct := types.NewStruct([]*types.Var{embeddedField}, nil)
		pkgInjectTypeName := types.NewTypeName(token.NoPos, annotationPkg, "Injectable", nil)
		pkgInjectNamed := types.NewNamed(pkgInjectTypeName, pkgInjectStruct, nil)

		// Build user's MyService struct embedding pkg/annotation.Injectable
		userPkg := types.NewPackage("example.com/foo", "foo")
		userField := types.NewField(token.NoPos, nil, "", pkgInjectNamed, true)
		userStruct := types.NewStruct([]*types.Var{userField}, nil)
		userTypeName := types.NewTypeName(token.NoPos, userPkg, "MyService", nil)
		userNamed := types.NewNamed(userTypeName, userStruct, nil)

		if !types.Implements(userNamed, markerIface) {
			t.Fatal("types.Implements must return true for user struct embedding annotation.Injectable")
		}
	})

	// --- Case 4: Completely unrelated type → must NOT implement ---
	t.Run("unrelated_type_rejected", func(t *testing.T) {
		unrelatedPkg := types.NewPackage("example.com/bar", "bar")
		unrelatedStruct := types.NewStruct(nil, nil)
		unrelatedTypeName := types.NewTypeName(token.NoPos, unrelatedPkg, "Other", nil)
		unrelatedNamed := types.NewNamed(unrelatedTypeName, unrelatedStruct, nil)

		if types.Implements(unrelatedNamed, markerIface) {
			t.Fatal("types.Implements must return false for unrelated type")
		}
	})

	// --- Case 5: AddMethod itself enforces package identity ---
	// Named.AddMethod asserts samePkg(t.obj.pkg, m.pkg), so even the go/types API
	// prevents creating a method with a mismatched package on a named type.
	// This further strengthens the seal: it is impossible to forge a method
	// belonging to internal/annotation on a type defined in another package.
	t.Run("AddMethod_enforces_package_identity", func(t *testing.T) {
		externalPkg := types.NewPackage(externalPkgPath, "foo")
		fakeStruct := types.NewStruct(nil, nil)
		fakeTypeName := types.NewTypeName(token.NoPos, externalPkg, "Forged", nil)
		fakeNamed := types.NewNamed(fakeTypeName, fakeStruct, nil)

		recv := types.NewVar(token.NoPos, externalPkg, "", fakeNamed)
		sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
		// Create a method with markerPkg (internal/annotation) on a type from externalPkg
		method := types.NewFunc(token.NoPos, markerPkg, "_IsInjectable", sig)

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("AddMethod must panic when method package differs from type package")
			}
		}()
		fakeNamed.AddMethod(method)
	})
}
