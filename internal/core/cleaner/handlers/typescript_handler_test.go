package handlers

import "testing"

func TestTypeScriptHandler(t *testing.T) {
	handler := &TypeScriptHandler{}

	// Verify that TypeScriptHandler extends JavaScriptHandler
	importTypes := handler.GetImportTypes()
	baseImportTypes := handler.JavaScriptHandler.GetImportTypes()

	// All base import types should be included
	for _, baseType := range baseImportTypes {
		found := false
		for _, importType := range importTypes {
			if importType == baseType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TypeScript handler missing base import type: %s", baseType)
		}
	}

	// Should have additional TypeScript specific import types
	if len(importTypes) <= len(baseImportTypes) {
		t.Error("TypeScript handler should have additional import types")
	}

	extraTypes := []string{"import_require_clause", "import_alias"}
	for _, extraType := range extraTypes {
		found := false
		for _, importType := range importTypes {
			if importType == extraType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TypeScript handler missing import type: %s", extraType)
		}
	}
}
