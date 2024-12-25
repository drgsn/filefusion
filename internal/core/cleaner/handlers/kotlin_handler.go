package handlers

import (
	"bytes"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type KotlinHandler struct {
	BaseHandler
}

func (h *KotlinHandler) GetCommentTypes() []string {
	return []string{"comment", "multiline_comment", "kdoc"}
}

func (h *KotlinHandler) GetImportTypes() []string {
	return []string{"import_header"}
}

func (h *KotlinHandler) GetDocCommentPrefix() string {
	return "/**"
}

func (h *KotlinHandler) IsLoggingCall(node *sitter.Node, content []byte) bool {
	if node == nil || node.Type() != "call_expression" {
		return false
	}

	if node.StartByte() >= uint32(len(content)) || node.EndByte() > uint32(len(content)) {
		return false
	}

	callText := content[node.StartByte():node.EndByte()]
	return bytes.Contains(bytes.ToLower(callText), []byte("println(")) ||
		bytes.Contains(bytes.ToLower(callText), []byte("print(")) ||
		bytes.Contains(callText, []byte("Logger.")) ||
		bytes.Contains(callText, []byte("logger.")) ||
		bytes.Contains(callText, []byte("log."))
}

// findFirstChild recursively searches for a child node of the specified type
func findFirstChild(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == nodeType {
		return node
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if found := findFirstChild(node.NamedChild(i), nodeType); found != nil {
			return found
		}
	}
	return nil
}

func (h *KotlinHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}

	// If we get a source_file, we need to check its children
	if node.Type() == "source_file" {
		// First, look for property_declaration
		var propertyFound bool
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "property_declaration" {
				propertyFound = true
				break
			}
		}

		// If we found a property, check for getter/setter siblings
		if propertyFound {
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				if child.Type() == "getter" || child.Type() == "setter" {
					return true
				}
			}
		}

		// Check for getter/setter functions
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "function_declaration" {
				// Find the simple_identifier node which contains the function name
				for j := 0; j < int(child.NamedChildCount()); j++ {
					nameNode := child.NamedChild(j)
					if nameNode.Type() == "simple_identifier" {
						name := string(content[nameNode.StartByte():nameNode.EndByte()])
						if strings.HasPrefix(strings.ToLower(name), "get") || strings.HasPrefix(strings.ToLower(name), "set") {
							return true
						}
						break // Found the name node, no need to continue
					}
				}
			}
		}
		return false
	}

	switch node.Type() {
	case "property_declaration":
		// For property declarations, check siblings for getter/setter
		if parent := node.Parent(); parent != nil {
			for i := 0; i < int(parent.NamedChildCount()); i++ {
				child := parent.NamedChild(i)
				if child.Type() == "getter" || child.Type() == "setter" {
					return true
				}
			}
		}
	case "getter", "setter":
		return true
	case "function_declaration":
		// Find the simple_identifier node which contains the function name
		for i := 0; i < int(node.NamedChildCount()); i++ {
			nameNode := node.NamedChild(i)
			if nameNode.Type() == "simple_identifier" {
				name := string(content[nameNode.StartByte():nameNode.EndByte()])
				if strings.HasPrefix(strings.ToLower(name), "get") || strings.HasPrefix(strings.ToLower(name), "set") {
					return true
				}
				break // Found the name node, no need to continue
			}
		}
	}

	return false
}
