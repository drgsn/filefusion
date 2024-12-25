package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/html"
)

func TestHTMLHandlerBasics(t *testing.T) {
	handler := &HTMLHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"link_element", "script_element"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "<!--" {
		t.Errorf("Expected '<!--', got %s", prefix)
	}
}

func TestHTMLLoggingCalls(t *testing.T) {
	handler := &HTMLHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(html.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "regular element",
			input:    "<div>Test</div>",
			expected: false,
		},
		{
			name:     "script tag with console.log",
			input:    "<script>console.log('test');</script>",
			expected: false, // HTML handler doesn't handle script content
		},
		{
			name:     "empty element",
			input:    "<br/>",
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

	// Test empty content
	tree := parser.Parse(nil, []byte(""))
	if tree == nil {
		t.Fatal("Failed to parse empty input")
	}
	defer tree.Close()
	if handler.IsLoggingCall(tree.RootNode(), []byte("")) {
		t.Error("Expected IsLoggingCall to return false for empty content")
	}
}

func TestHTMLGetterSetter(t *testing.T) {
	handler := &HTMLHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(html.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "regular element",
			input:    "<div>Test</div>",
			expected: false,
		},
		{
			name:     "input element",
			input:    "<input type=\"text\" value=\"test\">",
			expected: false,
		},
		{
			name:     "script tag with getter",
			input:    "<script>get value() { return this._value; }</script>",
			expected: false, // HTML handler doesn't handle script content
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

	// Test empty content
	tree := parser.Parse(nil, []byte(""))
	if tree == nil {
		t.Fatal("Failed to parse empty input")
	}
	defer tree.Close()
	if handler.IsGetterSetter(tree.RootNode(), []byte("")) {
		t.Error("Expected IsGetterSetter to return false for empty content")
	}
}
