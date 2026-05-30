package lsp

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/miyamo2/braider/internal/detect"
)

// Server is a minimal LSP server that provides DI annotation assistance for braider.
type Server struct {
	transport *transport
	logger    *log.Logger

	// mu guards the fields below.
	mu sync.RWMutex

	// openFiles tracks the current text content of open documents (URI → content).
	openFiles map[string]string

	// Cached registration info populated by the background workspace analysis pass.
	// Outer key is the fully qualified type name; inner key is the binding name
	// (empty string for the default/unnamed binding).
	providers map[string]map[string]*providerEntry
	injectors map[string]map[string]*injectorEntry
	variables map[string]map[string]*variableEntry

	// shutdownRequested is set when the client sends a shutdown request.
	shutdownRequested bool
}

// providerEntry is a lightweight cache of Provide annotation fields.
type providerEntry struct {
	ConstructorName string
	PackagePath     string
	Name            string
}

// injectorEntry is a lightweight cache of Injectable annotation fields.
type injectorEntry struct {
	ConstructorName string
	PackagePath     string
	Name            string
}

// variableEntry is a lightweight cache of Variable annotation fields.
type variableEntry struct {
	PackagePath string
	Name        string
}

// NewServer creates a new LSP server that reads from r and writes to w.
func NewServer(r io.Reader, w io.Writer) *Server {
	return &Server{
		transport: newTransport(r, w),
		logger:    log.New(os.Stderr, "[braider-lsp] ", log.LstdFlags),
		openFiles: make(map[string]string),
		providers: make(map[string]map[string]*providerEntry),
		injectors: make(map[string]map[string]*injectorEntry),
		variables: make(map[string]map[string]*variableEntry),
	}
}

// Run is the main loop. It reads messages and dispatches them until the
// connection is closed or a fatal error occurs.
func (s *Server) Run() error {
	for {
		raw, err := s.transport.readMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("reading message: %w", err)
		}

		var base struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			s.logger.Printf("malformed message: %v", err)
			continue
		}

		var id any
		if len(base.ID) > 0 && string(base.ID) != "null" {
			var rawID any
			if err := json.Unmarshal(base.ID, &rawID); err == nil {
				id = rawID
			}
		}

		if err := s.dispatch(id, base.Method, base.Params); err != nil {
			s.logger.Printf("dispatch %s: %v", base.Method, err)
		}
	}
}

// dispatch routes an incoming JSON-RPC message to the appropriate handler.
func (s *Server) dispatch(id any, method string, rawParams json.RawMessage) error {
	switch method {
	case "initialize":
		return s.handleInitialize(id, rawParams)
	case "initialized":
		go s.analyzeWorkspace()
		return nil
	case "shutdown":
		s.mu.Lock()
		s.shutdownRequested = true
		s.mu.Unlock()
		return s.sendResult(id, nil)
	case "exit":
		s.mu.RLock()
		shutdown := s.shutdownRequested
		s.mu.RUnlock()
		if shutdown {
			os.Exit(0)
		}
		os.Exit(1)
	case "textDocument/didOpen":
		return s.handleDidOpen(rawParams)
	case "textDocument/didChange":
		return s.handleDidChange(rawParams)
	case "textDocument/didClose":
		return s.handleDidClose(rawParams)
	case "textDocument/completion":
		return s.handleCompletion(id, rawParams)
	case "textDocument/hover":
		return s.handleHover(id, rawParams)
	case "textDocument/codeAction":
		return s.handleCodeAction(id, rawParams)
	default:
		if id != nil {
			return s.sendError(id, ErrMethodNotFound, fmt.Sprintf("method not found: %s", method))
		}
	}
	return nil
}

// handleInitialize responds to the initialize request with server capabilities.
func (s *Server) handleInitialize(id any, rawParams json.RawMessage) error {
	var params InitializeParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return s.sendError(id, ErrInvalidParams, "invalid initialize params")
	}

	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: 1, // Full sync
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{"[", ","},
			},
			HoverProvider:      true,
			CodeActionProvider: true,
		},
		ServerInfo: &ServerInfo{
			Name:    "braider-lsp",
			Version: "0.1.0",
		},
	}
	return s.sendResult(id, result)
}

// handleDidOpen tracks newly opened files.
func (s *Server) handleDidOpen(rawParams json.RawMessage) error {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}
	s.mu.Lock()
	s.openFiles[params.TextDocument.URI] = params.TextDocument.Text
	s.mu.Unlock()
	return nil
}

