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
					if !strings.Contains(xmlContent, fmt.Sprintf("index=\"%d\"", expectedIndex)) {
						t.Errorf("Missing index %d", expectedIndex)
					}
					if !strings.Contains(xmlContent, "<source>"+content.Path+"</source>") {
						t.Errorf("Missing source %s", content.Path)
					}
					if !strings.Contains(xmlContent, "<document_content>"+content.Content+"</document_content>") {
						t.Errorf("Missing content for document %d", expectedIndex)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "output"+string(tt.outputType))
			generator := NewOutputGenerator(&MixOptions{
				OutputPath: outputPath,
				OutputType: tt.outputType,
			})

			err := generator.Generate(contents)
			if err != nil {
				t.Fatalf("Failed to generate output: %v", err)
			}

			tt.validate(t, outputPath, contents)
		})
	}
}
