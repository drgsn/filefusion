package handlers

import (
    sitter "github.com/smacker/go-tree-sitter"
)

// SQLHandler handles SQL language specifics
type SQLHandler struct {
    BaseHandler
}

func (h *SQLHandler) GetCommentTypes() []string {
    return []string{"comment", "block_comment"}
}

func (h *SQLHandler) GetImportTypes() []string {
    // SQL doesn't have traditional imports, but we can consider these statements
    return []string{"create_extension_statement", "use_statement"}
}

func (h *SQLHandler) GetDocCommentPrefix() string {
    return "--"
}

func (h *SQLHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
    // SQL doesn't have traditional logging calls
    return false
}

func (h *SQLHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
    // SQL doesn't have traditional getters/setters
    return false
}