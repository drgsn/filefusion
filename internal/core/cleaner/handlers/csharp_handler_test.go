package handlers

import (
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/stretchr/testify/assert"
)

func getCSharpLanguage() *sitter.Language {
	return csharp.GetLanguage()
}

// Helper function to find node by type in the AST
func findNodeByType(root *sitter.Node, nodeType string) *sitter.Node {
	if root == nil {
		return nil
	}

	if root.Type() == nodeType {
		return root
	}

	cursor := sitter.NewTreeCursor(root)
	defer cursor.Close()

	for ok := cursor.GoToFirstChild(); ok; ok = cursor.GoToNextSibling() {
		if node := findNodeByType(cursor.CurrentNode(), nodeType); node != nil {
			return node
		}
	}

	return nil
}

// Helper function to debug print node structure
func debugPrintNode(t *testing.T, node *sitter.Node, content []byte, level int) {
	if node == nil {
		return
	}

	indent := strings.Repeat("  ", level)
	nodeText := ""
	if content != nil {
		nodeText = string(content[node.StartByte():node.EndByte()])
	}
	t.Logf("%sType: %s, Text: %s", indent, node.Type(), nodeText)

	cursor := sitter.NewTreeCursor(node)
	defer cursor.Close()

	for ok := cursor.GoToFirstChild(); ok; ok = cursor.GoToNextSibling() {
		debugPrintNode(t, cursor.CurrentNode(), content, level+1)
	}
}

func TestCSharpHandler_GetCommentTypes(t *testing.T) {
	handler := &CSharpHandler{}
	expected := []string{"comment", "multiline_comment"}
	actual := handler.GetCommentTypes()
	assert.Equal(t, expected, actual, "Comment types should match expected values")
}

func TestCSharpHandler_GetImportTypes(t *testing.T) {
	handler := &CSharpHandler{}
	expected := []string{"using_directive"}
	actual := handler.GetImportTypes()
	assert.Equal(t, expected, actual, "Import types should match expected values")
}

func TestCSharpHandler_GetDocCommentPrefix(t *testing.T) {
	handler := &CSharpHandler{}
	expected := "///"
	actual := handler.GetDocCommentPrefix()
	assert.Equal(t, expected, actual, "Doc comment prefix should be ///")
}

func TestCSharpHandler_IsLoggingCall(t *testing.T) {
	handler := &CSharpHandler{}
	
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name: "Console.WriteLine call",
			code: `namespace TestNamespace {
				class Program {
					void Method() {
						Console.WriteLine("test");
					}
				}
			}`,
			expected: true,
		},
		{
			name: "Debug.Log call",
			code: `namespace TestNamespace {
				class Program {
					void Method() {
						Debug.Log("test");
					}
				}
			}`,
			expected: true,
		},
		{
			name: "Logger.Info call",
			code: `namespace TestNamespace {
				class Program {
					void Method() {
						Logger.Info("test");
					}
				}
			}`,
			expected: true,
		},
		{
			name: "Trace.WriteLine call",
			code: `namespace TestNamespace {
				class Program {
					void Method() {
						Trace.WriteLine("test");
					}
				}
			}`,
			expected: true,
		},
		{
			name: "Regular method call",
			code: `namespace TestNamespace {
				class Program {
					void Method() {
						MyMethod("test");
					}
				}
			}`,
			expected: false,
		},
		{
			name: "Regular property access",
			code: `namespace TestNamespace {
				class Program {
					void Method() {
						var x = myObject.Property.ToString();
					}
				}
			}`,
			expected: false,
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(getCSharpLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.code))
			invocationNode := findNodeByType(tree.RootNode(), "invocation_expression")
			
			if invocationNode == nil && tt.expected {
				t.Fatalf("Failed to find invocation_expression node in AST for test case: %s", tt.name)
			}

			if invocationNode != nil {
				result := handler.IsLoggingCall(invocationNode, []byte(tt.code))
				assert.Equal(t, tt.expected, result, "IsLoggingCall result should match expected for %s", tt.name)
			}
		})
	}
}

