package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CSharpHandler handles C# language specifics
type CSharpHandler struct {
	BaseHandler
}

func (h *CSharpHandler) GetCommentTypes() []string {
	return []string{"comment", "multiline_comment"}
}

func (h *CSharpHandler) GetImportTypes() []string {
	return []string{"using_directive"}
}

func (h *CSharpHandler) GetDocCommentPrefix() string {
	return "///"
}

func (h *CSharpHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node.Type() != "invocation_expression" {
		return false
	}
	callText := content[node.StartByte():node.EndByte()]
	return bytes.Contains(callText, []byte("Console.")) ||
		bytes.Contains(callText, []byte("Debug.")) ||
		bytes.Contains(callText, []byte("Logger.")) ||
		bytes.Contains(callText, []byte("Trace."))
}

func (h *CSharpHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node.Type() != "property_declaration" && node.Type() != "method_declaration" {
		return false
	}
	methodText := string(content[node.StartByte():node.EndByte()])

	// Check for auto-implemented properties
	if strings.Contains(methodText, "{ get;") || strings.Contains(methodText, "{ set;") {
		return true
	}

	// Check for traditional getter/setter methods
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		name := string(content[nameNode.StartByte():nameNode.EndByte()])
		return strings.HasPrefix(name, "Get") || strings.HasPrefix(name, "Set")
	}

	return false
}
