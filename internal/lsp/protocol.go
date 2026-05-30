// Package lsp provides a minimal LSP server for braider DI annotation assistance.
package lsp

// LSP message types and protocol structures (JSON-RPC 2.0 / LSP 3.17 subset).

// Message is the base JSON-RPC 2.0 message envelope.
type Message struct {
	JSONRPC string `json:"jsonrpc"`
}

// RequestMessage is an incoming JSON-RPC request.
type RequestMessage struct {
	Message
	ID     any    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// ResponseMessage is an outgoing JSON-RPC response.
type ResponseMessage struct {
	Message
	ID     any          `json:"id"`
	Result any          `json:"result,omitempty"`
	Error  *ResponseErr `json:"error,omitempty"`
}

// NotificationMessage is an outgoing/incoming notification (no ID).
type NotificationMessage struct {
	Message
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// ResponseErr represents an LSP error response.
type ResponseErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error codes defined by JSON-RPC 2.0 and LSP.
const (
	ErrParseError     = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternalError  = -32603
)

// InitializeParams is the parameter for the initialize request.
type InitializeParams struct {
	RootURI        string             `json:"rootUri"`
	RootPath       string             `json:"rootPath,omitempty"`
	Capabilities   ClientCapabilities `json:"capabilities"`
	InitOptions    any                `json:"initializationOptions,omitempty"`
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders,omitempty"`
}

// ClientCapabilities is an abbreviated subset of LSP ClientCapabilities.
type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
}

// TextDocumentClientCapabilities is a subset of LSP text document capabilities.
type TextDocumentClientCapabilities struct {
	Completion *CompletionClientCapabilities `json:"completion,omitempty"`
}

// CompletionClientCapabilities describes client completion capabilities.
type CompletionClientCapabilities struct {
	CompletionItem *CompletionItemCapabilities `json:"completionItem,omitempty"`
}

// CompletionItemCapabilities describes per-item completion capabilities.
type CompletionItemCapabilities struct {
	SnippetSupport bool `json:"snippetSupport,omitempty"`
}

// WorkspaceFolder represents a workspace folder.
type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// InitializeResult is the response to initialize.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerCapabilities describes what the server supports.
type ServerCapabilities struct {
	TextDocumentSync   int  `json:"textDocumentSync,omitempty"` // 1 = Full, 2 = Incremental
	CompletionProvider *CompletionOptions `json:"completionProvider,omitempty"`
	HoverProvider      bool               `json:"hoverProvider,omitempty"`
	CodeActionProvider bool               `json:"codeActionProvider,omitempty"`
}

// CompletionOptions configures completion capabilities.
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// ServerInfo describes the LSP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// TextDocumentIdentifier identifies a text document.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentItem is a text document with content.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// VersionedTextDocumentIdentifier is a versioned document reference.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// Position represents a 0-based line/character position.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a start/end position pair.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location is a URI + Range.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// DidOpenTextDocumentParams is the parameter for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams is the parameter for textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent represents a full-text content change.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// DidCloseTextDocumentParams is the parameter for textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// TextDocumentPositionParams identifies a document and position.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionParams is the parameter for textDocument/completion.
type CompletionParams struct {
	TextDocumentPositionParams
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionContext provides additional info about how completion was triggered.
type CompletionContext struct {
	TriggerKind      int    `json:"triggerKind"`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`
}

// CompletionItem is a single completion candidate.
type CompletionItem struct {
	Label         string `json:"label"`
	Kind          int    `json:"kind,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	InsertText    string `json:"insertText,omitempty"`
	SortText      string `json:"sortText,omitempty"`
}

// CompletionItemKind values (subset used by braider).
const (
	CompletionItemKindClass     = 7
	CompletionItemKindInterface = 8
	CompletionItemKindStruct    = 22
)

// CompletionList wraps completion items with an isIncomplete flag.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// HoverParams is the parameter for textDocument/hover.
type HoverParams struct {
	TextDocumentPositionParams
}

// HoverResult is the response to textDocument/hover.
type HoverResult struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// MarkupContent is a string with a kind (plaintext or markdown).
type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// CodeActionParams is the parameter for textDocument/codeAction.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext       `json:"context"`
}

// CodeActionContext describes the code action context.
type CodeActionContext struct {
	Diagnostics []any    `json:"diagnostics"`
	Only        []string `json:"only,omitempty"`
}

// CodeAction is a single code action suggestion.
type CodeAction struct {
	Title   string       `json:"title"`
	Kind    string       `json:"kind,omitempty"`
	Edit    *WorkspaceEdit `json:"edit,omitempty"`
	Command *Command     `json:"command,omitempty"`
}

// WorkspaceEdit represents a set of text edits across files.
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

// TextEdit is a single text replacement.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// Command represents an LSP command.
type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}
