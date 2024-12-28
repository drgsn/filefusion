package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// OutputGenerator handles the creation of output files in various formats
type OutputGenerator struct {
	options *MixOptions
	workDir string // Current working directory for path normalization
}

// NewOutputGenerator creates a new OutputGenerator instance
func NewOutputGenerator(options *MixOptions) (*OutputGenerator, error) {
	if options == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	return &OutputGenerator{
		options: options,
		workDir: workDir,
	}, nil
}

// normalizePath removes all but the last directory from the working directory path
// and normalizes path separators
func (g *OutputGenerator) normalizePath(path string) string {
	// Handle empty path
	if path == "" {
		return filepath.Base(g.workDir)
	}

	// Convert Windows backslashes to forward slashes first
	path = strings.ReplaceAll(path, "\\", "/")
	workDir := strings.ReplaceAll(g.workDir, "\\", "/")

	// Remove drive letters if present (Windows paths)
	if len(path) >= 2 && path[1] == ':' {
		path = path[2:]
	}
	if len(workDir) >= 2 && workDir[1] == ':' {
		workDir = workDir[2:]
	}

	// Clean the paths to handle .. and .
	path = filepath.Clean(strings.TrimPrefix(path, "/"))
	workDir = filepath.Clean(strings.TrimPrefix(workDir, "/"))

	// Get the last directory from workDir
	lastDir := filepath.Base(workDir)

	// If the path starts with the working directory, remove it except for the last part
	if strings.HasPrefix(path, workDir) {
		path = strings.TrimPrefix(path, workDir)
		path = strings.TrimPrefix(path, "/")
		return filepath.Join(lastDir, path)
	}

	// For paths not under working directory, just prefix with lastDir
	return filepath.Join(lastDir, path)
}

// Generate creates an output file containing the provided file contents
func (g *OutputGenerator) Generate(contents []FileContent) error {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "filefusion-*")
	if err != nil {
		return &MixError{
			File:    g.options.OutputPath,
			Message: fmt.Sprintf("error creating temporary file: %v", err),
		}
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// Normalize paths in contents
	normalizedContents := make([]FileContent, len(contents))
	for i, content := range contents {
		normalizedContents[i] = FileContent{
			Path:      g.normalizePath(content.Path),
			Name:      content.Name,
			Content:   content.Content,
			Extension: content.Extension,
			Size:      content.Size,
		}
	}

	// Generate content in the specified format
	switch g.options.OutputType {
	case OutputTypeJSON:
		err = g.generateJSON(tempFile, normalizedContents)
	case OutputTypeYAML:
		err = g.generateYAML(tempFile, normalizedContents)
	case OutputTypeXML:
		err = g.generateXML(tempFile, normalizedContents)
	default:
		return &MixError{Message: fmt.Sprintf("unsupported output type: %s", g.options.OutputType)}
	}

	if err != nil {
		return err
	}

	// Close the temp file
	tempFile.Close()

	// Check the size
	info, err := os.Stat(tempPath)
	if err != nil {
		return &MixError{Message: fmt.Sprintf("error checking output file size: %v", err)}
	}

	if info.Size() > g.options.MaxOutputSize {
		return &MixError{
			Message: fmt.Sprintf("output size (%d bytes) exceeds maximum allowed size (%d bytes)",
				info.Size(), g.options.MaxOutputSize),
		}
	}

	// Move temp file to final destination
	return os.Rename(tempPath, g.options.OutputPath)
}

// generateJSON creates a JSON output file
func (g *OutputGenerator) generateJSON(file *os.File, contents []FileContent) error {
	output := struct {
		Documents []struct {
			Index           int    `json:"index"`
			Source          string `json:"source"`
			DocumentContent string `json:"document_content"`
		} `json:"documents"`
	}{
		Documents: make([]struct {
			Index           int    `json:"index"`
			Source          string `json:"source"`
			DocumentContent string `json:"document_content"`
		}, len(contents)),
	}

	for i, content := range contents {
		output.Documents[i] = struct {
			Index           int    `json:"index"`
			Source          string `json:"source"`
			DocumentContent string `json:"document_content"`
		}{
			Index:           i + 1,
			Source:          content.Path,
			DocumentContent: content.Content,
		}
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return &MixError{Message: fmt.Sprintf("error encoding JSON: %v", err)}
	}
	return nil
}

// generateYAML writes the content in YAML format
func (g *OutputGenerator) generateYAML(file *os.File, contents []FileContent) error {
	docs := struct {
		Documents []struct {
			Index           int    `yaml:"index"`
			Source          string `yaml:"source"`
			DocumentContent string `yaml:"document_content"`
		} `yaml:"documents"`
	}{
		Documents: make([]struct {
			Index           int    `yaml:"index"`
			Source          string `yaml:"source"`
			DocumentContent string `yaml:"document_content"`
		}, len(contents)),
	}

	for i, content := range contents {
		docs.Documents[i] = struct {
			Index           int    `yaml:"index"`
			Source          string `yaml:"source"`
			DocumentContent string `yaml:"document_content"`
		}{
			Index:           i + 1,
			Source:          content.Path,
			DocumentContent: content.Content,
		}
	}

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(docs); err != nil {
		return &MixError{Message: fmt.Sprintf("error encoding YAML: %v", err)}
	}
	return nil
}

// generateXML writes the content in XML format
func (g *OutputGenerator) generateXML(file *os.File, contents []FileContent) error {
	const xmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<documents>{{range $index, $file := .}}
<document index="{{add $index 1}}">
<source>{{.Path}}</source>
<document_content>{{- escapeXML .Content -}}</document_content>
</document>{{end}}
</documents>`

	t, err := template.New("llm").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"escapeXML": func(s string) string {
			s = strings.ReplaceAll(s, "&", "&amp;")
			s = strings.ReplaceAll(s, "<", "&lt;")
			s = strings.ReplaceAll(s, ">", "&gt;")
			s = strings.ReplaceAll(s, "'", "&apos;")
			s = strings.ReplaceAll(s, "\"", "&quot;")
			return s
		},
	}).Parse(xmlTemplate)
	if err != nil {
		return &MixError{Message: fmt.Sprintf("error parsing template: %v", err)}
	}

	if err := t.Execute(file, contents); err != nil {
		return &MixError{Message: fmt.Sprintf("error executing template: %v", err)}
	}
	return nil
}
