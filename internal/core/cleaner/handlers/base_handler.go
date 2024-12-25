package handlers

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// LanguageHandler defines the interface for language-specific processing
type LanguageHandler interface {
	GetCommentTypes() []string
	GetImportTypes() []string
	GetDocCommentPrefix() string
	IsLoggingCall(node *sitter.Node, content []byte) bool
	IsGetterSetter(node *sitter.Node, content []byte) bool
}

// BaseHandler provides common functionality for all language handlers
type BaseHandler struct{}

// IsMethodNamed checks if a node represents a method/function with the given prefix
func (h *BaseHandler) IsMethodNamed(node *sitter.Node, content []byte, prefix string) bool {
	if node.Type() != "method_declaration" &&
		node.Type() != "function_declaration" &&
		node.Type() != "getter_declaration" &&
		node.Type() != "method_definition" {
		return false
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}

	name := string(content[nameNode.StartByte():nameNode.EndByte()])
	return strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix))
}