// handleDidChange updates the in-memory content of a file.
func (s *Server) handleDidChange(rawParams json.RawMessage) error {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}
	if len(params.ContentChanges) == 0 {
		return nil
	}
	// Full sync: use the last change event's text.
	s.mu.Lock()
	s.openFiles[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
	s.mu.Unlock()
	return nil
}

// handleDidClose removes a file from the in-memory store.
func (s *Server) handleDidClose(rawParams json.RawMessage) error {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.openFiles, params.TextDocument.URI)
	s.mu.Unlock()
	return nil
}

// overlayForFile returns a go/packages overlay map for the given file path if
// an unsaved (in-memory) version exists.
func (s *Server) overlayForFile(path string) map[string][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uri := filePathToURI(path)
	if content, ok := s.openFiles[uri]; ok {
		return map[string][]byte{path: []byte(content)}
	}
	return nil
}

// sendResult sends a successful JSON-RPC response.
func (s *Server) sendResult(id any, result any) error {
	return s.transport.writeMessage(ResponseMessage{
		Message: Message{JSONRPC: "2.0"},
		ID:      id,
		Result:  result,
	})
}

// sendError sends a JSON-RPC error response.
func (s *Server) sendError(id any, code int, msg string) error {
	return s.transport.writeMessage(ResponseMessage{
		Message: Message{JSONRPC: "2.0"},
		ID:      id,
		Error:   &ResponseErr{Code: code, Message: msg},
	})
}

// analyzeWorkspace performs a best-effort workspace scan to populate the
// server's registration caches with all braider annotations visible under
// the current working directory.  Runs in the background after "initialized".
func (s *Server) analyzeWorkspace() {
	cwd, err := os.Getwd()
	if err != nil {
		s.logger.Printf("analyzeWorkspace: getwd: %v", err)
		return
	}

	// Resolve module path and marker interfaces outside any lock.
	// Both are best-effort: failures degrade gracefully.
	modPath, _ := detect.ModulePath()
	markers, _ := detect.ResolveMarkers()

	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps,
		Fset: fset,
		Dir:  cwd,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		s.logger.Printf("analyzeWorkspace: load: %v", err)
		return
	}

	// Build caches in local maps without holding any lock.
	localProviders := make(map[string]map[string]*providerEntry)
	localInjectors := make(map[string]map[string]*injectorEntry)
	localVariables := make(map[string]map[string]*variableEntry)

	for _, pkg := range pkgs {
		scanPackageForAnnotations(pkg, modPath, localProviders, localInjectors, localVariables)
		scanPackageForInjectables(pkg, markers, localInjectors)
	}

	// Swap in the results under a brief write lock.
	s.mu.Lock()
	s.providers = localProviders
	s.injectors = localInjectors
	s.variables = localVariables
	s.mu.Unlock()

	s.logger.Printf("workspace analysis complete: %d packages scanned", len(pkgs))
}

// scanPackageForAnnotations examines a package's AST for braider Provide/Variable
// call expressions and writes results into the supplied local maps.
func scanPackageForAnnotations(
	pkg *packages.Package,
	modPath string,
	providers map[string]map[string]*providerEntry,
	injectors map[string]map[string]*injectorEntry,
	variables map[string]map[string]*variableEntry,
) {
	if pkg.TypesInfo == nil {
		return
	}
	for _, f := range pkg.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			switch expr := n.(type) {
			case *ast.IndexExpr:
				tryRegisterAnnotation(pkg, expr.X, expr.Index, modPath, providers, injectors, variables)
			case *ast.IndexListExpr:
				for _, idx := range expr.Indices {
					tryRegisterAnnotation(pkg, expr.X, idx, modPath, providers, injectors, variables)
				}
			}
			return true
		})
	}
}

// scanPackageForInjectables detects exported structs that embed annotation.Injectable[T]
// via types.Implements and registers them as injector entries.
func scanPackageForInjectables(
	pkg *packages.Package,
	markers *detect.MarkerInterfaces,
	injectors map[string]map[string]*injectorEntry,
) {
	if pkg.Types == nil || markers == nil || markers.Injectable == nil {
		return
	}
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok || !tn.Exported() {
			continue
		}
		t := tn.Type()
		if !types.Implements(t, markers.Injectable) && !types.Implements(types.NewPointer(t), markers.Injectable) {
			continue
		}
		typeName := pkg.PkgPath + "." + name
		if injectors[typeName] == nil {
			injectors[typeName] = make(map[string]*injectorEntry)
		}
		if _, exists := injectors[typeName][""]; !exists {
			injectors[typeName][""] = &injectorEntry{PackagePath: pkg.PkgPath}
		}
	}
}

