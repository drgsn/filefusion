package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/css"
)

func TestCSSHandlerBasics(t *testing.T) {
	handler := &CSSHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"import_statement", "@import"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "/*" {
		t.Errorf("Expected '/*', got %s", prefix)
	}
}

func TestCSSLoggingCalls(t *testing.T) {
	handler := &CSSHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(css.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "regular rule",
			input:    ".class { color: red; }",
			expected: false,
		},
		{
			name:     "import statement",
			input:    "@import 'styles.css';",
			expected: false,
		},
		{
			name:     "media query",
			input:    "@media screen { body { color: blue; } }",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			if tree == nil {
				t.Fatal("Failed to parse input")
			}
			defer tree.Close()

			node := tree.RootNode()
			result := handler.IsLoggingCall(node, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsLoggingCall() = %v for input %q", tt.expected, tt.input)
			}
		})
	}

	// Test nil node
	if handler.IsLoggingCall(nil, []byte("")) {
		t.Error("Expected IsLoggingCall to return false for nil node")
	}
}

func TestCSSGetterSetter(t *testing.T) {
	handler := &CSSHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(css.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "regular rule",
			input:    ".class { color: red; }",
			expected: false,
		},
		{
			name:     "pseudo class",
			input:    ".class:hover { color: blue; }",
			expected: false,
		},
		{
			name:     "variable declaration",
			input:    ":root { --main-color: blue; }",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			if tree == nil {
				t.Fatal("Failed to parse input")
			}
			defer tree.Close()

			node := tree.RootNode()
			result := handler.IsGetterSetter(node, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsGetterSetter() = %v for input %q", tt.expected, tt.input)
			}
		})
	}

	// Test nil node
	if handler.IsGetterSetter(nil, []byte("")) {
		t.Error("Expected IsGetterSetter to return false for nil node")
	}
}
