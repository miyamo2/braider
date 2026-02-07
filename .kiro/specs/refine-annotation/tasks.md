# Implementation Plan

## Task Breakdown

- [ ] 1. Implement public API layer for generic annotations
- [x] 1.1 (P) Update annotation package with generic interfaces
  - Define `Injectable[T inject.Option]` interface with `isInjectable()` and `option() T` marker methods
  - Define `Provider[T provide.Option]` interface with `isProvider()` and `option() T` marker methods
  - Implement `Provide[T](providerFunc)` generic function returning `Provider[T]`
  - Add backward compatibility type alias `Inject = Injectable[inject.Default]` and `Provide = Provider[provide.Default]`
  - _Requirements: 1.1, 1.2_

- [x] 1.2 (P) Implement inject option interfaces
  - Define base `inject.Option` interface with `isOption() option` marker method
  - Define `inject.Default` interface extending `Option` with `isDefault()` marker method
  - Define `inject.Typed[T any]` interface extending `Option` with `typed() T` marker method
  - Define `inject.Named[T namer.Namer]` interface extending `Option` with `named() T` marker method
  - Define `inject.WithoutConstructor` interface extending `Option` with `withoutConstructor()` marker method
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 5.1_

- [x] 1.3 (P) Implement provide option interfaces
  - Define base `provide.Option` interface with `isOption() option` marker method
  - Define `provide.Default` interface extending `Option` with `isDefault()` marker method
  - Define `provide.Typed[T any]` interface extending `Option` with `typed() T` marker method
  - Define `provide.Named[T namer.Namer]` interface extending `Option` with `named() T` marker method
  - _Requirements: 3.1, 3.2, 3.3, 5.2_

- [x] 2. Extend registry domain with option metadata support
- [x] 2.1 Define option metadata structures
  - Create `OptionMetadata` struct in `internal/detect` with fields: `IsDefault bool`, `TypedInterface types.Type`, `Name string`, `WithoutConstructor bool`
  - Extend `InjectorInfo` struct with `RegisteredType types.Type`, `Name string`, `OptionMetadata OptionMetadata` fields
  - Extend `ProviderInfo` struct with `RegisteredType types.Type`, `Name string`, `OptionMetadata OptionMetadata` fields
  - _Requirements: 2.6, 2.7, 3.4, 3.5, 3.6_

- [x] 2.2 Implement package cache for external AST loading
  - Create `PackageCache` struct in `internal/registry` with `sync.RWMutex` and `map[string]*packages.Package` fields
  - Implement `NewPackageCache()` constructor returning initialized cache
  - Implement `Get(pkgPath string)` method returning cached package or nil
  - Implement `Set(pkgPath string, pkg *packages.Package)` method storing package in cache with thread-safe locking
  - _Requirements: 4.3_

- [x] 2.3 Add named dependency lookup support to registries
  - Add `GetByName(typeName, name string) (*InjectorInfo, bool)` method to `InjectorRegistry`
  - Add `GetByName(typeName, name string) (*ProviderInfo, bool)` method to `ProviderRegistry`
  - Implement duplicate `(TypeName, Name)` pair validation in `InjectorRegistry.Register()` with correlation error diagnostic
  - Implement duplicate `(TypeName, Name)` pair validation in `ProviderRegistry.Register()` with correlation error diagnostic
  - _Requirements: 4.5_

- [ ] 3. Implement detection domain components for option extraction
- [ ] 3.1 Create PackageLoader for external package AST access
  - Create `PackageLoader` interface in `internal/detect` with `LoadPackage(pkgPath string) (*packages.Package, error)` method
  - Implement `packageLoaderImpl` struct holding reference to `PackageCache`
  - Implement `LoadPackage()` checking cache first, then calling `packages.Load()` with `NeedSyntax|NeedTypes` mode on cache miss
  - Handle package loading errors with diagnostic message indicating source unavailability
  - Store loaded package in `PackageCache` before returning
  - _Requirements: 4.3_

