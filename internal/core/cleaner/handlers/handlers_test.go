package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
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

func TestPythonHandler(t *testing.T) {
	handler := &PythonHandler{}

	// Test comment types
	if len(handler.GetCommentTypes()) == 0 {
		t.Error("Expected non-empty comment types for Python")
	}

	// Test import types
	if len(handler.GetImportTypes()) == 0 {
		t.Error("Expected non-empty import types for Python")
	}

	// Test doc comment prefix
	if handler.GetDocCommentPrefix() != "\"\"\"" {
		t.Errorf("Expected doc comment prefix \"\"\", got %s", handler.GetDocCommentPrefix())
	}
}

func TestJavaHandler(t *testing.T) {
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
			name:      "getter method",
			input:     "class Test { public String getName() { return name; } }",
			isLogging: false,
			isGetter:  true,
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

func TestJavaScriptHandler(t *testing.T) {
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

func TestGoHandler(t *testing.T) {
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

func TestTypeScriptHandler(t *testing.T) {
	handler := &TypeScriptHandler{}

	// Verify that TypeScriptHandler extends JavaScriptHandler
	importTypes := handler.GetImportTypes()
	baseImportTypes := handler.JavaScriptHandler.GetImportTypes()

	// All base import types should be included
	for _, baseType := range baseImportTypes {
		found := false
		for _, importType := range importTypes {
			if importType == baseType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TypeScript handler missing base import type: %s", baseType)
		}
	}

	// Should have additional TypeScript specific import types
	if len(importTypes) <= len(baseImportTypes) {
		t.Error("TypeScript handler should have additional import types")
	}

	extraTypes := []string{"import_require_clause", "import_alias"}
	for _, extraType := range extraTypes {
		found := false
		for _, importType := range importTypes {
			if importType == extraType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("TypeScript handler missing import type: %s", extraType)
		}
	}
}
