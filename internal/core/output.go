package core

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

// OutputGenerator handles the generation of output files
type OutputGenerator struct {
	options *MixOptions
}

// NewOutputGenerator creates a new OutputGenerator instance
func NewOutputGenerator(options *MixOptions) *OutputGenerator {
	return &OutputGenerator{options: options}
}

// Generate creates the output file in the specified format
func (g *OutputGenerator) Generate(contents []FileContent) error {
	// Create a temporary file first
	tempFile, err := os.CreateTemp("", "filefusion-*")
	if err != nil {
		return &MixError{
			File:    g.options.OutputPath,
			Message: fmt.Sprintf("error creating temporary file: %v", err),
		}
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up temp file

	// Generate content to temporary file
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

	// If size is OK, move the temp file to the final destination
	return os.Rename(tempPath, g.options.OutputPath)
}

// In internal/core/output.go

func (g *OutputGenerator) generateJSON(file *os.File, contents []FileContent) error {
	// Create a wrapper structure
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

func (g *OutputGenerator) generateYAML(file *os.File, output interface{}) error {
	// Create the same structure as JSON
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
