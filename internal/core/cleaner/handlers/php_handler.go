package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// PHPHandler handles PHP language specifics
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

func (h *PHPHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node.Type() != "function_call" {
		return false
	}
	callText := content[node.StartByte():node.EndByte()]
	return bytes.HasPrefix(callText, []byte("error_log(")) ||
		bytes.HasPrefix(callText, []byte("print_r(")) ||
		bytes.HasPrefix(callText, []byte("var_dump("))
}

func (h *PHPHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node.Type() != "method_declaration" {
		return false
	}
	methodText := string(content[node.StartByte():node.EndByte()])
	return strings.Contains(methodText, "public function get") ||
		strings.Contains(methodText, "public function set")
}
