package core

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

// OutputGenerator handles the creation of output files in various formats (XML, JSON, YAML).
// It implements safety measures such as:
// - Using temporary files for atomic writes
// - Size limit validation
// - Proper error handling and cleanup
type OutputGenerator struct {
	options *MixOptions // Configuration options for output generation
}

// NewOutputGenerator creates a new OutputGenerator instance with the specified options.
//
// Parameters:
//   - options: Configuration settings for output generation
//
// Returns:
//   - A new OutputGenerator instance
func NewOutputGenerator(options *MixOptions) *OutputGenerator {
	return &OutputGenerator{options: options}
}

// Generate creates an output file containing the provided file contents in the specified format.
// The function implements a safe generation process:
// 1. Creates a temporary file
// 2. Writes content in the specified format
// 3. Validates size constraints
// 4. Atomically moves the file to its final location
//
// Parameters:
//   - contents: Slice of FileContent to include in the output
//
// Returns:
//   - error: nil if successful, otherwise describes what went wrong
func (g *OutputGenerator) Generate(contents []FileContent) error {
	// Create a temporary file for safe writing
	tempFile, err := os.CreateTemp("", "filefusion-*")
	if err != nil {
		return &MixError{
			File:    g.options.OutputPath,
			Message: fmt.Sprintf("error creating temporary file: %v", err),
		}
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up temp file in case of failure

	// Generate content in the specified format
	switch g.options.OutputType {
	case OutputTypeJSON:
		err = g.generateJSON(tempFile, contents)
	case OutputTypeYAML:
		err = g.generateYAML(tempFile, contents)
	case OutputTypeXML:
		err = g.generateXML(tempFile, contents)
	default:
		return &MixError{Message: fmt.Sprintf("unsupported output type: %s", g.options.OutputType)}
	}

	if err != nil {
		return err
	}

	// Close the temp file to ensure all content is written
	tempFile.Close()

	// Check the size of the generated file
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

// generateJSON creates a JSON output file with the provided file contents.
// It wraps the contents in a "documents" array and writes it to the specified file.
//
// Parameters:
//   - file: The file to write the JSON output to
//   - contents: Slice of FileContent to include in the output
//
// Returns:
//   - error: nil if successful, otherwise describes what went wrong
func (g *OutputGenerator) generateJSON(file *os.File, contents []FileContent) error {
	// Create wrapper structure for consistent output format
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

	// Fill the structure
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

// generateYAML writes the content in YAML format with proper indentation.
// The output maintains the same structure as JSON for consistency.
//
// Parameters:
//   - file: Open file to write to
//   - output: Interface containing the data to encode
//
// Returns:
//   - error: nil if successful, error if encoding fails
func (g *OutputGenerator) generateYAML(file *os.File, output interface{}) error {
	// Use same structure as JSON for consistency
	docs := struct {
		Documents []struct {
			Index           int    `yaml:"index"`
			Source          string `yaml:"source"`
			DocumentContent string `yaml:"document_content"`
		} `yaml:"documents"`
	}{}

	// Convert the input to []FileContent
	contents, ok := output.([]FileContent)
	if !ok {
		return &MixError{Message: "invalid input type for YAML generation"}
	}

	// Fill the structure
	docs.Documents = make([]struct {
		Index           int    `yaml:"index"`
		Source          string `yaml:"source"`
		DocumentContent string `yaml:"document_content"`
	}, len(contents))

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

// generateXML writes the content in XML format using a template.
// The output includes proper XML declaration and document structure.
//
// Parameters:
//   - file: Open file to write to
//   - contents: Slice of FileContent to encode
//
// Returns:
//   - error: nil if successful, error if template execution fails
func (g *OutputGenerator) generateXML(file *os.File, contents []FileContent) error {
	const xmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<documents>{{range $index, $file := .}}
<document index="{{add $index 1}}">
<source>{{.Path}}</source>
<document_content>{{.Content}}</document_content>
</document>{{end}}
</documents>`

	t, err := template.New("llm").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(xmlTemplate)
	if err != nil {
		return &MixError{Message: fmt.Sprintf("error parsing template: %v", err)}
	}

	if err := t.Execute(file, contents); err != nil {
		return &MixError{Message: fmt.Sprintf("error executing template: %v", err)}
	}
	return nil
}