- [ ] 3.2 Create NamerValidator for literal validation
  - Create `NamerValidator` interface in `internal/detect` with `ExtractName(pass, namerType) (string, error)` method
  - Implement validator using `types.LookupFieldOrMethod()` to find `Name() string` method on namer type
  - For same-package Namers, traverse `pass.Files` AST to locate method declaration
  - For external-package Namers, use `PackageLoader.LoadPackage()` to retrieve cached AST
  - Traverse `*ast.FuncDecl` body to locate `*ast.ReturnStmt` and validate `Results[0]` is `*ast.BasicLit` with `token.STRING`
  - Return extracted name string with quotes stripped, or error with diagnostic position for non-literal returns
  - _Requirements: 4.2, 4.3, 8.3_

- [ ] 3.3 Implement OptionExtractor for type parameter analysis
  - Create `OptionExtractor` interface in `internal/detect` with `ExtractInjectOptions(pass, fieldType, concreteType)` and `ExtractProvideOptions(pass, callExpr, providerFunc)` methods
  - Implement type parameter extraction using `types.Named.TypeArgs().At(0)` from generic instantiations
  - Validate type parameter implements `inject.Option` or `provide.Option` constraint using `types.Implements()`
  - Detect which option interfaces type parameter satisfies via interface implementation checks
  - For `Typed[I]` option, extract interface type `I` and validate concrete type implements it using `types.Implements()`
  - For `Named[N]` option, extract namer type `N` and delegate to `NamerValidator.ExtractName()`
  - Detect conflicting options (Default + WithoutConstructor, Default + Typed) and return error
  - Return populated `OptionMetadata` struct with all extracted configuration
  - _Requirements: 1.3, 1.4, 1.5, 2.6, 2.7, 3.5, 3.6, 3.7, 5.3, 5.4, 5.5, 7.5, 8.1, 8.2_

- [ ] 3.4 Update InjectDetector to support Injectable[T] interface
  - Modify `isNamedInjectType()` method to detect both `annotation.Inject` struct and `annotation.Injectable[T]` interface types
  - Update `FindInjectField()` to handle generic interface type checking via `types.Named` with non-empty `TypeArgs()`
  - Integrate `OptionExtractor.ExtractInjectOptions()` call after detecting `Injectable[T]` field
  - Store returned `OptionMetadata` in detection result for registry registration
  - _Requirements: 1.3, 2.5, 2.6, 2.7, 2.8_

- [ ] 3.5 Update ProvideDetector to support Provider[T] annotation
  - Modify provider detection logic to recognize `annotation.Provide[T](fn)` call expressions
  - Extract type parameter from `Provide[T]` generic function call using `types.Named.TypeArgs()`
  - Integrate `OptionExtractor.ExtractProvideOptions()` call after detecting provider annotation
  - Validate provider function return type compatibility with `Typed[I]` interface if present
  - Store returned `OptionMetadata` in detection result for registry registration
  - _Requirements: 1.4, 3.4, 3.5, 3.6, 3.7_

- [ ] 4. Integrate option extraction into DependencyAnalyzer with error handling
- [ ] 4.1 Add context cancellation for fatal validation errors
  - Create `context.WithCancel()` from `pass.Context` at start of `DependencyAnalyzer.Run()`
  - When `OptionExtractor` returns validation error (constraint violation, interface implementation failure, non-literal Namer), call `pass.Report()` with diagnostic and `cancel()` context
  - Pass cancelled context to downstream components to halt processing
  - Update `InjectorRegistry.Register()` and `ProviderRegistry.Register()` to populate `OptionMetadata`, `RegisteredType`, `Name` fields from extractor output
  - _Requirements: 8.5_

- [ ] 4.2 Handle correlation errors as non-fatal
  - When duplicate `(TypeName, Name)` pair detected during registration, emit diagnostic via `pass.Report()` but do not cancel context
  - Continue processing remaining dependencies to report all correlation errors in single pass
  - _Requirements: 8.6_

