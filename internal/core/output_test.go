package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOutputGenerator(t *testing.T) {
	// Create test data
	contents := []FileContent{
		{
			Path:      "test1.go",
			Name:      "test1.go",
			Content:   "package main\n",
			Extension: "go",
			Size:      12,
		},
		{
			Path:      "test2.go",
			Name:      "test2.go",
			Content:   "package test\n",
			Extension: "go",
			Size:      12,
		},
	}

	// Create temporary directory for output files
	tmpDir, err := os.MkdirTemp("", "filefusion-output-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		outputType OutputType
		validate   func(t *testing.T, path string, contents []FileContent)
	}{
		{
			name:       "JSON output",
			outputType: OutputTypeJSON,
			validate: func(t *testing.T, path string, contents []FileContent) {
				var output struct {
					Documents []struct {
						Index           int    `json:"index"`
						Source          string `json:"source"`
						DocumentContent string `json:"document_content"`
					} `json:"documents"`
				}

				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if err := json.Unmarshal(data, &output); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}

				if len(output.Documents) != len(contents) {
					t.Errorf("Expected %d documents, got %d", len(contents), len(output.Documents))
				}

				for i, doc := range output.Documents {
					if doc.Index != i+1 {
						t.Errorf("Expected index %d, got %d", i+1, doc.Index)
					}
					if doc.Source != contents[i].Path {
						t.Errorf("Expected source %s, got %s", contents[i].Path, doc.Source)
					}
					if doc.DocumentContent != contents[i].Content {
						t.Errorf("Content mismatch for document %d", i+1)
					}
				}
			},
		},
		{
			name:       "YAML output",
			outputType: OutputTypeYAML,
			validate: func(t *testing.T, path string, contents []FileContent) {
				var output struct {
					Documents []struct {
						Index           int    `yaml:"index"`
						Source          string `yaml:"source"`
						DocumentContent string `yaml:"document_content"`
					} `yaml:"documents"`
				}

				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				if err := yaml.Unmarshal(data, &output); err != nil {
					t.Fatalf("Failed to parse YAML: %v", err)
				}

				if len(output.Documents) != len(contents) {
					t.Errorf("Expected %d documents, got %d", len(contents), len(output.Documents))
				}

				for i, doc := range output.Documents {
					if doc.Index != i+1 {
						t.Errorf("Expected index %d, got %d", i+1, doc.Index)
					}
					if doc.Source != contents[i].Path {
						t.Errorf("Expected source %s, got %s", contents[i].Path, doc.Source)
					}
					if doc.DocumentContent != contents[i].Content {
						t.Errorf("Content mismatch for document %d", i+1)
					}
				}
			},
		},
		{
			name:       "XML output",
			outputType: OutputTypeXML,
			validate: func(t *testing.T, path string, contents []FileContent) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}

				xmlContent := string(data)
				if !strings.Contains(xmlContent, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
					t.Error("XML declaration missing")
				}

				if !strings.Contains(xmlContent, "<documents>") {
					t.Error("Root element missing")
				}

				for i, content := range contents {
					expectedIndex := i + 1
					expectedSource := fmt.Sprintf("<source>%s</source>", content.Path)
					expectedContent := fmt.Sprintf("<document_content>%s</document_content>", content.Content)

					if !strings.Contains(xmlContent, fmt.Sprintf("index=\"%d\"", expectedIndex)) {
						t.Errorf("Missing index %d", expectedIndex)
					}
					if !strings.Contains(xmlContent, expectedSource) {
						t.Errorf("Missing source %s", content.Path)
					}
					if !strings.Contains(xmlContent, expectedContent) {
						t.Errorf("Missing content for document %d", expectedIndex)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("output-%s", string(tt.outputType)))
			generator := NewOutputGenerator(&MixOptions{
				OutputPath:    outputPath,
				MaxOutputSize: 10 * 1024 * 1024, // 10MB limit for tests
				OutputType:    tt.outputType,
			})

			if err := generator.Generate(contents); err != nil {
				t.Fatalf("Failed to generate output: %v", err)
			}

			tt.validate(t, outputPath, contents)
		})
	}
}

func TestOutputSizeLimit(t *testing.T) {
	// Create temporary directory for output files
	tmpDir, err := os.MkdirTemp("", "filefusion-output-size-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test content that will generate a predictable output size
	largeContent := make([]FileContent, 5)
	contentSize := 1024 * 1024 // 1MB per file
	content := strings.Repeat("x", contentSize)
	for i := range largeContent {
		largeContent[i] = FileContent{
			Path:    fmt.Sprintf("test%d.go", i),
			Name:    fmt.Sprintf("test%d.go", i),
			Content: content,
			Size:    int64(contentSize),
		}
	}

	tests := []struct {
		name        string
		maxSize     int64
		shouldError bool
		outputType  OutputType
	}{
		{
			name:        "within size limit",
			maxSize:     10 * 1024 * 1024, // 10MB
			shouldError: false,
			outputType:  OutputTypeXML,
		},
		{
			name:        "exceeds size limit",
			maxSize:     1 * 1024 * 1024, // 1MB
			shouldError: true,
			outputType:  OutputTypeXML,
		},
		{
			name:        "within size limit json",
			maxSize:     10 * 1024 * 1024,
			shouldError: false,
			outputType:  OutputTypeJSON,
		},
		{
			name:        "exceeds size limit json",
			maxSize:     1 * 1024 * 1024,
			shouldError: true,
			outputType:  OutputTypeJSON,
		},
		{
			name:        "within size limit yaml",
			maxSize:     10 * 1024 * 1024,
			shouldError: false,
			outputType:  OutputTypeYAML,
		},
		{
			name:        "exceeds size limit yaml",
			maxSize:     1 * 1024 * 1024,
			shouldError: true,
			outputType:  OutputTypeYAML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("output-%s%s",
				tt.name, string(tt.outputType)))

			generator := NewOutputGenerator(&MixOptions{
				OutputPath:    outputPath,
				MaxOutputSize: tt.maxSize,
				OutputType:    tt.outputType,
			})

			err := generator.Generate(largeContent)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), "output size") {
					t.Errorf("Expected output size error, got: %v", err)
				}
				// Verify the output file was not created
				if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
					t.Error("Output file should not exist")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				// Verify the output file exists and is within size limit
				info, err := os.Stat(outputPath)
				if err != nil {
					t.Errorf("Failed to stat output file: %v", err)
					return
				}
				if info.Size() > tt.maxSize {
					t.Errorf("Output file size %d exceeds limit %d", info.Size(), tt.maxSize)
				}
			}
		})
	}
}

func TestOutputSymlinkHandling(t *testing.T) {
	if os.Getenv("SKIP_SYMLINK_TESTS") != "" {
		t.Skip("Skipping symlink tests")
	}

	tmpDir, err := os.MkdirTemp("", "filefusion-symlink-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory for output
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create a symlink to the output directory
	symlinkPath := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(outputDir, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Create test content
	contents := []FileContent{
		{
			Path:    "test.go",
			Content: "package main\n",
			Size:    12,
		},
	}

	// Test writing through symlink
	outputPath := filepath.Join(symlinkPath, "output.xml")
	generator := NewOutputGenerator(&MixOptions{
		OutputPath:    outputPath,
		MaxOutputSize: 1024,
		OutputType:    OutputTypeXML,
	})

	if err := generator.Generate(contents); err != nil {
		t.Fatalf("Failed to generate output through symlink: %v", err)
	}

	// Verify file exists in actual directory
	realPath := filepath.Join(outputDir, "output.xml")
	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		t.Error("Output file not created in real directory")
	}
}
