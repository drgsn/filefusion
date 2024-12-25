package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/sql"
)

func TestSQLHandlerBasics(t *testing.T) {
	handler := &SQLHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment", "block_comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"create_extension_statement", "use_statement"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "--" {
		t.Errorf("Expected '--', got %s", prefix)
	}
}

func TestSQLLoggingCalls(t *testing.T) {
	handler := &SQLHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(sql.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "select statement",
			input:    "SELECT * FROM users;",
			expected: false,
		},
		{
			name:     "insert statement",
			input:    "INSERT INTO logs (message) VALUES ('test');",
			expected: false,
		},
		{
			name:     "create table",
			input:    "CREATE TABLE logs (id INT, message TEXT);",
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

	// Test malformed input
	malformed := "SELECT * FROM"
	tree = parser.Parse(nil, []byte(malformed))
	if tree == nil {
		t.Fatal("Failed to parse malformed input")
	}
	defer tree.Close()
	if handler.IsLoggingCall(tree.RootNode(), []byte(malformed)) {
		t.Error("Expected IsLoggingCall to return false for malformed input")
	}
}

func TestSQLGetterSetter(t *testing.T) {
	handler := &SQLHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(sql.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "select statement",
			input:    "SELECT value FROM settings WHERE key = 'name';",
			expected: false,
		},
		{
			name:     "update statement",
			input:    "UPDATE settings SET value = 'new' WHERE key = 'name';",
			expected: false,
		},
		{
			name:     "create function",
			input:    "CREATE FUNCTION get_setting(p_key TEXT) RETURNS TEXT AS $$ SELECT value FROM settings WHERE key = p_key; $$ LANGUAGE SQL;",
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

	// Test empty content
	tree := parser.Parse(nil, []byte(""))
	if tree == nil {
		t.Fatal("Failed to parse empty input")
	}
	defer tree.Close()
	if handler.IsGetterSetter(tree.RootNode(), []byte("")) {
		t.Error("Expected IsGetterSetter to return false for empty content")
	}

	// Test malformed input
	malformed := "CREATE FUNCTION get_value"
	tree = parser.Parse(nil, []byte(malformed))
	if tree == nil {
		t.Fatal("Failed to parse malformed input")
	}
	defer tree.Close()
	if handler.IsGetterSetter(tree.RootNode(), []byte(malformed)) {
		t.Error("Expected IsGetterSetter to return false for malformed input")
	}
}
