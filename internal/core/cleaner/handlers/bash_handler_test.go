package handlers

import (
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
)

func TestBashHandlerBasics(t *testing.T) {
	handler := &BashHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	if len(commentTypes) != 1 || commentTypes[0] != "comment" {
		t.Errorf("Expected ['comment'], got %v", commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	if len(importTypes) != 2 || importTypes[0] != "source_command" || importTypes[1] != "command" {
		t.Errorf("Expected ['source_command', 'command'], got %v", importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "#" {
		t.Errorf("Expected '#', got %s", prefix)
	}
}

func TestBashLoggingCalls(t *testing.T) {
	handler := &BashHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(bash.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "stderr redirection",
			input:    "echo 'Error' >&2",
			expected: true,
		},
		{
			name:     "logger command",
			input:    "logger 'System startup complete'",
			expected: true,
		},
		{
			name:     "debug echo",
			input:    "echo \"Debug: process started\"",
			expected: true,
		},
		{
			name:     "debug printf",
			input:    "printf \"Debug: %s\\n\" \"$var\"",
			expected: true,
		},
		{
			name:     "regular echo",
			input:    "echo 'Hello world'",
			expected: false,
		},
		{
			name:     "regular command",
			input:    "ls -l",
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
			if node == nil {
				t.Fatal("Failed to get root node")
			}

			// Print all node types and their content
			var printNode func(*sitter.Node, int)
			printNode = func(n *sitter.Node, depth int) {
				if n == nil {
					return
				}
				indent := strings.Repeat("  ", depth)
				content := ""
				if n.StartByte() < uint32(len(tt.input)) && n.EndByte() <= uint32(len(tt.input)) {
					content = string([]byte(tt.input)[n.StartByte():n.EndByte()])
				}
				t.Logf("%sNode type: %s, Content: %q", indent, n.Type(), content)
				for i := 0; i < int(n.ChildCount()); i++ {
					printNode(n.Child(i), depth+1)
				}
			}
			printNode(node, 0)

			// Find the first command or redirected_statement node
			var cmdNode *sitter.Node
			var findCommand func(*sitter.Node)
			findCommand = func(n *sitter.Node) {
				if n.Type() == "command" || n.Type() == "redirected_statement" {
					cmdNode = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					findCommand(n.NamedChild(i))
				}
			}
			findCommand(node)

			if cmdNode == nil {
				t.Fatal("No command node found")
			}

			result := handler.IsLoggingCall(cmdNode, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsLoggingCall() = %v for input %q", tt.expected, tt.input)
			}
		})
	}
}

func TestBashGetterSetter(t *testing.T) {
	handler := &BashHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(bash.GetLanguage())

	// Test with a function definition
	input := `function get_value() { echo "$value"; }`
	tree := parser.Parse(nil, []byte(input))
	defer tree.Close()

	node := tree.RootNode()
	// Bash doesn't have traditional getters/setters
	if handler.IsGetterSetter(node, []byte(input)) {
		t.Error("Expected IsGetterSetter to always return false for Bash")
	}
}
