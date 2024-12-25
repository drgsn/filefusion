package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
)

func TestCPPHandlerBasics(t *testing.T) {
	handler := &CPPHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment", "multiline_comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"preproc_include", "using_declaration"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "///" {
		t.Errorf("Expected '///', got %s", prefix)
	}
}

func TestCPPLoggingCalls(t *testing.T) {
	handler := &CPPHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(cpp.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "cout logging",
			input:    "cout << \"Log message\" << endl;",
			expected: true,
		},
		{
			name:     "cerr logging",
			input:    "cerr << \"Error message\" << endl;",
			expected: true,
		},
		{
			name:     "clog logging",
			input:    "clog << \"Debug info\" << endl;",
			expected: true,
		},
		{
			name:     "printf logging",
			input:    "printf(\"Debug: %s\\n\", message);",
			expected: true,
		},
		{
			name:     "fprintf logging",
			input:    "fprintf(stderr, \"Error: %s\\n\", error);",
			expected: true,
		},
		{
			name:     "custom log call",
			input:    "log(\"Message\");",
			expected: true,
		},
		{
			name:     "regular function call",
			input:    "process(\"data\");",
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
			cursor := sitter.NewTreeCursor(node)
			defer cursor.Close()

			// Find the call expression node
			var findCall func(*sitter.Node)
			findCall = func(n *sitter.Node) {
				if n.Type() == "call_expression" || n.Type() == "binary_expression" {
					callNode = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					findCall(n.NamedChild(i))
				}
			}
			findCall(node)

			if callNode == nil {
				t.Fatal("No call/binary expression node found")
			}

			result := handler.IsLoggingCall(callNode, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsLoggingCall() = %v for input %q", tt.expected, tt.input)
			}
		})
	}
}

func TestCPPGetterSetter(t *testing.T) {
	handler := &CPPHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(cpp.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "getter method",
			input: `string getName() const {
                return name;
            }`,
			expected: true,
		},
		{
			name: "setter method",
			input: `void setName(string value) {
                name = value;
            }`,
			expected: true,
		},
		{
			name: "is getter",
			input: `bool isValid() const {
                return valid;
            }`,
			expected: true,
		},
		{
			name: "regular method",
			input: `void process() {
                doWork();
            }`,
			expected: false,
		},
		{
			name: "complex getter",
			input: `int getValue(int index) {
                return values[index];
            }`,
			expected: false, // Has parameter, not a simple getter
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
			cursor := sitter.NewTreeCursor(node)
			defer cursor.Close()

			// Find the function definition node
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
