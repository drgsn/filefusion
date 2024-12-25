package handlers

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type BashHandler struct {
	BaseHandler
}

func (h *BashHandler) GetCommentTypes() []string {
	return []string{"comment"}
}

func (h *BashHandler) GetImportTypes() []string {
	return []string{"source_command", "command"}
}

func (h *BashHandler) GetDocCommentPrefix() string {
	return "#"
}

func (h *BashHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}

	nodeType := node.Type()
	if nodeType == "redirected_statement" {
		// If it's a redirected statement, it's a logging call
		return true
	}

	if nodeType != "command" {
		return false
	}

	// Get the full command text for other checks
	if node.StartByte() >= uint32(len(content)) || node.EndByte() > uint32(len(content)) {
		return false
	}
	cmdText := string(content[node.StartByte():node.EndByte()])

	// Check for logging commands
	return strings.Contains(cmdText, "logger") ||
		strings.Contains(cmdText, "echo \"Debug") ||
		strings.Contains(cmdText, "printf \"Debug")
}

func (h *BashHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	return false
}
