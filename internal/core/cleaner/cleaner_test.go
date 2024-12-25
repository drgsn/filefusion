package cleaner

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestNewCleaner(t *testing.T) {
	tests := []struct {
		name        string
		lang        Language
		options     *CleanerOptions
		shouldError bool
	}{
		{
			name:        "valid Go cleaner",
			lang:        LangGo,
			options:     DefaultOptions(),
			shouldError: false,
		},
		{
			name:        "nil options",
			lang:        LangGo,
			options:     nil,
			shouldError: true,
		},
		{
			name:        "unsupported language",
			lang:        "invalid",
			options:     DefaultOptions(),
			shouldError: true,
		},
		{
			name:        "valid Java cleaner",
			lang:        LangJava,
			options:     DefaultOptions(),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaner, err := NewCleaner(tt.lang, tt.options)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if cleaner == nil {
				t.Error("Expected non-nil cleaner")
			}
		})
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		name           string
		lang           Language
		input          string
		options        *CleanerOptions
		expected       string
		shouldContain  []string
		shouldNotMatch []string
		shouldError    bool
	}{
		{
			name:  "remove Go comments",
			lang:  LangGo,
			input: "package main\n// This is a comment\nfunc main() {}\n/* Block comment */\n",
			options: &CleanerOptions{
				RemoveComments:      true,
				PreserveDocComments: false,
				OptimizeWhitespace:  true,
				RemoveEmptyLines:    true,
			},
			expected: "package main\nfunc main() {}\n",
		},
		{
			name:  "preserve Go doc comments",
			lang:  LangGo,
			input: "package main\n// Regular comment\n/// Doc comment\nfunc main() {}\n",
			options: &CleanerOptions{
				RemoveComments:      true,
				PreserveDocComments: true,
			},
			shouldContain:  []string{"/// Doc comment"},
			shouldNotMatch: []string{"// Regular comment"},
		},
		{
			name:  "remove Java logging",
			lang:  LangJava,
			input: "class Test {\nvoid test() {\nlog.info(\"test\");\nSystem.out.println(\"debug\");\n}\n}",
			options: &CleanerOptions{
				RemoveLogging: true,
			},
			shouldNotMatch: []string{"log.info", "System.out.println"},
		},
		{
			name:  "remove getters",
			lang:  LangJava,
			input: "public class Test {\n    public String getName() { return name; }\n    void otherMethod() {}\n}",
			options: &CleanerOptions{
				RemoveGettersSetters: true,
			},
			shouldContain:  []string{"otherMethod"},
			shouldNotMatch: []string{"getName"},
		},
		{
			name:  "optimize whitespace",
			lang:  LangGo,
			input: "package main\n\n\nfunc main() {\n\n\n}\n\n",
			options: &CleanerOptions{
				OptimizeWhitespace: true,
				RemoveEmptyLines:   true,
			},
			expected: "package main\nfunc main() {\n}\n",
		},
		{
			name:  "invalid syntax",
			lang:  LangGo,
			input: "package main\nfunc main() {",
			options: &CleanerOptions{
				RemoveComments: true,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaner, err := NewCleaner(tt.lang, tt.options)
			if err != nil {
				t.Fatalf("Failed to create cleaner: %v", err)
			}

			output, err := cleaner.Clean([]byte(tt.input))
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check expected exact match
			if tt.expected != "" && string(output) != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, string(output))
			}

			// Check content that should be present
			for _, s := range tt.shouldContain {
				if !bytes.Contains(output, []byte(s)) {
					t.Errorf("Expected output to contain %q", s)
				}
			}

			// Check content that should be removed
			for _, s := range tt.shouldNotMatch {
				if bytes.Contains(output, []byte(s)) {
					t.Errorf("Expected output to not contain %q", s)
				}
			}
		})
	}
}

func TestCleanFile(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		options     *CleanerOptions
		shouldError bool
	}{
		{
			name:        "valid input",
			input:       "package main\nfunc main() {}\n",
			shouldError: false,
		},
		{
			name:        "invalid reader",
			input:       "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaner, err := NewCleaner(LangGo, DefaultOptions())
			if err != nil {
				t.Fatalf("Failed to create cleaner: %v", err)
			}

			var input bytes.Buffer
			if tt.name == "invalid reader" {
				// Create a custom reader that always returns an error
				r := &errorReader{err: fmt.Errorf("read error")}
				var output bytes.Buffer
				err = cleaner.CleanFile(r, &output)
			} else {
				input.WriteString(tt.input)
				var output bytes.Buffer
				err = cleaner.CleanFile(&input, &output)
			}

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
		})
	}
}

func TestProcessNode(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	input := []byte("package main\n// Comment\nfunc main() {}\n")
	tree, err := parser.ParseCtx(context.Background(), nil, input)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}
	defer tree.Close()

	tests := []struct {
		name           string
		options        *CleanerOptions
		shouldContain  []string
		shouldNotMatch []string
	}{
		{
			name: "remove comments",
			options: &CleanerOptions{
				RemoveComments:      true,
				PreserveDocComments: false,
			},
			shouldNotMatch: []string{"// Comment"},
		},
		{
			name: "preserve structure",
			options: &CleanerOptions{
				RemoveComments: true,
			},
			shouldContain: []string{"package main", "func main()"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaner, err := NewCleaner(LangGo, tt.options)
			if err != nil {
				t.Fatalf("Failed to create cleaner: %v", err)
			}

			content := make([]byte, len(input))
			copy(content, input)

			err = cleaner.processNode(tree.RootNode(), &content)
			if err != nil {
				t.Fatalf("Failed to process node: %v", err)
			}

			// Check content that should be present
			for _, s := range tt.shouldContain {
				if !bytes.Contains(content, []byte(s)) {
					t.Errorf("Expected output to contain %q", s)
				}
			}

			// Check content that should be removed
			for _, s := range tt.shouldNotMatch {
				if bytes.Contains(content, []byte(s)) {
					t.Errorf("Expected output to not contain %q", s)
				}
			}
		})
	}
}

func TestShouldRemoveComment(t *testing.T) {
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	tests := []struct {
		name        string
		options     *CleanerOptions
		commentText string
		shouldKeep  bool
	}{
		{
			name: "regular comment with removal enabled",
			options: &CleanerOptions{
				RemoveComments:      true,
				PreserveDocComments: false,
			},
			commentText: "// Regular comment",
			shouldKeep:  false,
		},
		{
			name: "doc comment with preservation enabled",
			options: &CleanerOptions{
				RemoveComments:      true,
				PreserveDocComments: true,
			},
			commentText: "// Doc comment",
			shouldKeep:  true,
		},
		{
			name: "comment with removal disabled",
			options: &CleanerOptions{
				RemoveComments:      false,
				PreserveDocComments: true,
			},
			commentText: "// Any comment",
			shouldKeep:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaner, err := NewCleaner(LangGo, tt.options)
			if err != nil {
				t.Fatalf("Failed to create cleaner: %v", err)
			}

			tree, err := parser.ParseCtx(context.Background(), nil, []byte(tt.commentText))
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}
			defer tree.Close()

			node := tree.RootNode()
			shouldRemove := cleaner.shouldRemoveComment(node, []byte(tt.commentText))

			if shouldRemove == tt.shouldKeep {
				t.Errorf("Expected shouldRemove=%v for comment %q", !tt.shouldKeep, tt.commentText)
			}
		})
	}
}
