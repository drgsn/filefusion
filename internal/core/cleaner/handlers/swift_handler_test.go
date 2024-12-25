package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/swift"
)

func TestSwiftHandlerBasics(t *testing.T) {
	handler := &SwiftHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment", "multiline_comment", "documentation_comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"import_declaration"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "///" {
		t.Errorf("Expected '///', got %s", prefix)
	}
}

func TestSwiftLoggingCalls(t *testing.T) {
	handler := &SwiftHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(swift.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "print call",
			input:    "print(\"Debug message\")",
			expected: true,
		},
		{
			name:     "debugPrint call",
			input:    "debugPrint(\"Debug info\")",
			expected: true,
		},
		{
			name:     "NSLog call",
			input:    "NSLog(\"Log message\")",
			expected: true,
		},
		{
			name:     "logger call",
			input:    "logger.debug(\"Debug info\")",
			expected: true,
		},
		{
			name:     "regular function call",
			input:    "process(\"data\")",
			expected: false,
		},
		{
			name:     "method call",
			input:    "obj.process()",
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
				if n.Type() == "call_expression" {
					callNode = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					findCall(n.NamedChild(i))
				}
			}
			findCall(node)

			if callNode == nil {
				t.Fatal("No call node found")
			}

			result := handler.IsLoggingCall(callNode, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsLoggingCall() = %v for input %q", tt.expected, tt.input)
			}
		})
	}
}

func TestSwiftGetterSetter(t *testing.T) {
	handler := &SwiftHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(swift.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "computed property getter",
			input: `var name: String {
				get { return _name }
			}`,
			expected: true,
		},
		{
			name: "computed property getter and setter",
			input: `var name: String {
				get { return _name }
				set { _name = newValue }
			}`,
			expected: true,
		},
		{
			name: "regular method",
			input: `func process() {
				doWork()
			}`,
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
			var propNode *sitter.Node
			var findProp func(*sitter.Node)
			findProp = func(n *sitter.Node) {
				if n == nil {
					return
				}

				if n.Type() == "variable_declaration" || n.Type() == "function_declaration" || n.Type() == "getter_specifier" || n.Type() == "setter_specifier" {
					propNode = n
					return
				}

				// Check children first
				for i := 0; i < int(n.NamedChildCount()); i++ {
					child := n.NamedChild(i)
					findProp(child)
					if propNode != nil {
						return
					}
				}
			}
			findProp(node)

			if propNode == nil {
				t.Fatal("No property or function declaration node found")
			}

			result := handler.IsGetterSetter(propNode, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsGetterSetter() = %v for input %q", tt.expected, tt.input)
			}
		})
	}
}