func TestCSharpHandler_IsGetterSetter(t *testing.T) {
	handler := &CSharpHandler{}
	
	tests := []struct {
		name     string
		code     string
		nodeType string
		expected bool
	}{
		{
			name: "Simple property with get/set",
			code: `namespace TestNamespace {
				class TestClass {
					public int MyProperty { get; set; }
				}
			}`,
			nodeType: "property_declaration",
			expected: true,
		},
		{
			name: "Property with only getter",
			code: `namespace TestNamespace {
				class TestClass {
					public int MyProperty { get; }
				}
			}`,
			nodeType: "property_declaration",
			expected: true,
		},
		{
			name: "Property with only setter",
			code: `namespace TestNamespace {
				class TestClass {
					public int MyProperty { set { value = value; } }
				}
			}`,
			nodeType: "property_declaration",
			expected: true,
		},
		{
			name: "Getter method",
			code: `namespace TestNamespace {
				class TestClass {
					public int GetValue() { return value; }
				}
			}`,
			nodeType: "method_declaration",
			expected: true,
		},
		{
			name: "Setter method",
			code: `namespace TestNamespace {
				class TestClass {
					public void SetValue(int value) { this.value = value; }
				}
			}`,
			nodeType: "method_declaration",
			expected: true,
		},
		{
			name: "Regular method",
			code: `namespace TestNamespace {
				class TestClass {
					public void DoSomething() { }
				}
			}`,
			nodeType: "method_declaration",
			expected: false,
		},
		{
			name: "Property without accessors",
			code: `namespace TestNamespace {
				class TestClass {
					public int MyProperty => 42;
				}
			}`,
			nodeType: "property_declaration",
			expected: false,
		},
	}

	parser := sitter.NewParser()
	parser.SetLanguage(getCSharpLanguage())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := parser.Parse(nil, []byte(tt.code))
			targetNode := findNodeByType(tree.RootNode(), tt.nodeType)
			
			if targetNode == nil {
				t.Fatalf("Failed to find %s node in AST for test case: %s", tt.nodeType, tt.name)
			}
			
			// Debug print the node structure if needed
			if testing.Verbose() {
				t.Log("Node structure for:", tt.name)
				debugPrintNode(t, targetNode, []byte(tt.code), 0)
			}
			
			result := handler.IsGetterSetter(targetNode, []byte(tt.code))
			assert.Equal(t, tt.expected, result, "IsGetterSetter result should match expected for %s", tt.name)
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(getCSharpLanguage())

	t.Run("hasAccessor", func(t *testing.T) {
		code := `namespace TestNamespace {
			class TestClass {
				public int MyProperty { get; set; }
			}
		}`
		tree := parser.Parse(nil, []byte(code))
		propertyNode := findNodeByType(tree.RootNode(), "property_declaration")
		
		if propertyNode == nil {
			t.Fatal("Failed to find property_declaration node")
		}

		accessorList := propertyNode.ChildByFieldName("accessors")
		if accessorList == nil {
			t.Fatal("Failed to find accessor_list node")
		}

		if testing.Verbose() {
			t.Log("Node structure for hasAccessor test:")
			debugPrintNode(t, accessorList, []byte(code), 0)
		}

		result := hasAccessor(accessorList, []byte(code))
		assert.True(t, result, "hasAccessor should return true for property with get/set")
	})

	t.Run("findNextSibling", func(t *testing.T) {
		code := `namespace TestNamespace {
			class MyClass {
				public void Method1() {}
				public void Method2() {}
			}
		}`
		tree := parser.Parse(nil, []byte(code))
		method1 := findNodeByType(tree.RootNode(), "method_declaration")
		
		if method1 == nil {
			t.Fatal("Failed to find first method declaration")
		}
		
		nextMethod := findNextSibling(method1, "method_declaration")
		assert.NotNil(t, nextMethod, "findNextSibling should find Method2")
		assert.Equal(t, "method_declaration", nextMethod.Type(), "Next sibling should be a method declaration")
	})
}

func TestCSharpHandler_EdgeCases(t *testing.T) {
	handler := &CSharpHandler{}
	parser := sitter.NewParser()
	parser.SetLanguage(getCSharpLanguage())
	
	t.Run("IsLoggingCall with nil node", func(t *testing.T) {
		result := handler.IsLoggingCall(nil, []byte(""))
		assert.False(t, result, "IsLoggingCall should return false for nil node")
	})
	
	t.Run("IsGetterSetter with nil node", func(t *testing.T) {
		result := handler.IsGetterSetter(nil, []byte(""))
		assert.False(t, result, "IsGetterSetter should return false for nil node")
	})
	
	t.Run("IsLoggingCall with empty content", func(t *testing.T) {
		code := `namespace TestNamespace {
			class TestClass {
				void Method() {}
			}
		}`
		tree := parser.Parse(nil, []byte(code))
		methodNode := findNodeByType(tree.RootNode(), "method_declaration")
		
		if methodNode == nil {
			t.Fatal("Failed to find method declaration")
		}
		
		result := handler.IsLoggingCall(methodNode, []byte(code))
		assert.False(t, result, "IsLoggingCall should return false for empty method")
	})
	
	t.Run("IsGetterSetter with empty content", func(t *testing.T) {
		code := `namespace TestNamespace {
			class TestClass {
				public int EmptyProperty { }
			}
		}`
		tree := parser.Parse(nil, []byte(code))
		propertyNode := findNodeByType(tree.RootNode(), "property_declaration")
		
		if propertyNode == nil {
			t.Fatal("Failed to find property declaration")
		}
		
		result := handler.IsGetterSetter(propertyNode, []byte(code))
		assert.False(t, result, "IsGetterSetter should return false for empty property")
	})

	t.Run("hasAccessor with nil node", func(t *testing.T) {
		result := hasAccessor(nil, []byte(""))
		assert.False(t, result, "hasAccessor should return false for nil node")
	})
}
		