package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/kotlin"
)

func TestKotlinHandlerBasics(t *testing.T) {
	handler := &KotlinHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment", "multiline_comment", "kdoc"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"import_header"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "/**" {
		t.Errorf("Expected '/**', got %s", prefix)
	}
}

func TestKotlinLoggingCalls(t *testing.T) {
	handler := &KotlinHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(kotlin.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "println call",
			input:    "println(\"Debug message\")",
			expected: true,
		},
		{
			name:     "print call",
			input:    "print(\"Debug info\")",
			expected: true,
		},
		{
			name:     "Logger call",
			input:    "Logger.debug(\"Debug info\")",
			expected: true,
		},
		{
			name:     "logger instance call",
			input:    "logger.info(\"message\")",
			expected: true,
		},
		{
			name:     "log call",
			input:    "log.info(\"Info message\")",
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
		// Edge cases
		{
			name:     "nested logging call",
			input:    "logger.info(getData().toString())",
			expected: true,
		},
		{
			name:     "logging with string template",
			input:    "println(\"Value: ${value}\")",
			expected: true,
		},
		{
			name:     "logging with multiple arguments",
			input:    "logger.info(\"Message: {}\", value)",
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
	malformed := "println(\"unclosed string"
	tree = parser.Parse(nil, []byte(malformed))
	if tree == nil {
		t.Fatal("Failed to parse malformed input")
	}
	defer tree.Close()
	if handler.IsLoggingCall(tree.RootNode(), []byte(malformed)) {
		t.Error("Expected IsLoggingCall to return false for malformed input")
	}
}

func TestKotlinGetterSetter(t *testing.T) {
	handler := &KotlinHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(kotlin.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "property with getter",
			input: `var name: String
				get() = field`,
			expected: true,
		},
		{
			name: "property with getter and setter",
			input: `var name: String
				get() = field
				set(value) { field = value }`,
			expected: true,
		},
		{
			name: "getter function",
			input: `fun getName(): String {
				return name
			}`,
			expected: true,
		},
		{
			name: "setter function",
			input: `fun setName(value: String) {
				this.name = value
			}`,
			expected: true,
		},
		{
			name: "regular method",
			input: `fun process() {
				doWork()
			}`,
			expected: false,
		},
		// Edge cases
		{
			name: "property with annotations",
			input: `@JsonProperty
				var name: String
				get() = field`,
			expected: true,
		},
		{
			name: "property with complex type",
			input: `var items: List<Map<String, Int>>
				get() = field`,
			expected: true,
		},
		{
			name: "nested property",
			input: `class Outer {
				inner class Inner {
					var name: String
					get() = field
				}
			}`,
			expected: true,
		},
		{
			name: "property with custom getter logic",
			input: `var fullName: String
				get() = "$firstName $lastName"`,
			expected: true,
		},
		{
			name: "property with visibility modifiers",
			input: `private var _name: String = ""
				public var name: String
					get() = _name
					set(value) { _name = value }`,
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

				if n.Type() == "property_declaration" || n.Type() == "function_declaration" {
					targetNode = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					findNode(n.NamedChild(i))
				}
			}
			findNode(node)

			if targetNode == nil {
				t.Fatal("No property or function declaration node found")
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
	if handler.IsGetterSetter(tree.RootNode(), []byte("")) {
		t.Error("Expected IsGetterSetter to return false for empty content")
	}

	// Test malformed input
	malformed := "var name: String get("
	tree = parser.Parse(nil, []byte(malformed))
	if tree == nil {
		t.Fatal("Failed to parse malformed input")
	}
	defer tree.Close()
	if handler.IsGetterSetter(tree.RootNode(), []byte(malformed)) {
		t.Error("Expected IsGetterSetter to return false for malformed input")
	}
}

func TestKotlinFindFirstChild(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(kotlin.GetLanguage())

	tests := []struct {
		name         string
		input        string
		searchType   string
		shouldFind   bool
		expectedText string
	}{
		{
			name: "find simple identifier",
			input: `fun test() {
				val name = "test"
			}`,
			searchType:   "simple_identifier",
			shouldFind:   true,
			expectedText: "test",
		},
		{
			name: "find in nested structure",
			input: `class Test {
				fun method() {
					val x = 1
				}
			}`,
			searchType:   "property_declaration",
			shouldFind:   true,
			expectedText: "val x = 1",
		},
		{
			name:       "non-existent type",
			input:      "val x = 1",
			searchType: "non_existent_type",
			shouldFind: false,
		},
		{
			name:       "empty input",
			input:      "",
			searchType: "simple_identifier",
			shouldFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.input))
			if tree == nil {
				t.Fatal("Failed to parse input")
			}
			defer tree.Close()

			found := findFirstChild(tree.RootNode(), tt.searchType)

			if tt.shouldFind {
				if found == nil {
					t.Errorf("Expected to find node of type %s, but found nil", tt.searchType)
				} else if string([]byte(tt.input)[found.StartByte():found.EndByte()]) != tt.expectedText {
					t.Errorf("Expected node text %q, got %q", tt.expectedText, string([]byte(tt.input)[found.StartByte():found.EndByte()]))
				}
			} else if found != nil {
				t.Errorf("Expected not to find node of type %s, but found one", tt.searchType)
			}
		})
	}

	// Test nil node
	if found := findFirstChild(nil, "any_type"); found != nil {
		t.Error("Expected findFirstChild to return nil for nil node")
	}
}
