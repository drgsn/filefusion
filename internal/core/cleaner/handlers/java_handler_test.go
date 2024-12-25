package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func TestJavaHandlerBasics(t *testing.T) {
	handler := &JavaHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"line_comment", "block_comment", "javadoc_comment"}
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
	if prefix := handler.GetDocCommentPrefix(); prefix != "/**" {
		t.Errorf("Expected '/**', got %s", prefix)
	}
}

func TestJavaHandlerLoggingAndGetterSetter(t *testing.T) {
	handler := &JavaHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())

	tests := []struct {
		name      string
		input     string
		isLogging bool
		isGetter  bool
	}{
		{
			name:      "logger field",
			input:     "class Test { void test() { logger.info(\"test\"); } }",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "system out",
			input:     "class Test { void test() { System.out.println(\"test\"); } }",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "system err",
			input:     "class Test { void test() { System.err.println(\"error\"); } }",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "log call",
			input:     "class Test { void test() { log.debug(\"debug\"); } }",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "Logger static call",
			input:     "class Test { void test() { Logger.getLogger(Test.class).info(\"test\"); } }",
			isLogging: true,
			isGetter:  false,
		},
		{
			name:      "regular method call",
			input:     "class Test { void test() { process(\"data\"); } }",
			isLogging: false,
			isGetter:  false,
		},
		{
			name:      "getter method",
			input:     "class Test { public String getName() { return name; } }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "setter method",
			input:     "class Test { public void setName(String name) { this.name = name; } }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "boolean getter with is prefix",
			input:     "class Test { public boolean isValid() { return valid; } }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "getter with javadoc",
			input:     "class Test { /** Gets the name */ public String getName() { return name; } }",
			isLogging: false,
			isGetter:  true,
		},
		{
			name:      "regular method",
			input:     "class Test { public void process() { doWork(); } }",
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

			// Navigate to find the relevant nodes
			var cursor = sitter.NewTreeCursor(tree.RootNode())
			defer cursor.Close()

			var processNode func(*sitter.Node)
			processNode = func(node *sitter.Node) {
				if node == nil {
					return
				}

				nodeType := node.Type()
				if nodeType == "method_invocation" {
					if got := handler.IsLoggingCall(node, []byte(tt.input)); got != tt.isLogging {
						t.Errorf("IsLoggingCall() = %v, want %v for %s", got, tt.isLogging, nodeType)
					}
				} else if nodeType == "method_declaration" {
					if got := handler.IsGetterSetter(node, []byte(tt.input)); got != tt.isGetter {
						t.Errorf("IsGetterSetter() = %v, want %v for %s", got, tt.isGetter, nodeType)
					}
				}

				// Process children
				for i := 0; i < int(node.NamedChildCount()); i++ {
					child := node.NamedChild(i)
					processNode(child)
				}
			}

			processNode(tree.RootNode())
		})
	}
}
