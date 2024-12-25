package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

func TestBaseHandler(t *testing.T) {
	handler := &BaseHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	tests := []struct {
		name     string
		input    string
		prefix   string
		expected bool
	}{
		{
			name:     "get method",
			input:    "package main\nfunc GetName() string { return name }",
			prefix:   "Get",
			expected: true,
		},
		{
			name:     "set method",
			input:    "package main\nfunc SetName(name string) { this.name = name }",
			prefix:   "Set",
			expected: true,
		},
		{
			name:     "non-matching method",
			input:    "package main\nfunc Process() error { return nil }",
			prefix:   "Get",
			expected: false,
		},
		{
			name:     "case insensitive match",
			input:    "package main\nfunc getName() string { return name }",
			prefix:   "Get",
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

			// Find the function declaration node
			var funcNode *sitter.Node
			cursor := sitter.NewTreeCursor(tree.RootNode())
			defer cursor.Close()

			ok := cursor.GoToFirstChild()
			for ok {
				if cursor.CurrentNode().Type() == "function_declaration" {
					funcNode = cursor.CurrentNode()
					break
				}
				ok = cursor.GoToNextSibling()
			}

			if funcNode == nil {
				t.Fatal("No function declaration found")
			}

			if got := handler.IsMethodNamed(funcNode, []byte(tt.input), tt.prefix); got != tt.expected {
				t.Errorf("IsMethodNamed() = %v, want %v", got, tt.expected)
			}
		})
	}
}
