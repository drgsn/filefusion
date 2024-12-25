package handlers

import (
	"bytes"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// CSharpHandler handles C# language specifics
type CSharpHandler struct {
	BaseHandler
}

func (h *CSharpHandler) GetCommentTypes() []string {
	return []string{"comment", "multiline_comment"}
}

func (h *CSharpHandler) GetImportTypes() []string {
	return []string{"using_directive"}
}

func (h *CSharpHandler) GetDocCommentPrefix() string {
	return "///"
}

func (h *CSharpHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node == nil {
		fmt.Println("IsLoggingCall: Node is nil")
		return false
	}
	if node.Type() != "invocation_expression" {
		fmt.Printf("IsLoggingCall: Node type is not invocation_expression, got %s\n", node.Type())
		return false
	}

	memberAccess := node.Child(0)
	if memberAccess == nil {
		fmt.Println("IsLoggingCall: Member access is nil")
		return false
	}

	if memberAccess.Type() != "member_access_expression" {
		fmt.Printf("IsLoggingCall: Member access type is not member_access_expression, got %s\n", memberAccess.Type())
		return false
	}

	callText := content[memberAccess.StartByte():memberAccess.EndByte()]
	fmt.Printf("IsLoggingCall: Call text is %s\n", callText)

	return bytes.Contains(callText, []byte("Console.")) ||
		bytes.Contains(callText, []byte("Debug.")) ||
		bytes.Contains(callText, []byte("Logger.")) ||
		bytes.Contains(callText, []byte("Trace."))
}

func (h *CSharpHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}

	switch node.Type() {
	case "property_declaration":
		accessorList := node.ChildByFieldName("accessors")
		if accessorList != nil {
			hasGetter := false
			hasSetter := false

			for i := 0; i < int(accessorList.ChildCount()); i++ {
				accessor := accessorList.Child(i)

				if accessor.Type() != "accessor_declaration" {
					// Skip unsupported accessor node types
					continue
				}

				text := string(content[accessor.StartByte():accessor.EndByte()])
				if strings.Contains(text, "get") {
					hasGetter = true
				}
				if strings.Contains(text, "set") {
					hasSetter = true
				}
			}
			return hasGetter || hasSetter
		}
		return false

	case "method_declaration":
		nameNode := node.ChildByFieldName("name")
		bodyNode := node.ChildByFieldName("body")
		if nameNode != nil && bodyNode != nil {
			name := string(content[nameNode.StartByte():nameNode.EndByte()])
			bodyText := string(content[bodyNode.StartByte():bodyNode.EndByte()])
			if strings.HasPrefix(name, "Get") && strings.Contains(bodyText, "return") {
				return true
			}
			if strings.HasPrefix(name, "Set") && strings.Contains(bodyText, "=") {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// Helper function to check for `get` or `set` in a block
func hasAccessor(block *sitter.Node, content []byte) bool {
	hasGetter := false
	hasSetter := false

	for i := 0; i < int(block.ChildCount()); i++ {
		child := block.Child(i)
		if child.Type() == "ERROR" {
			// Interpret `ERROR` nodes as `get` or `set`
			text := string(content[child.StartByte():child.EndByte()])
			if text == "get" {
				hasGetter = true
			} else if text == "set" {
				hasSetter = true
			}
		}
	}

	return hasGetter || hasSetter
}

// Helper function to find the next sibling node
func findNextSibling(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil || node.Parent() == nil {
		return nil
	}

	parent := node.Parent()
	for i := 0; i < int(parent.ChildCount()); i++ {
		child := parent.Child(i)
		if child == node {
			// Find the next sibling
			for j := i + 1; j < int(parent.ChildCount()); j++ {
				sibling := parent.Child(j)
				if sibling.Type() == nodeType {
					return sibling
				}
			}
			break
		}
	}
	return nil
}
