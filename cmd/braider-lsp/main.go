package main

import (
	"fmt"
	"os"

	"github.com/miyamo2/braider/internal/lsp"
)

const usage = `braider-lsp: LSP server for braider DI annotation assistance.

Usage:
  braider-lsp

Start an LSP server (JSON-RPC 2.0 over stdio) for editor integration.

Capabilities:
  textDocument/completion   Surface exported type candidates for DI annotation type arguments.
  textDocument/hover        Show which provider/injector/variable binding wins for the type under the cursor.
  textDocument/codeAction   Offer "Register with annotation.Provide" quick-fix on exported constructors.

The server performs a best-effort go/packages load per request, using open-file
overlays for unsaved edits, and runs a background workspace scan on startup to
populate in-memory provider/injector/variable caches.
`

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		fmt.Print(usage)
		return
	}
	server := lsp.NewServer(os.Stdin, os.Stdout)
	if err := server.Run(); err != nil {
		os.Exit(1)
	}
}
