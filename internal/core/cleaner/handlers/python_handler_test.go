package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

func TestPythonHandlerBasics(t *testing.T) {
	handler := &PythonHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	if len(commentTypes) != 1 || commentTypes[0] != "comment" {
		t.Errorf("Expected ['comment'], got %v", commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected := []string{"import_statement", "import_from_statement"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != `"""` {
		t.Errorf("Expected '\"\"\"', got %s", prefix)
	}
}

func TestPythonHandlerLogging(t *testing.T) {
	handler := &PythonHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "simple print call",
			input:    "print('Debug message')",
			expected: true,
		},
		{
			name:     "print with formatting",
			input:    "print(f'Value is {value}')",
			expected: true,
		},
		{
			name:     "logging module info",
			input:    "logging.info('Info message')",
			expected: true,
		},
		{
			name:     "logging module debug",
			input:    "logging.debug('Debug info')",
			expected: true,
		},
		{
			name:     "logging module error",
			input:    "logging.error('Error occurred')",
			expected: true,
		},
		{
			name:     "logger instance debug",
			input:    "logger.debug('Debug info')",
			expected: true,
		},
		{
			name:     "logger instance info",
			input:    "logger.info('Info message')",
			expected: true,
		},
		{
			name:     "regular function call",
			input:    "process_data('input')",
			expected: false,
		},
		{
			name:     "method call",
			input:    "obj.process()",
			expected: false,
		},
		{
			name:     "print-like method",
			input:    "printer.execute('message')",
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
				if n.Type() == "call" {
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

func TestPythonHandlerGetterSetter(t *testing.T) {
	handler := &PythonHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "property getter",
			input: `@property
def name(self):
    return self._name`,
			expected: true,
		},
		{
			name: "property with decorator and docstring",
			input: `@property
def name(self):
    """Get the name value."""
    return self._name`,
			expected: true,
		},
		{
			name: "traditional getter",
			input: `def get_name(self):
    return self._name`,
			expected: true,
		},
		{
			name: "traditional setter",
			input: `def set_name(self, value):
    self._name = value`,
			expected: true,
		},
		{
			name: "get method with validation",
			input: `def get_value(self):
    if self._value is None:
        raise ValueError("Value not set")
    return self._value`,
			expected: true,
		},
		{
			name: "set method with validation",
			input: `def set_value(self, value):
    if value < 0:
        raise ValueError("Value must be positive")
    self._value = value`,
			expected: true,
		},
		{
			name: "regular method",
			input: `def process_data(self):
    return self.data.process()`,
			expected: false,
		},
		{
			name: "get method with args",
			input: `def get_item(self, index):
    return self._items[index]`,
			expected: true,
		},
		{
			name: "method with get prefix",
			input: `def getting_started():
    print("Tutorial")`,
			expected: false,
		},
		{
			name: "non-property decorated method",
			input: `@staticmethod
def get_version():
    return "1.0.0"`,
			expected: true,
		},
		{
			name: "unrelated decorator method",
			input: `@deprecated
def get_legacy_value():
    return old_value`,
			expected: true,
		},
		{
			name: "complex getter",
			input: `def get_calculated_value(self):
    return sum(self._values) / len(self._values)`,
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
			var funcNode *sitter.Node
			var findFunc func(*sitter.Node)
			findFunc = func(n *sitter.Node) {
				if n.Type() == "function_definition" {
					funcNode = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					findFunc(n.NamedChild(i))
				}
			}
			findFunc(node)

			if funcNode == nil {
				t.Fatal("No function definition node found")
			}

			result := handler.IsGetterSetter(funcNode, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsGetterSetter() = %v for input %q", tt.expected, tt.input)
			}
		})
	}
}

func TestPythonHandlerEdgeCases(t *testing.T) {
	handler := &PythonHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())

	// Test invalid node
	if handler.IsLoggingCall(nil, []byte("")) {
		t.Error("Expected IsLoggingCall to return false for nil node")
	}
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
	if handler.IsLoggingCall(node, []byte("")) {
		t.Error("Expected IsLoggingCall to return false for empty content")
	}
	if handler.IsGetterSetter(node, []byte("")) {
		t.Error("Expected IsGetterSetter to return false for empty content")
	}

	// Test malformed input
	malformed := "def get_value(self:"
	tree = parser.Parse(nil, []byte(malformed))
	if tree == nil {
		t.Fatal("Failed to parse malformed input")
	}
	defer tree.Close()
	node = tree.RootNode()
	if handler.IsGetterSetter(node, []byte(malformed)) {
		t.Error("Expected IsGetterSetter to return false for malformed input")
	}
}