// tryRegisterAnnotation verifies that fnExpr is a braider annotation function
// (by checking its resolved package path, not just its name) and records the
// type argument in the appropriate local map.
func tryRegisterAnnotation(
	pkg *packages.Package,
	fnExpr ast.Expr,
	typeArgExpr ast.Expr,
	modPath string,
	providers map[string]map[string]*providerEntry,
	injectors map[string]map[string]*injectorEntry,
	variables map[string]map[string]*variableEntry,
) {
	// Extract the identifier for the annotation function (e.g. "Provide").
	var ident *ast.Ident
	switch x := fnExpr.(type) {
	case *ast.SelectorExpr:
		ident = x.Sel
	case *ast.Ident:
		ident = x
	default:
		return
	}

	kind := annotationKind(ident.Name)
	if kind == contextNone {
		return
	}

	// Verify the function object comes from the braider annotation package so
	// that user-defined functions named "Provide"/"Injectable"/"Variable" in
	// unrelated packages are not mistakenly registered.
	obj := pkg.TypesInfo.ObjectOf(ident)
	if obj == nil || obj.Pkg() == nil {
		return
	}
	if !isAnnotationPkg(obj.Pkg().Path(), modPath) {
		return
	}

	typeArgType := pkg.TypesInfo.TypeOf(typeArgExpr)
	if typeArgType == nil {
		return
	}

	// Dereference pointer to get the base named type.
	base := typeArgType
	if ptr, ok := base.(*types.Pointer); ok {
		base = ptr.Elem()
	}

	named, ok := base.(*types.Named)
	if !ok {
		return
	}
	// Skip option types that belong to annotation sub-packages.
	if named.Obj().Pkg() == nil {
		return
	}
	pkgPath := named.Obj().Pkg().Path()
	if isAnnotationPkg(pkgPath, modPath) {
		return
	}

	typeName := pkgPath + "." + named.Obj().Name()

	switch kind {
	case contextProvide:
		if providers[typeName] == nil {
			providers[typeName] = make(map[string]*providerEntry)
		}
		if _, exists := providers[typeName][""]; !exists {
			providers[typeName][""] = &providerEntry{PackagePath: pkg.PkgPath}
		}
	case contextInject:
		if injectors[typeName] == nil {
			injectors[typeName] = make(map[string]*injectorEntry)
		}
		if _, exists := injectors[typeName][""]; !exists {
			injectors[typeName][""] = &injectorEntry{PackagePath: pkg.PkgPath}
		}
	case contextVariable:
		if variables[typeName] == nil {
			variables[typeName] = make(map[string]*variableEntry)
		}
		if _, exists := variables[typeName][""]; !exists {
			variables[typeName][""] = &variableEntry{PackagePath: pkg.PkgPath}
		}
	}
}

// isAnnotationPkg reports whether pkgPath belongs to the braider annotation API.
// modPath is the binary's module path (from detect.ModulePath); when non-empty it
// is used for a fork-safe prefix check.  When modPath is empty it is resolved via
// detect.ModulePath so the check remains correct in forked modules.
// Path boundaries are checked explicitly (exact match or "/" separator) to
// prevent false positives from paths like ".../pkg/annotationevil".
func isAnnotationPkg(pkgPath, modPath string) bool {
	if modPath == "" {
		modPath, _ = detect.ModulePath()
	}
	if modPath == "" {
		return false
	}
	pkgAnnotation := modPath + "/pkg/annotation"
	internalAnnotation := modPath + "/internal/annotation"
	return pkgPath == pkgAnnotation || strings.HasPrefix(pkgPath, pkgAnnotation+"/") ||
		pkgPath == internalAnnotation || strings.HasPrefix(pkgPath, internalAnnotation+"/")
}

// findSyntaxFile returns the *ast.File in pkg.Syntax whose path matches filePath.
func findSyntaxFile(pkg *packages.Package, filePath string) *ast.File {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	for _, f := range pkg.Syntax {
		pos := pkg.Fset.Position(f.Pos())
		if pos.Filename == absPath || pos.Filename == filePath {
			return f
		}
	}
	return nil
}
