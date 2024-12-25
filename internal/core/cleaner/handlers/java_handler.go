package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// JavaHandler handles Java language specifics
type JavaHandler struct {
	BaseHandler
}

func (h *JavaHandler) GetCommentTypes() []string {
	return []string{"line_comment", "block_comment", "javadoc_comment"}
}

func (h *JavaHandler) GetImportTypes() []string {
	return []string{"import_declaration"}
}

func (h *JavaHandler) GetDocCommentPrefix() string {
	return "/**"
}

func (h *JavaHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	// For Java, we need to check both method invocations and the qualifier
	nodeType := node.Type()
	if nodeType != "method_invocation" {
		return false
	}

	// Get the method identifier
	callText := content[node.StartByte():node.EndByte()]

	// Check if it's a logging call
	loggingPatterns := []string{
		"Logger",
		"System.out",
		"System.err",
		"log.",
		"logger.",
	}

	for _, pattern := range loggingPatterns {
		if bytes.Contains(callText, []byte(pattern)) {
			return true
		}
	}

	return false
}

func (h *JavaHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node.Type() != "method_declaration" {
		return false
	}

	methodText := string(content[node.StartByte():node.EndByte()])

	// Get the method name
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}
	name := string(content[nameNode.StartByte():nameNode.EndByte()])

	// Check for getter pattern
	isGetter := strings.HasPrefix(strings.ToLower(name), "get") &&
		strings.Contains(methodText, "return")

	// Check for setter pattern
	isSetter := strings.HasPrefix(strings.ToLower(name), "set") &&
		strings.Contains(methodText, "void")

	return isGetter || isSetter
}
