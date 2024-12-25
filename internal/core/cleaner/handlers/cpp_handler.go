package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CPPHandler handles C/C++ language specifics
type CPPHandler struct {
	BaseHandler
}

func (h *CPPHandler) GetCommentTypes() []string {
	return []string{"comment", "multiline_comment"}
}

func (h *CPPHandler) GetImportTypes() []string {
	return []string{"preproc_include", "using_declaration"}
}

func (h *CPPHandler) GetDocCommentPrefix() string {
	return "///"
}

func (h *CPPHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node.Type() != "call_expression" && node.Type() != "binary_expression" {
		return false
	}
	callText := content[node.StartByte():node.EndByte()]
	return bytes.Contains(callText, []byte("cout")) ||
		bytes.Contains(callText, []byte("cerr")) ||
		bytes.Contains(callText, []byte("clog")) ||
		bytes.Contains(callText, []byte("printf")) ||
		bytes.Contains(callText, []byte("fprintf")) ||
		bytes.Contains(callText, []byte("log"))
}

func (h *CPPHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node.Type() != "function_definition" && node.Type() != "function_declarator" {
		return false
	}

	// Check for function name patterns
	funcText := string(content[node.StartByte():node.EndByte()])

	// Common getter/setter patterns in C++
	isGetter := strings.Contains(funcText, "get") ||
		strings.Contains(funcText, "Get") ||
		strings.Contains(funcText, "is") ||
		strings.Contains(funcText, "Is")

	isSetter := strings.Contains(funcText, "set") ||
		strings.Contains(funcText, "Set")

	// Check function body or prototype for typical getter/setter patterns
	if isGetter {
		// Getters typically return a value and have no parameters
		return !strings.Contains(funcText, "void") &&
			strings.Count(funcText, ",") == 0
	}

	if isSetter {
		// Setters typically return void and have exactly one parameter
		return strings.Contains(funcText, "void") &&
			strings.Count(funcText, ",") == 0
	}

	return false
}
