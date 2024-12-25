package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// RubyHandler handles Ruby language specifics
type RubyHandler struct {
	BaseHandler
}

func (h *RubyHandler) GetCommentTypes() []string {
	return []string{"comment"}
}

func (h *RubyHandler) GetImportTypes() []string {
	return []string{"require", "include", "require_relative"}
}

func (h *RubyHandler) GetDocCommentPrefix() string {
	return "#"
}

func (h *RubyHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node == nil || len(content) == 0 {
		return false
	}

	nodeType := node.Type()
	if nodeType != "call" && nodeType != "method_call" && nodeType != "command" {
		return false
	}

	callText := content[node.StartByte():node.EndByte()]
	return bytes.HasPrefix(callText, []byte("puts ")) ||
		bytes.HasPrefix(callText, []byte("print ")) ||
		bytes.HasPrefix(callText, []byte("p ")) ||
		bytes.Contains(callText, []byte("logger."))
}

func (h *RubyHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node == nil || len(content) == 0 {
		return false
	}

	nodeType := node.Type()
	if nodeType != "call" && nodeType != "method" && nodeType != "method_definition" {
		return false
	}

	methodText := string(content[node.StartByte():node.EndByte()])
	return strings.Contains(methodText, "attr_reader") ||
		strings.Contains(methodText, "attr_writer") ||
		strings.Contains(methodText, "attr_accessor") ||
		strings.HasPrefix(methodText, "def get_") ||
		strings.HasPrefix(methodText, "def set_")
}
