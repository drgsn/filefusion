package handlers

// TypeScriptHandler extends JavaScript handler functionality
type TypeScriptHandler struct {
	JavaScriptHandler
}

func (h *TypeScriptHandler) GetImportTypes() []string {
	baseTypes := h.JavaScriptHandler.GetImportTypes()
	return append(baseTypes, "import_require_clause", "import_alias")
}
