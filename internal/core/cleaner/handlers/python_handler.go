package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type PythonHandler struct {
	BaseHandler
}

func (h *PythonHandler) GetCommentTypes() []string {
	return []string{"comment"}
}

func (h *PythonHandler) GetImportTypes() []string {
	return []string{"import_statement", "import_from_statement"}
}

func (h *PythonHandler) GetDocCommentPrefix() string {
	return "\"\"\""
}

func (h *PythonHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node == nil || node.Type() != "call" {
		return false
	}
	if node.StartByte() >= uint32(len(content)) || node.EndByte() > uint32(len(content)) {
		return false
	}
	callText := content[node.StartByte():node.EndByte()]
	return bytes.HasPrefix(callText, []byte("print(")) ||
		bytes.HasPrefix(callText, []byte("logging.")) ||
		bytes.HasPrefix(callText, []byte("logger."))
}

func (h *PythonHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node == nil || node.Type() != "function_definition" {
		return false
	}
	if node.StartByte() >= uint32(len(content)) || node.EndByte() > uint32(len(content)) {
		return false
	}

	// Check for property decorator
	parent := node.Parent()
	if parent != nil {
		parentText := string(content[parent.StartByte():parent.EndByte()])
		if strings.Contains(parentText, "@property") {
			return true
		}
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	name := string(content[nameNode.StartByte():nameNode.EndByte()])
	return strings.HasPrefix(name, "get_") ||
		strings.HasPrefix(name, "set_")
}
