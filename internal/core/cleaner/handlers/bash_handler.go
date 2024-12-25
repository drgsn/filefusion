package handlers

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// BashHandler handles Bash script specifics
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
	if node.Type() != "command" {
		return false
	}
	cmdText := string(content[node.StartByte():node.EndByte()])
	return strings.Contains(cmdText, "logger") ||
		strings.Contains(cmdText, ">&2") ||
		strings.Contains(cmdText, "echo \"Debug") ||
		strings.Contains(cmdText, "printf \"Debug")
}

func (h *BashHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	// Bash doesn't have traditional getters/setters
	return false
}
