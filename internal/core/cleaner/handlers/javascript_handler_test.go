package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
)

func TestJavaScriptHandlerBasics(t *testing.T) {
	handler := &JavaScriptHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment", "multiline_comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"import_statement", "import_specifier"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "/**" {
		t.Errorf("Expected '/**', got %s", prefix)
	}
}

func TestJavaScriptHandlerLoggingAndGetterSetter(t *testing.T) {
	handler := &JavaScriptHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(javascript.GetLanguage())

	tests := []struct {
		name      string
		input     string
		isLogging bool
		isGetter  bool
	}{
		{
			name:      "console log",
			input:     "console.log('test');",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "logger debug",
			input:     "logger.debug('test');",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "getter method",
			input:     "class Test { get name() { return this._name; } }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "setter method",
			input:     "class Test { set name(value) { this._name = value; } }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "regular method",
			input:     "function test() { return true; }",
			isLogging: false,
			isGetter:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			if tree == nil {
				t.Fatal("Failed to parse input")
			}
			defer tree.Close()

			root := tree.RootNode()
			if root == nil {
				t.Fatal("Failed to get root node")
			}

			var checkNode func(*sitter.Node)
			checkNode = func(n *sitter.Node) {
				if n == nil {
					return
				}

				nodeType := n.Type()
				if nodeType == "call_expression" ||
					nodeType == "method_definition" ||
					nodeType == "getter_declaration" ||
					nodeType == "setter_declaration" {
					if got := handler.IsLoggingCall(n, []byte(tt.input)); got != tt.isLogging {
						t.Errorf("IsLoggingCall() = %v, want %v for %s", got, tt.isLogging, nodeType)
					}
					if got := handler.IsGetterSetter(n, []byte(tt.input)); got != tt.isGetter {
						t.Errorf("IsGetterSetter() = %v, want %v for %s", got, tt.isGetter, nodeType)
					}
				}

				for i := 0; i < int(n.ChildCount()); i++ {
					checkNode(n.Child(i))
				}
			}

			checkNode(root)
		})
	}
}