- [ ] 5. Extend ConstructorGenerator for option-based code generation
- [ ] 5.1 Implement option-aware return type selection
  - Check `info.OptionMetadata.WithoutConstructor` flag; if true, skip generation and emit validation diagnostic if manual constructor missing
  - Check `info.OptionMetadata.TypedInterface`; if non-nil, use interface type as constructor return type rendered via `types.TypeString(registeredType, qualifier)`
  - Default to `*ConcreteStruct` return type for `inject.Default` or no option
  - _Requirements: 2.5, 2.8, 6.1, 6.2, 6.4_

- [ ] 5.2 Implement named dependency parameter naming
  - When constructor depends on named dependencies, look up dependency names from `InjectorRegistry.GetByName()` and `ProviderRegistry.GetByName()`
  - Use extracted `Name` field for parameter identifier instead of default `lowerCamelCase(TypeName)`
  - Ensure parameter names are valid Go identifiers and do not conflict with keywords
  - _Requirements: 6.3_

- [ ] 6. Extend BootstrapGenerator for typed and named variables
- [ ] 6.1 Implement interface-typed variable declarations
  - For each `InjectorInfo`, use `info.RegisteredType` for variable type instead of concrete struct type
  - Render variable type using `types.TypeString(info.RegisteredType, qualifier)` to handle package-qualified interface types
  - For `ProviderInfo` with `Typed[I]`, declare variable with interface type `I` and assign provider function result
  - _Requirements: 7.1, 7.2_

- [ ] 6.2 Implement named variable naming
  - Check `info.Name` field; if non-empty, use as variable identifier in IIFE bootstrap code
  - Default to `lowerCamelCase(info.LocalName)` for unnamed dependencies
  - Validate variable name uniqueness within IIFE scope using registry duplicate checks
  - _Requirements: 4.4_

- [ ] 6.3 Update topological sort for named dependencies
  - Treat named dependencies as distinct nodes in dependency graph with key `(TypeName, Name)`
  - Update dependency edge resolution to support looking up both unnamed and named dependencies
  - Ensure initialization order respects dependency edges for both typed and named dependencies
  - _Requirements: 7.4_

- [ ] 7. Update AppAnalyzer to check context cancellation
- [ ] 7.1 Add context cancellation check before bootstrap generation
  - At start of `AppAnalyzer.Run()`, check `ctx.Done()` channel from shared context
  - If context cancelled, skip bootstrap code generation and return early without error
  - If context active, proceed with normal bootstrap generation using updated `BootstrapGenerator`
  - _Requirements: 8.5_

- [ ] 8. Implement comprehensive test coverage
- [ ] 8.1 (P) Unit tests for OptionExtractor
  - Test type parameter extraction for each option type: Default, Typed[I], Named[N], WithoutConstructor
  - Test mixed-in option types implementing multiple interfaces (Typed[I] + Named[N])
  - Test constraint violation errors when type parameter does not implement inject.Option or provide.Option
  - Test interface implementation validation errors when concrete type does not implement Typed[I]
  - Test conflicting option detection (Default + WithoutConstructor)
  - _Requirements: 1.3, 1.4, 1.5, 5.3, 5.4, 5.5, 7.5, 8.1, 8.2_

- [ ] 8.2 (P) Unit tests for NamerValidator
  - Test hardcoded string literal detection from Name() method body
  - Test rejection of computed values (concatenation, variables, function calls)
  - Test error diagnostic for method not found or invalid signature
  - Test same-package Namer validation via pass.Files AST traversal
  - Test external-package Namer validation via PackageLoader
  - _Requirements: 4.2, 4.3, 8.3_

- [ ] 8.3 (P) Unit tests for registry extensions
  - Test `InjectorRegistry.Register()` with OptionMetadata, RegisteredType, Name fields populated
  - Test `GetByName()` lookup for named dependencies
  - Test duplicate (TypeName, Name) pair detection and correlation error diagnostic
  - Test unnamed and named dependencies coexistence in same registry
  - _Requirements: 2.6, 2.7, 4.5_

