package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

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
	if node == nil || (node.Type() != "call_expression" && node.Type() != "binary_expression") {
		return false
	}
	if node.StartByte() >= uint32(len(content)) || node.EndByte() > uint32(len(content)) {
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
	if node == nil || (node.Type() != "function_definition" && node.Type() != "function_declarator") {
		return false
	}
	if node.StartByte() >= uint32(len(content)) || node.EndByte() > uint32(len(content)) {
		return false
	}
	funcText := string(content[node.StartByte():node.EndByte()])

	isGetter := (strings.Contains(strings.ToLower(funcText), "get") && !strings.Contains(strings.ToLower(funcText), "getvalue")) ||
		(strings.Contains(strings.ToLower(funcText), "is") && !strings.Contains(strings.ToLower(funcText), "isvalue"))
	isSetter := strings.Contains(funcText, "set") ||
		strings.Contains(funcText, "Set")

	if isGetter {
		return !strings.Contains(funcText, "void") &&
			strings.Count(funcText, ",") == 0
	}
	if isSetter {
		return strings.Contains(funcText, "void") &&
			strings.Count(funcText, ",") <= 1
	}
	return false
}
