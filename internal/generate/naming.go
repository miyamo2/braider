package generate

import (
	"strings"
	"unicode"
)

// DeriveFieldName converts a type name to lowerCamelCase for use as a field name.
// It handles all-caps abbreviations and resolves conflicts with Go keywords/builtins.
// Examples:
//   - "Service" -> "service"
//   - "DBConnection" -> "dbConnection"
//   - "HTTPClient" -> "httpClient"
//   - "Type" -> "type_" (keyword conflict)
func DeriveFieldName(typeName string) string {
	// Extract local name from fully qualified type name
	parts := strings.Split(typeName, ".")
	localName := parts[len(parts)-1]

	if localName == "" {
		return "field"
	}

	// Convert to lowerCamelCase (reuse existing utility)
	fieldName := ToLowerCamelCase(localName)

	// Resolve keyword conflicts
	if IsKeywordOrBuiltin(fieldName) {
		fieldName = fieldName + "_"
	}

	return fieldName
}

// ToLowerCamelCase converts a name to lowerCamelCase.
//
// Purpose: Generate variable/field names from type names while preserving
// readability of common abbreviations (HTTP, DB, ID, etc.).
//
// Algorithm:
//  1. Count consecutive leading uppercase letters (index i)
//  2. Apply conversion rules based on i:
//     - i == 0: Already lowerCamelCase, return unchanged
//     - i == len(runes): All uppercase (e.g., "DB"), lowercase all
//     - i == 1: Single uppercase (e.g., "Service"), lowercase first char only
//     - i > 1: Multiple uppercase (e.g., "DBConnection"), lowercase all but last
//
// Examples:
//   - "HTTPClient" -> "httpClient" (i=4, lowercase first 3)
//   - "DBConnection" -> "dbConnection" (i=2, lowercase first 1)
//   - "UserID" -> "userID" (i=1, then i=2 after "user", lowercase "U", keep "ID")
//   - "Service" -> "service" (i=1, lowercase first char)
//   - "DB" -> "db" (i=2==len, lowercase all)
//
// Edge cases:
//   - Empty string returns empty string
//   - Already lowercase returns unchanged
//   - Unicode runes are handled correctly
//
// Time complexity: O(n) where n is the length of the string
func ToLowerCamelCase(name string) string {
	if name == "" {
		return ""
	}

	runes := []rune(name)

	// Find the index where we should stop lowercasing
	// i represents the position after the last consecutive uppercase letter
	// For "DB" -> i=2 (lowercase all)
	// For "DBConnection" -> i=2 (lowercase "D", keep "B" uppercase before "Connection")
	// For "UserID" -> i=1 (lowercase "U", then later handle "ID")
	i := 0
	for i < len(runes) && unicode.IsUpper(runes[i]) {
		i++
	}

	if i == 0 {
		// Already starts with lowercase
		return name
	}

	if i == len(runes) {
		// All uppercase: convert all to lowercase (e.g., "DB" -> "db")
		return strings.ToLower(name)
	}

	if i == 1 {
		// Single leading uppercase: just lowercase first char (e.g., "Service" -> "service")
		runes[0] = unicode.ToLower(runes[0])
		return string(runes)
	}

	// Multiple leading uppercase: lowercase all but the last
	// This preserves the readability of abbreviations before the next word
	// "DBConnection" -> "dbConnection" (lowercase "D", keep "B" for "BConnection")
	// "HTTPClient" -> "httpClient" (lowercase "HTT", keep "P" for "PClient")
	for j := 0; j < i-1; j++ {
		runes[j] = unicode.ToLower(runes[j])
	}
	return string(runes)
}

// ToUpperCamelCase ensures that the first character of name is uppercase.
// Examples:
//   - "service" -> "Service"
//   - "Service" -> "Service" (unchanged)
//   - "dbHandler" -> "DbHandler"
//   - "" -> ""
func ToUpperCamelCase(name string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
