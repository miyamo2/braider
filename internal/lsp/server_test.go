package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

// sendRequest writes a JSON-RPC request with Content-Length framing into a buffer.
func buildRequest(method string, id any, params any) []byte {
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	if id != nil {
		req["id"] = id
	}
	body, _ := json.Marshal(req)
	return []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body))
}

// dispatchOne feeds one framed message to a fresh Server and returns the framed response bytes.
func dispatchOne(t *testing.T, base *Server, msgBytes []byte) []byte {
	t.Helper()

	var out bytes.Buffer
	srv := &Server{
		transport: newTransport(bytes.NewReader(msgBytes), &out),
		logger:    base.logger,
		openFiles: base.openFiles,
		providers: base.providers,
		injectors: base.injectors,
		variables: base.variables,
	}

	raw, err := srv.transport.readMessage()
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}

	var base2 struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &base2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var id any
	if len(base2.ID) > 0 && string(base2.ID) != "null" {
		_ = json.Unmarshal(base2.ID, &id)
	}

	if err := srv.dispatch(id, base2.Method, base2.Params); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	return out.Bytes()
}

// readResponse reads one framed JSON-RPC response from a byte slice.
func readResponse(t *testing.T, data []byte) ResponseMessage {
	t.Helper()
	var out bytes.Buffer
	tr := newTransport(bytes.NewReader(data), &out)
	body, err := tr.readMessage()
	if err != nil {
		t.Fatalf("readResponse: %v", err)
	}
	var resp ResponseMessage
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func newTestServer() *Server {
	s := NewServer(bytes.NewReader(nil), &bytes.Buffer{})
	return s
}

func TestInitialize(t *testing.T) {
	s := newTestServer()
	out := dispatchOne(t, s, buildRequest("initialize", 1, InitializeParams{
		RootURI: "file:///tmp/workspace",
	}))

	resp := readResponse(t, out)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result InitializeResult
	raw, _ := json.Marshal(resp.Result)
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}

	if !result.Capabilities.HoverProvider {
		t.Error("expected HoverProvider=true")
	}
	if !result.Capabilities.CodeActionProvider {
		t.Error("expected CodeActionProvider=true")
	}
	if result.Capabilities.CompletionProvider == nil {
		t.Error("expected CompletionProvider to be set")
	}
	if result.Capabilities.TextDocumentSync != 1 {
		t.Errorf("expected TextDocumentSync=1, got %d", result.Capabilities.TextDocumentSync)
	}
	if result.ServerInfo == nil || result.ServerInfo.Name != "braider-lsp" {
		t.Error("expected ServerInfo.Name=braider-lsp")
	}
}

func TestShutdown(t *testing.T) {
	s := newTestServer()
	out := dispatchOne(t, s, buildRequest("shutdown", 2, nil))
	resp := readResponse(t, out)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestMethodNotFound(t *testing.T) {
	s := newTestServer()
	out := dispatchOne(t, s, buildRequest("unknown/method", 3, nil))
	resp := readResponse(t, out)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != ErrMethodNotFound {
		t.Errorf("expected code %d, got %d", ErrMethodNotFound, resp.Error.Code)
	}
}

func TestDidOpenDidChange(t *testing.T) {
	s := newTestServer()

	const uri = "file:///tmp/test.go"
	const content1 = "package main\n"
	const content2 = "package main\n\nfunc main() {}\n"

	// didOpen (notification – no response expected)
	dispatchOne(t, s, buildRequest("textDocument/didOpen", nil, DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{URI: uri, LanguageID: "go", Version: 1, Text: content1},
	}))
	s.mu.RLock()
	got := s.openFiles[uri]
	s.mu.RUnlock()
	if got != content1 {
		t.Errorf("after didOpen: got %q, want %q", got, content1)
	}

	// didChange
	dispatchOne(t, s, buildRequest("textDocument/didChange", nil, DidChangeTextDocumentParams{
		TextDocument:   VersionedTextDocumentIdentifier{URI: uri, Version: 2},
		ContentChanges: []TextDocumentContentChangeEvent{{Text: content2}},
	}))
	s.mu.RLock()
	got = s.openFiles[uri]
	s.mu.RUnlock()
	if got != content2 {
		t.Errorf("after didChange: got %q, want %q", got, content2)
	}

	// didClose
	dispatchOne(t, s, buildRequest("textDocument/didClose", nil, DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}))
	s.mu.RLock()
	_, still := s.openFiles[uri]
	s.mu.RUnlock()
	if still {
		t.Error("file still present after didClose")
	}
}

