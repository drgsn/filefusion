package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// GoHandler handles Go language specifics
type GoHandler struct {
	BaseHandler
}

func (h *GoHandler) GetCommentTypes() []string {
	return []string{"comment"}
}

func (h *GoHandler) GetImportTypes() []string {
	return []string{"import_declaration", "import_spec"}
}

func (h *GoHandler) GetDocCommentPrefix() string {
	return "///"
}

func (h *GoHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node.Type() != "call_expression" {
		return false
	}
	callText := content[node.StartByte():node.EndByte()]
	return bytes.Contains(callText, []byte("log.")) ||
		bytes.Contains(callText, []byte("logger.")) ||
		bytes.Contains(callText, []byte("Debug(")) ||
		bytes.Contains(callText, []byte(".Print"))
}

func (h *GoHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node.Type() != "function_declaration" {
		return false
	}

	// Get function name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}

	name := string(content[nameNode.StartByte():nameNode.EndByte()])
	return strings.HasPrefix(strings.ToLower(name), "get") ||
		strings.HasPrefix(strings.ToLower(name), "set")
}
