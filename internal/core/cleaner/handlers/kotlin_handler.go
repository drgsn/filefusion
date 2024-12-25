package handlers

import (
    "bytes"
    "strings"
    sitter "github.com/smacker/go-tree-sitter"
)

// KotlinHandler handles Kotlin language specifics
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
    if node.Type() != "call_expression" {
        return false
    }
    callText := content[node.StartByte():node.EndByte()]
    return bytes.HasPrefix(callText, []byte("println(")) ||
           bytes.HasPrefix(callText, []byte("print(")) ||
           bytes.Contains(callText, []byte("Logger")) ||
           bytes.Contains(callText, []byte(".log"))
}

func (h *KotlinHandler) IsGetterSetter(node *sitter.Node, content []byte) bool {
    nodeType := node.Type()
    
    // Check for property declarations with explicit getters/setters
    if nodeType == "property_declaration" {
        propText := string(content[node.StartByte():node.EndByte()])
        return strings.Contains(propText, "get()") || strings.Contains(propText, "set(")
    }
    
    // Check for function declarations that might be getters/setters
    if nodeType == "function_declaration" {
        funcName := string(content[node.StartByte():node.EndByte()])
        return strings.HasPrefix(funcName, "get") || strings.HasPrefix(funcName, "set")
    }
    
    return false
}