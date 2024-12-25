package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// SwiftHandler handles Swift language specifics
type SwiftHandler struct {
	BaseHandler
}

func (h *SwiftHandler) GetCommentTypes() []string {
	return []string{"comment", "multiline_comment", "documentation_comment"}
}

func (h *SwiftHandler) GetImportTypes() []string {
	return []string{"import_declaration"}
}

func (h *SwiftHandler) GetDocCommentPrefix() string {
	return "///"
}

func (h *SwiftHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node.Type() != "call_expression" {
		return false
	}
	callText := content[node.StartByte():node.EndByte()]
	return bytes.HasPrefix(callText, []byte("print(")) ||
		bytes.HasPrefix(callText, []byte("debugPrint(")) ||
		bytes.HasPrefix(callText, []byte("NSLog(")) ||
		bytes.Contains(callText, []byte("logger."))
}

func (h *SwiftHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	nodeType := node.Type()

	// For variable declarations or getter/setter specifiers
	if nodeType == "variable_declaration" || nodeType == "getter_specifier" || nodeType == "setter_specifier" {
		// Look for getter and setter declarations in the node's children
		var findAccessor func(*sitter.Node) bool
		findAccessor = func(n *sitter.Node) bool {
			if n == nil {
				return false
			}

			// Check if this node is a getter or setter
			nType := n.Type()
			if nType == "getter_specifier" || nType == "setter_specifier" {
				return true
			}

			// Check children
			for i := 0; i < int(n.NamedChildCount()); i++ {
				if findAccessor(n.NamedChild(i)) {
					return true
				}
			}
			return false
		}

		return findAccessor(node)
	}

	// For function declarations, check if they look like getters/setters
	if nodeType == "function_declaration" {
		funcText := string(content[node.StartByte():node.EndByte()])
		funcName := ""

		// Find the function name
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "simple_identifier" {
				funcName = string(content[child.StartByte():child.EndByte()])
				break
			}
		}

		// Check if it's a getter/setter
		return (strings.HasPrefix(funcName, "get") && strings.Contains(funcText, "return")) ||
			(strings.HasPrefix(funcName, "set") && strings.Contains(funcText, "="))
	}

	return false
}
