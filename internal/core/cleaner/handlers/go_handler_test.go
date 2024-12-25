package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

func TestGoHandlerBasics(t *testing.T) {
	handler := &GoHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"import_declaration", "import_spec"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "///" {
		t.Errorf("Expected '///', got %s", prefix)
	}
}

func TestGoHandlerLoggingAndGetterSetter(t *testing.T) {
	handler := &GoHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	tests := []struct {
		name      string
		input     string
		isLogging bool
		isGetter  bool
	}{
		{
			name:      "log print",
			input:     "package main\nfunc main() { log.Println(\"test\") }",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "logger debug",
			input:     "package main\nfunc main() { logger.Debug(\"test\") }",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "getter method",
			input:     "package main\nfunc GetName() string { return name }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "setter method",
			input:     "package main\nfunc SetName(name string) { this.name = name }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "regular function",
			input:     "package main\nfunc Process() error { return nil }",
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

			var processNode func(*sitter.Node)
			processNode = func(node *sitter.Node) {
				if node == nil {
					return
				}

				nodeType := node.Type()
				if nodeType == "call_expression" {
					if got := handler.IsLoggingCall(node, []byte(tt.input)); got != tt.isLogging {
						t.Errorf("IsLoggingCall() = %v, want %v for %s", got, tt.isLogging, nodeType)
					}
				} else if nodeType == "function_declaration" {
					if got := handler.IsGetterSetter(node, []byte(tt.input)); got != tt.isGetter {
						t.Errorf("IsGetterSetter() = %v, want %v for %s", got, tt.isGetter, nodeType)
					}
				}

				// Process children
				for i := 0; i < int(node.NamedChildCount()); i++ {
					processNode(node.NamedChild(i))
				}
			}

			processNode(tree.RootNode())
		})
	}
}