- [ ] 8.4 (P) Unit tests for ConstructorGenerator extensions
  - Test constructor return type selection: *ConcreteStruct for Default, interface I for Typed[I]
  - Test WithoutConstructor skip logic and validation diagnostic emission
  - Test named dependency parameter naming using registry lookups
  - _Requirements: 2.5, 2.8, 6.1, 6.2, 6.3, 6.4_

- [ ] 8.5 (P) Unit tests for BootstrapGenerator extensions
  - Test interface-typed variable declarations for Injectable[Typed[I]] and Provide[Typed[I]]
  - Test named variable naming using info.Name field
  - Test topological sort preservation for typed and named dependencies
  - _Requirements: 4.4, 7.1, 7.2, 7.4_

- [ ] 8.6 Integration tests for complete annotation flows
  - Test Injectable[Typed[I]] flow: struct with interface annotation → constructor returns interface → bootstrap declares interface variable
  - Test Named dependency flow: multiple structs with same type but different names → separate variables with unique names
  - Test mixed options flow: custom option type implementing Typed[I] + Named[N] → constructor returns interface, variable uses custom name
  - Test Provide[Typed[I]] flow: provider function annotated with interface type → bootstrap assigns to interface variable
  - Test error handling flow: constraint violation → fatal diagnostic → AppAnalyzer skipped
  - Test correlation error flow: duplicate names → non-fatal diagnostic → bootstrap generation continues
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.5, 2.6, 2.7, 2.8, 3.4, 3.5, 3.6, 3.7, 4.2, 4.3, 4.4, 4.5, 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2, 7.4, 7.5, 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

- [ ] 8.7 (P) E2E tests using analysistest
  - Create testdata/refine_annotation/typed_inject with Injectable[Typed[I]] struct, verify constructor signature and bootstrap code
  - Create testdata/refine_annotation/named_inject with multiple Named[N] structs, verify distinct variables and no name collision
  - Create testdata/refine_annotation/without_constructor with WithoutConstructor option, verify no constructor generated and manual constructor used
  - Create testdata/refine_annotation/provide_typed with Provide[Typed[I]] function, verify bootstrap interface assignment
  - Create testdata/refine_annotation/error_cases with constraint violations, non-literal names, duplicate names, verify diagnostics
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 2.5, 2.6, 2.7, 2.8, 3.4, 3.5, 3.6, 3.7, 4.2, 4.3, 4.4, 4.5, 8.1, 8.2, 8.3, 8.4_

- [ ] 9. Add godoc documentation and examples
- [ ] 9.1 (P) Document public API interfaces
  - Add godoc comments to Injectable[T] and Provider[T] interfaces with usage examples showing Default, Typed[I], Named[N], WithoutConstructor patterns
  - Add godoc comments to inject.Option, inject.Default, inject.Typed[T], inject.Named[T], inject.WithoutConstructor interfaces with code examples
  - Add godoc comments to provide.Option, provide.Default, provide.Typed[T], provide.Named[T] interfaces with code examples
  - Add godoc comments to namer.Namer interface explaining hardcoded literal requirement
  - _Requirements: 9.1, 9.2, 9.3_

- [ ] 9.2 (P) Create example projects
  - Create example project demonstrating interface-typed dependencies with Injectable[Typed[I]] and Provide[Typed[I]]
  - Create example project demonstrating named dependencies with Injectable[Named[N]] for multiple instances of same type
  - Create example project demonstrating custom constructors with Injectable[WithoutConstructor]
  - Create example project demonstrating mixed options with custom types implementing multiple option interfaces
  - _Requirements: 9.4_

- [ ] 9.3 (P) Update README documentation
  - Document generic annotation pattern with Injectable[T] and Provider[T] syntax
  - Explain option types: Default, Typed[I], Named[N], WithoutConstructor
  - Link to example projects for each pattern
  - Document migration path from old Inject struct to new Injectable[inject.Default] interface
  - _Requirements: 9.5_
