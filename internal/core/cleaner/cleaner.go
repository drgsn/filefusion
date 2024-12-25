package cleaner

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/drgsn/filefusion/internal/core/cleaner/handlers"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/bash"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/css"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/html"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/php"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/sql"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// Cleaner represents a code cleaner instance that processes source code
// according to the configured options
type Cleaner struct {
	options  *CleanerOptions
	language Language
	handler  handlers.LanguageHandler
}

// NewCleaner creates a new Cleaner instance with the given options and language
func NewCleaner(lang Language, options *CleanerOptions) (*Cleaner, error) {
	if options == nil {
		return nil, fmt.Errorf("cleaner options cannot be nil")
	}

	_, handler, err := getLanguageAndHandler(lang)
	if err != nil {
		return nil, err
	}

	return &Cleaner{
		options:  options,
		language: lang,
		handler:  handler,
	}, nil
}

// getLanguageAndHandler returns the appropriate tree-sitter language parser
// and language handler for the given language
func getLanguageAndHandler(lang Language) (*sitter.Language, handlers.LanguageHandler, error) {
	switch lang {
	case LangGo:
		return golang.GetLanguage(), &handlers.GoHandler{}, nil
	case LangJava:
		return java.GetLanguage(), &handlers.JavaHandler{}, nil
	case LangPython:
		return python.GetLanguage(), &handlers.PythonHandler{}, nil
	case LangSwift:
		return swift.GetLanguage(), &handlers.SwiftHandler{}, nil
	case LangKotlin:
		return kotlin.GetLanguage(), &handlers.KotlinHandler{}, nil
	case LangSQL:
		return sql.GetLanguage(), &handlers.SQLHandler{}, nil
	case LangHTML:
		return html.GetLanguage(), &handlers.HTMLHandler{}, nil
	case LangJavaScript:
		return javascript.GetLanguage(), &handlers.JavaScriptHandler{}, nil
	case LangTypeScript:
		return typescript.GetLanguage(), &handlers.TypeScriptHandler{}, nil
	case LangCSS:
		return css.GetLanguage(), &handlers.CSSHandler{}, nil
	case LangCPP:
		return cpp.GetLanguage(), &handlers.CPPHandler{}, nil
	case LangCSharp:
		return csharp.GetLanguage(), &handlers.CSharpHandler{}, nil
	case LangPHP:
		return php.GetLanguage(), &handlers.PHPHandler{}, nil
	case LangRuby:
		return ruby.GetLanguage(), &handlers.RubyHandler{}, nil
	case LangBash:
		return bash.GetLanguage(), &handlers.BashHandler{}, nil
	default:
		return nil, nil, fmt.Errorf("unsupported language: %s", lang)
	}
}

// Clean processes the input code and returns the cleaned version
func (c *Cleaner) Clean(input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	// Create a new parser for each Clean call to avoid concurrency issues
	parser := sitter.NewParser()
	language, _, err := getLanguageAndHandler(c.language)
	if err != nil {
		return nil, fmt.Errorf("failed to get language handler: %w", err)
	}
	parser.SetLanguage(language)

	tree := parser.Parse(nil, input)
	if tree == nil {
		return nil, fmt.Errorf("parsing error: failed to create syntax tree")
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("parsing error: empty syntax tree")
	}

	// Verify the syntax is valid
	if root.HasError() {
		return nil, fmt.Errorf("parsing error: invalid syntax")
	}

	output := make([]byte, len(input))
	copy(output, input)

	if err := c.processNode(root, &output); err != nil {
		return nil, fmt.Errorf("processing error: %w", err)
	}

	if c.options.OptimizeWhitespace {
		output = c.optimizeWhitespace(output)
	}

	return output, nil
}

