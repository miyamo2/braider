package generate

import "go/token"

// IsGoKeyword checks if a name is a Go keyword.
func IsGoKeyword(name string) bool {
	return token.Lookup(name).IsKeyword()
}

// IsGoBuiltin checks if a name is a Go builtin type, function, or constant.
func IsGoBuiltin(name string) bool {
	return goBuiltins[name]
}

// IsKeywordOrBuiltin checks if a name conflicts with Go keywords or built-in identifiers.
func IsKeywordOrBuiltin(name string) bool {
	return IsGoKeyword(name) || IsGoBuiltin(name)
}

// goBuiltins is the set of Go builtin types, functions, and constants.
var goBuiltins = map[string]bool{
	// Builtin types
	"bool":       true,
	"byte":       true,
	"complex64":  true,
	"complex128": true,
	"float32":    true,
	"float64":    true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"rune":       true,
	"string":     true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uintptr":    true,

	// Builtin functions
	"append":  true,
	"cap":     true,
	"close":   true,
	"complex": true,
	"copy":    true,
	"delete":  true,
	"imag":    true,
	"len":     true,
	"make":    true,
	"new":     true,
	"panic":   true,
	"print":   true,
	"println": true,
	"real":    true,
	"recover": true,

	// Builtin constants
	"true":  true,
	"false": true,
	"iota":  true,
	"nil":   true,

	// Special builtin
	"error": true,

	// Go 1.18+ additions
	"any":        true, // Alias for interface{}
	"comparable": true, // Constraint for types that support == and !=

	// Go 1.21+ additions
	"clear": true, // Builtin function to clear maps and slices
	"min":   true, // Builtin function for minimum value
	"max":   true, // Builtin function for maximum value
}
