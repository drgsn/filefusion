package handlers

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type PHPHandler struct {
	BaseHandler
}

func (h *PHPHandler) GetCommentTypes() []string {
	return []string{"comment", "doc_comment"}
}

func (h *PHPHandler) GetImportTypes() []string {
	return []string{"namespace_use_declaration", "require", "require_once", "include", "include_once"}
}

func (h *PHPHandler) GetDocCommentPrefix() string {
	return "/**"
}

func findNodeOfType(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == nodeType {
		return node
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if found := findNodeOfType(node.NamedChild(i), nodeType); found != nil {
			return found
		}
	}
	return nil
}

func (h *PHPHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}

	// Get the function name from the node
	var funcName string
	nodeType := node.Type()

	if nodeType == "function_call_expression" {
		nameNode := node.ChildByFieldName("function")
		if nameNode != nil && nameNode.StartByte() < uint32(len(content)) && nameNode.EndByte() <= uint32(len(content)) {
			funcName = string(content[nameNode.StartByte():nameNode.EndByte()])
		}
	} else if nodeType == "echo_statement" {
		return false
	}

	if funcName == "" {
		return false
	}

	// Check for logging function names
	return funcName == "error_log" ||
		funcName == "print_r" ||
		funcName == "var_dump"
}

func (h *PHPHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}

	methodNode := findNodeOfType(node, "method_declaration")
	if methodNode == nil {
		return false
	}

	if methodNode.StartByte() >= uint32(len(content)) || methodNode.EndByte() > uint32(len(content)) {
		return false
	}

	methodText := string(content[methodNode.StartByte():methodNode.EndByte()])
	return strings.Contains(methodText, "public function get") ||
		strings.Contains(methodText, "public function set")
}
