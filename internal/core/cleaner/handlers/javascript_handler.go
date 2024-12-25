package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// JavaScriptHandler handles JavaScript language specifics
type JavaScriptHandler struct {
	BaseHandler
}

func (h *JavaScriptHandler) GetCommentTypes() []string {
	return []string{"comment", "multiline_comment"}
}

func (h *JavaScriptHandler) GetImportTypes() []string {
	return []string{"import_statement", "import_specifier"}
}

func (h *JavaScriptHandler) GetDocCommentPrefix() string {
	return "/**"
}

func (h *JavaScriptHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node.Type() != "call_expression" {
		return false
	}
	callText := content[node.StartByte():node.EndByte()]
	return bytes.HasPrefix(callText, []byte("console.")) ||
		bytes.HasPrefix(callText, []byte("logger."))
}

func (h *JavaScriptHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	nodeType := node.Type()
	if nodeType != "method_definition" &&
		nodeType != "getter_declaration" &&
		nodeType != "setter_declaration" {
		return false
	}

	methodText := string(content[node.StartByte():node.EndByte()])
	return strings.HasPrefix(methodText, "get ") ||
		strings.HasPrefix(methodText, "set ") ||
		nodeType == "getter_declaration" ||
		nodeType == "setter_declaration"
}