// processNode recursively processes a node in the syntax tree
func (c *Cleaner) processNode(node *sitter.Node, content *[]byte) error {
	if !c.shouldProcessNode(node, content) {
		return nil
	}

	// Process children in reverse order to maintain correct byte offsets
	for i := int(node.NamedChildCount()) - 1; i >= 0; i-- {
		child := node.NamedChild(i)
		if err := c.processNode(child, content); err != nil {
			return err
		}
	}

	// Check if this node should be removed
	shouldRemove := false

	// Process comments
	for _, commentType := range c.handler.GetCommentTypes() {
		if node.Type() == commentType && c.shouldRemoveComment(node, *content) {
			shouldRemove = true
			break
		}
	}

	// Process logging calls
	if c.options.RemoveLogging && c.handler.IsLoggingCall(node, *content) {
		shouldRemove = true
	}

	// Process getters/setters
	if c.options.RemoveGettersSetters && c.handler.IsGetterSetter(node, *content) {
		shouldRemove = true
	}

	// Remove the node if necessary
	if shouldRemove {
		*content = c.removeNode(node, *content)
	}

	return nil
}

// shouldProcessNode determines if a node should be processed based on its type
// and position in the syntax tree
func (c *Cleaner) shouldProcessNode(node *sitter.Node, content *[]byte) bool {
	// Skip processing for nil nodes or empty content
	if node == nil || len(*content) == 0 {
		return false
	}

	// Skip processing for nodes outside content bounds
	if node.StartByte() >= uint32(len(*content)) || node.EndByte() > uint32(len(*content)) {
		return false
	}

	return true
}

// shouldRemoveComment determines if a comment should be removed based on
// the cleaner options and whether it's a documentation comment
func (c *Cleaner) shouldRemoveComment(node *sitter.Node, content []byte) bool {
	if !c.options.RemoveComments {
		return false
	}

	if c.options.PreserveDocComments {
		commentText := content[node.StartByte():node.EndByte()]
		docPrefix := c.handler.GetDocCommentPrefix()

		// Handle both byte slices and string prefixes
		if bytes.HasPrefix(bytes.TrimSpace(commentText), []byte(docPrefix)) {
			return false
		}

		// Special handling for single-line comments in Go
		if bytes.HasPrefix(bytes.TrimSpace(commentText), []byte("// ")) {
			text := string(bytes.TrimSpace(commentText))
			if strings.HasPrefix(text, "// Doc ") {
				return false
			}
		}
	}

	return true
}

// removeNode removes a node from the content while preserving the surrounding
// content
func (c *Cleaner) removeNode(node *sitter.Node, content []byte) []byte {
	// Get the start and end of the line containing the node
	start := node.StartByte()
	end := node.EndByte()

	// Find start of line
	lineStart := int(start)
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}

	// Find end of line
	lineEnd := int(end)
	for lineEnd < len(content) && content[lineEnd] != '\n' {
		lineEnd++
	}

	// If the line contains only this node (plus whitespace), remove the entire line
	line := bytes.TrimSpace(content[lineStart:lineEnd])
	nodeContent := bytes.TrimSpace(content[start:end])
	if bytes.Equal(line, nodeContent) {
		return append(content[:lineStart], content[lineEnd:]...)
	}

	// Otherwise just remove the node itself
	return append(content[:start], content[end:]...)
}

// optimizeWhitespace removes excess whitespace and optionally empty lines
// from the content
func (c *Cleaner) optimizeWhitespace(content []byte) []byte {
	if !c.options.OptimizeWhitespace {
		return content
	}

	lines := bytes.Split(content, []byte("\n"))
	var result [][]byte
	var previousLineEmpty bool

	for i := range lines {
		line := bytes.TrimRight(lines[i], " \t")
		isEmpty := len(bytes.TrimSpace(line)) == 0

		if !isEmpty || (!c.options.RemoveEmptyLines && !previousLineEmpty) {
			result = append(result, line)
		}
		previousLineEmpty = isEmpty
	}

	return append(bytes.Join(result, []byte("\n")), '\n')
}

// CleanFile processes a file and writes the cleaned content to the writer
func (c *Cleaner) CleanFile(r io.Reader, w io.Writer) error {
	input, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading error: %w", err)
	}

	cleaned, err := c.Clean(input)
	if err != nil {
		return err
	}

	_, err = w.Write(cleaned)
	return err
}