func TestAnnotationKind(t *testing.T) {
	tests := []struct {
		name string
		want typeContext
	}{
		{"Provide", contextProvide},
		{"Injectable", contextInject},
		{"Variable", contextVariable},
		{"Other", contextNone},
		{"", contextNone},
	}
	for _, tc := range tests {
		if got := annotationKind(tc.name); got != tc.want {
			t.Errorf("annotationKind(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestURIConversions(t *testing.T) {
	cases := []struct {
		path string
		uri  string
	}{
		{"/home/user/foo.go", "file:///home/user/foo.go"},
		{"/tmp/bar/baz.go", "file:///tmp/bar/baz.go"},
	}
	for _, c := range cases {
		if got := filePathToURI(c.path); got != c.uri {
			t.Errorf("filePathToURI(%q) = %q, want %q", c.path, got, c.uri)
		}
		if got := uriToFilePath(c.uri); got != c.path {
			t.Errorf("uriToFilePath(%q) = %q, want %q", c.uri, got, c.path)
		}
	}
}

func TestCollectRegisteredTypeNames(t *testing.T) {
	s := newTestServer()
	s.providers["example.com/pkg.Foo"] = map[string]*providerEntry{"": {PackagePath: "example.com/pkg"}}
	s.injectors["example.com/pkg.Bar"] = map[string]*injectorEntry{"": {PackagePath: "example.com/pkg"}}
	s.variables["example.com/pkg.Cfg"] = map[string]*variableEntry{"": {PackagePath: "example.com/pkg"}}

	names := s.collectRegisteredTypeNames()

	for _, want := range []string{"example.com/pkg.Foo", "example.com/pkg.Bar", "example.com/pkg.Cfg"} {
		if !names[want] {
			t.Errorf("expected %q to be registered", want)
		}
	}
}

func TestLookupBinding(t *testing.T) {
	s := newTestServer()
	s.providers["example.com/pkg.Foo"] = map[string]*providerEntry{
		"": {ConstructorName: "NewFoo", PackagePath: "example.com/pkg"},
	}

	b := s.lookupBinding("example.com/pkg.Foo")
	if b == nil {
		t.Fatal("expected binding, got nil")
	}
	if b.Kind != "provide" {
		t.Errorf("kind: got %q, want %q", b.Kind, "provide")
	}
	if b.ConstructorFn != "NewFoo" {
		t.Errorf("constructor: got %q, want %q", b.ConstructorFn, "NewFoo")
	}

	if s.lookupBinding("example.com/pkg.Unknown") != nil {
		t.Error("expected nil for unknown type")
	}
}

func TestLookupBindingInjector(t *testing.T) {
	s := newTestServer()
	s.injectors["example.com/svc.Service"] = map[string]*injectorEntry{
		"": {ConstructorName: "NewService", PackagePath: "example.com/svc"},
	}

	b := s.lookupBinding("example.com/svc.Service")
	if b == nil {
		t.Fatal("expected binding, got nil")
	}
	if b.Kind != "inject" {
		t.Errorf("kind: got %q, want %q", b.Kind, "inject")
	}
}

func TestLookupBindingVariable(t *testing.T) {
	s := newTestServer()
	s.variables["example.com/cfg.Config"] = map[string]*variableEntry{
		"": {PackagePath: "example.com/cfg"},
	}

	b := s.lookupBinding("example.com/cfg.Config")
	if b == nil {
		t.Fatal("expected binding, got nil")
	}
	if b.Kind != "variable" {
		t.Errorf("kind: got %q, want %q", b.Kind, "variable")
	}
}

func TestIsAnnotationPkg(t *testing.T) {
	cases := []struct {
		pkg  string
		want bool
	}{
		{annotationProvidePkg, true},
		{"github.com/miyamo2/braider/pkg/annotation/inject", true},
		{"github.com/miyamo2/braider/pkg/annotation/provide", true},
		{"github.com/miyamo2/braider/pkg/annotation/variable", true},
		{"github.com/miyamo2/braider/pkg/annotation/app", true},
		{"github.com/miyamo2/braider/pkg/annotation/namer", true},
		{"example.com/user/myservice", false},
		{"github.com/miyamo2/braider/internal/registry", false},
	}
	for _, c := range cases {
		if got := isAnnotationPkg(c.pkg); got != c.want {
			t.Errorf("isAnnotationPkg(%q) = %v, want %v", c.pkg, got, c.want)
		}
	}
}

func TestTransportRoundTrip(t *testing.T) {
	msg := map[string]any{"jsonrpc": "2.0", "id": 1, "result": "ok"}
	body, _ := json.Marshal(msg)
	framed := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)

	var out bytes.Buffer
	tr := newTransport(bytes.NewBufferString(framed), &out)

	// Read back
	got, err := tr.readMessage()
	if err != nil {
		t.Fatalf("readMessage: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("body mismatch: got %q, want %q", got, body)
	}

	// Write
	if err := tr.writeMessage(msg); err != nil {
		t.Fatalf("writeMessage: %v", err)
	}
	written := out.String()
	if len(written) == 0 {
		t.Fatal("nothing written")
	}
	// Must start with Content-Length header
	if written[:16] != "Content-Length: " {
		t.Errorf("missing Content-Length header: %q", written[:min(len(written), 30)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
