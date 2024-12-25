package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/ruby"
)

func TestRubyHandlerBasics(t *testing.T) {
	handler := &RubyHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"require", "include", "require_relative"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "#" {
		t.Errorf("Expected '#', got %s", prefix)
	}
}

func TestRubyLoggingCalls(t *testing.T) {
	handler := &RubyHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(ruby.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "puts call",
			input:    "puts 'Debug message'",
			expected: true,
		},
		{
			name:     "print call",
			input:    "print 'Debug info'",
			expected: true,
		},
		{
			name:     "p call",
			input:    "p object",
			expected: true,
		},
		{
			name:     "logger call",
			input:    "logger.info('Log message')",
			expected: true,
		},
		{
			name:     "logger with block",
			input:    "logger.debug { 'Debug info' }",
			expected: true,
		},
		{
			name:     "regular method call",
			input:    "process_data('input')",
			expected: false,
		},
		{
			name:     "method with puts in name",
			input:    "outputs('test')",
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
			var callNode *sitter.Node
			var findCall func(*sitter.Node)
			findCall = func(n *sitter.Node) {
				if n == nil {
					return
				}

				nodeType := n.Type()
				if nodeType == "call" || nodeType == "method_call" || nodeType == "command" {
					callNode = n
					return
				}

				for i := 0; i < int(n.NamedChildCount()); i++ {
					findCall(n.NamedChild(i))
				}
			}
			findCall(node)

			if callNode == nil {
				t.Logf("AST: %s", node.String())
				t.Fatal("No call node found")
			}

			result := handler.IsLoggingCall(callNode, []byte(tt.input))
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
	malformed := "puts 'unclosed string"
	tree = parser.Parse(nil, []byte(malformed))
	if tree == nil {
		t.Fatal("Failed to parse malformed input")
	}
	defer tree.Close()
	node := tree.RootNode()
	if node == nil {
		t.Fatal("Failed to get root node")
	}
	if handler.IsLoggingCall(node, []byte(malformed)) {
		t.Error("Expected IsLoggingCall to return false for malformed input")
	}
}

func TestRubyGetterSetter(t *testing.T) {
	handler := &RubyHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(ruby.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "attr_reader",
			input:    "attr_reader :name",
			expected: true,
		},
		{
			name:     "attr_writer",
			input:    "attr_writer :name",
			expected: true,
		},
		{
			name:     "attr_accessor",
			input:    "attr_accessor :name, :age",
			expected: true,
		},
		{
			name:     "getter method",
			input:    "def get_name\n  @name\nend",
			expected: true,
		},
		{
			name:     "setter method",
			input:    "def set_name(value)\n  @name = value\nend",
			expected: true,
		},
		{
			name:     "regular method",
			input:    "def process\n  do_work\nend",
			expected: false,
		},
		{
			name:     "method with get in name",
			input:    "def forget\n  clear_memory\nend",
			expected: false,
		},
		{
			name:     "multiple attr_readers",
			input:    "attr_reader :name, :age, :email",
			expected: true,
		},
		{
			name:     "attr_accessor with symbols",
			input:    "attr_accessor :first_name, :last_name",
			expected: true,
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
			var targetNode *sitter.Node
			var findNode func(*sitter.Node)
			findNode = func(n *sitter.Node) {
				if n == nil {
					return
				}

				nodeType := n.Type()
				if nodeType == "call" || nodeType == "method" || nodeType == "method_definition" {
					targetNode = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					findNode(n.NamedChild(i))
				}
			}

			findNode(node)

			if targetNode == nil {
				t.Logf("AST: %s", node.String())
				t.Fatal("No call or method node found")
			}

			result := handler.IsGetterSetter(targetNode, []byte(tt.input))
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
	node := tree.RootNode()
	if node == nil {
		t.Fatal("Failed to get root node")
	}
	if handler.IsGetterSetter(node, []byte("")) {
		t.Error("Expected IsGetterSetter to return false for empty content")
	}

	// Test malformed input
	malformed := "def get_name"
	tree = parser.Parse(nil, []byte(malformed))
	if tree == nil {
		t.Fatal("Failed to parse malformed input")
	}
	defer tree.Close()
	node = tree.RootNode()
	if node == nil {
		t.Fatal("Failed to get root node")
	}
	if handler.IsGetterSetter(node, []byte(malformed)) {
		t.Error("Expected IsGetterSetter to return false for malformed input")
	}
}
