package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
)

func findFirstNodeOfType(root *sitter.Node, nodeType string) *sitter.Node {
	if root == nil {
		return nil
	}
	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	ok := cursor.GoToFirstChild()
	for ok {
		if cursor.CurrentNode().Type() == nodeType {
			return cursor.CurrentNode()
		}
		if node := findFirstNodeOfType(cursor.CurrentNode(), nodeType); node != nil {
			return node
		}
		ok = cursor.GoToNextSibling()
	}
	return nil
}

func TestCSharpHandlerBasics(t *testing.T) {
	handler := &CSharpHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment", "multiline_comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"using_directive"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "///" {
		t.Errorf("Expected '///', got %s", prefix)
	}
}

func TestCSharpLoggingCalls(t *testing.T) {
	handler := &CSharpHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(csharp.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "console write",
			input:    "Console.WriteLine(\"test\");",
			expected: true,
		},
		{
			name:     "debug log",
			input:    "Debug.Log(\"test\");",
			expected: true,
		},
		{
			name:     "logger info",
			input:    "Logger.Info(\"test\");",
			expected: true,
		},
		{
			name:     "trace write",
			input:    "Trace.WriteLine(\"test\");",
			expected: true,
		},
		{
			name:     "regular method call",
			input:    "Process(\"test\");",
			expected: false,
		},
		{
			name:     "string method",
			input:    "\"test\".ToString();",
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
			invocationNode := findFirstNodeOfType(tree.RootNode(), "invocation_expression")
			if invocationNode == nil {
				t.Fatal("Failed to find invocation_expression node")
			}

			result := handler.IsLoggingCall(invocationNode, []byte(tt.input))
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

func TestCSharpGetterSetter(t *testing.T) {
	handler := &CSharpHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(csharp.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"auto_property", "public string Name { get; set; }", true},
		{"getter_only_property", "public string Name { get; }", true},
		{"setter_only_property", "public string Name { private set; }", true},
		{"property_with_access_modifiers", "public string Name { private set; get; }", true},
		{"getter_method", "public string GetName() { return name; }", true},
		{"setter_method", "public void SetName(string value) { name = value; }", true},
		{"regular_method", "public void Process() { }", false},
		{"regular_property", "public string Name;", false},
		{"invalid_property_syntax", "public string Name { get set; }", false},
		{"empty_accessor_blocks", "public string Name { get; }", true},
		{"invalid_accessor_body", "public string Name { get {} set; }", true},
		{"missing_getter", "public string Name { set; }", true},
		{"missing_setter", "public string Name { get; }", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			if tree == nil {
				t.Fatal("Failed to parse input")
			}
			defer tree.Close()

			rootNode := tree.RootNode()
			if rootNode == nil {
				t.Fatal("Failed to get root node")
			}

			node := findRelevantNode(rootNode, []byte(tt.input), []string{"property_declaration", "method_declaration"})
			if node == nil || node.Type() == "ERROR" {
				t.Skip("Skipping unsupported syntax")
			}

			result := handler.IsGetterSetter(node, []byte(tt.input))

			if result != tt.expected {
				t.Errorf("Expected IsGetterSetter() = %v for input %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

func findRelevantNode(node *sitter.Node, content []byte, types []string) *sitter.Node {
	if node == nil {
		return nil
	}

	for _, nodeType := range types {
		if node.Type() == nodeType || node.Type() == "ERROR" {
			return node
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if found := findRelevantNode(child, content, types); found != nil {
			return found
		}
	}
	return nil
}
