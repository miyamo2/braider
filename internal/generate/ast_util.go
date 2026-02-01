package generate

import (
	"go/ast"
)

// IsDependencyReferenced checks if the "dependency" variable is referenced in the main function.
// It excludes blank identifier assignments (_ = dependency).
func IsDependencyReferenced(mainFunc *ast.FuncDecl) bool {
	if mainFunc == nil || mainFunc.Body == nil {
		return false
	}

	// Build parent map for context-aware checking
	parentMap := buildParentMap(mainFunc.Body)

	referenced := false
	ast.Inspect(mainFunc.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok || ident.Name != "dependency" {
			return true
		}

		// Check if this is a blank identifier assignment
		if isBlankAssignment(ident, parentMap) {
			return true // Skip this occurrence
		}

		referenced = true
		return false // Stop traversal once found
	})

	return referenced
}

// buildParentMap builds a map from child node to parent node.
//
// Algorithm: Stack-based single-pass traversal using ast.Inspect.
// The ast.Inspect function calls the visitor twice for each node:
//  1. With the node (n != nil) when entering
//  2. With nil when leaving the node's subtree
//
// Implementation:
//   - Maintain a stack of ancestor nodes
//   - When entering a node (n != nil):
//   - Record the top of the stack as this node's parent
//   - Push the current node onto the stack
//   - When leaving a node (n == nil):
//   - Pop from the stack
//
// Edge cases:
//   - Root node has no parent (not added to parentMap)
//   - Empty stack when n == nil is handled safely
//
// Time complexity: O(n) where n is the number of AST nodes
// Space complexity: O(h) for the stack, where h is the tree height
func buildParentMap(root ast.Node) map[ast.Node]ast.Node {
	parentMap := make(map[ast.Node]ast.Node)
	var stack []ast.Node

	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			// Pop from stack when leaving a node
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			return false
		}

		// Record parent for current node
		if len(stack) > 0 {
			parentMap[n] = stack[len(stack)-1]
		}

		// Push current node onto stack
		stack = append(stack, n)
		return true
	})

	return parentMap
}

// HasBlankDependencyAssignment checks if "_ = dependency" already exists in the main function.
// Returns true if found, meaning we should NOT add another one.
func HasBlankDependencyAssignment(mainFunc *ast.FuncDecl) bool {
	if mainFunc == nil || mainFunc.Body == nil {
		return false
	}

	parentMap := buildParentMap(mainFunc.Body)

	found := false
	ast.Inspect(mainFunc.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok || ident.Name != "dependency" {
			return true
		}

		if isBlankAssignment(ident, parentMap) {
			found = true
			return false // Stop traversal
		}

		return true
	})

	return found
}

// isBlankAssignment checks if the identifier is part of "_ = dependency" pattern.
func isBlankAssignment(ident *ast.Ident, parentMap map[ast.Node]ast.Node) bool {
	parent := parentMap[ident]

	// Check if parent is an assignment statement
	assignStmt, ok := parent.(*ast.AssignStmt)
	if !ok {
		return false
	}

	// Check if LHS is blank identifier
	if len(assignStmt.Lhs) == 1 {
		if lhsIdent, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
			if lhsIdent.Name == "_" {
				// Check if RHS is the dependency identifier
				if len(assignStmt.Rhs) == 1 && assignStmt.Rhs[0] == ident {
					return true
				}
			}
		}
	}

	return false
}
