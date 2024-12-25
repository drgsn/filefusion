package handlers

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// CSSHandler handles CSS language specifics
type CSSHandler struct {
	BaseHandler
}

func (h *CSSHandler) GetCommentTypes() []string {
	return []string{"comment"}
}

func (h *CSSHandler) GetImportTypes() []string {
	// @import rules are CSS imports
	return []string{"import_statement", "@import"}
}

func (h *CSSHandler) GetDocCommentPrefix() string {
	return "/*"
}

func (h *CSSHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	// CSS doesn't have logging calls
	return false
}

func (h *CSSHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	// CSS doesn't have getters/setters
	return false
}
