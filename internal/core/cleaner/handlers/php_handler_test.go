package handlers

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/php"
)

func TestPHPHandlerBasics(t *testing.T) {
	handler := &PHPHandler{}

	// Test comment types
	commentTypes := handler.GetCommentTypes()
	expected := []string{"comment", "doc_comment"}
	if !stringSliceEqual(commentTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, commentTypes)
	}

	// Test import types
	importTypes := handler.GetImportTypes()
	expected = []string{"namespace_use_declaration", "require", "require_once", "include", "include_once"}
	if !stringSliceEqual(importTypes, expected) {
		t.Errorf("Expected %v, got %v", expected, importTypes)
	}

	// Test doc comment prefix
	if prefix := handler.GetDocCommentPrefix(); prefix != "/**" {
		t.Errorf("Expected '/**', got %s", prefix)
	}
}

func TestPHPLoggingCalls(t *testing.T) {
	handler := &PHPHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(php.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "error_log call",
			input:    "<?php error_log('Error message'); ?>",
			expected: true,
		},
		{
			name:     "print_r call",
			input:    "<?php print_r($data); ?>",
			expected: true,
		},
		{
			name:     "var_dump call",
			input:    "<?php var_dump($variable); ?>",
			expected: true,
		},
		{
			name:     "regular echo",
			input:    "<?php echo 'Hello'; ?>",
			expected: false,
		},
		{
			name:     "regular function call",
			input:    "<?php process($data); ?>",
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
				if n == nil {
					return
				}

				// Debug node types
				t.Logf("Node type: %s", n.Type())

				nodeType := n.Type()
				if nodeType == "function_call_expression" {
					callNode = n
					return
				}

				if nodeType == "echo_statement" {
					callNode = n
					return
				}

				// Try both named and unnamed children
				for i := 0; i < int(n.ChildCount()); i++ {
					findCall(n.Child(i))
				}
			}
			findCall(node)

			if callNode == nil {
				t.Fatalf("No function call node found in input: %s\nAST structure: %s", tt.input, node.String())
			}

			result := handler.IsLoggingCall(callNode, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsLoggingCall() = %v for input %q", tt.expected, tt.input)
			}
		})
	}
}

func TestPHPGetterSetter(t *testing.T) {
	handler := &PHPHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(php.GetLanguage())

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "getter method",
			input: `<?php
                class Test {
                    public function getName() {
                        return $this->name;
                    }
                }
            ?>`,
			expected: true,
		},
		{
			name: "setter method",
			input: `<?php
                class Test {
                    public function setName($value) {
                        $this->name = $value;
                    }
                }
            ?>`,
			expected: true,
		},
		{
			name: "regular method",
			input: `<?php
                class Test {
                    public function process() {
                        return $this->doWork();
                    }
                }
            ?>`,
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
			var methodNode *sitter.Node

			var findMethod func(*sitter.Node)
			findMethod = func(n *sitter.Node) {
				if n == nil {
					return
				}
				if n.Type() == "method_declaration" {
					methodNode = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					findMethod(n.NamedChild(i))
				}
			}
			findMethod(node)

			if methodNode == nil {
				t.Fatal("No method declaration node found")
			}

			result := handler.IsGetterSetter(methodNode, []byte(tt.input))
			if result != tt.expected {
				t.Errorf("Expected IsGetterSetter() = %v for input %q", tt.expected, tt.input)
			}
		})
	}
}
