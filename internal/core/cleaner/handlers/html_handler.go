package handlers

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// HTMLHandler handles HTML language specifics
type HTMLHandler struct {
	BaseHandler
}

func (h *HTMLHandler) GetCommentTypes() []string {
	return []string{"comment"}
}

func (h *HTMLHandler) GetImportTypes() []string {
	// Consider <link> and <script> tags as imports
	return []string{"link_element", "script_element"}
}

func (h *HTMLHandler) GetDocCommentPrefix() string {
	return "<!--"
}

func (h *HTMLHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	// HTML doesn't have traditional logging calls
	return false
}

func (h *HTMLHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	// HTML doesn't have getters/setters
	return false
}
