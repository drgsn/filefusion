package handlers

import (
	"bytes"

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
	// Swift uses computed properties instead of explicit getters/setters
	if node.Type() != "computed_property" {
		return false
	}
	propText := content[node.StartByte():node.EndByte()]
	return bytes.Contains(propText, []byte("get")) || bytes.Contains(propText, []byte("set"))
}
