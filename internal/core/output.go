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
	file, err := os.Create(g.options.OutputPath)
	if err != nil {
		return &MixError{
			File:    g.options.OutputPath,
			Message: fmt.Sprintf("error creating output file: %v", err),
		}
	}
	defer file.Close()

	// Common structure for JSON and YAML output
	type Document struct {
		Index           int    `json:"index" yaml:"index"`
		Source          string `json:"source" yaml:"source"`
		DocumentContent string `json:"document_content" yaml:"document_content"`
	}

	type Output struct {
		Documents []Document `json:"documents" yaml:"documents"`
	}

	// Convert contents to output format
	output := Output{
		Documents: make([]Document, len(contents)),
	}

	for i, content := range contents {
		output.Documents[i] = Document{
			Index:           i + 1,
			Source:          content.Path,
			DocumentContent: content.Content,
		}
	}

	switch g.options.OutputType {
	case OutputTypeJSON:
		return g.generateJSON(file, output)
	case OutputTypeYAML:
		return g.generateYAML(file, output)
	case OutputTypeXML:
		return g.generateXML(file, contents)
	default:
		return &MixError{Message: fmt.Sprintf("unsupported output type: %s", g.options.OutputType)}
	}
}

func (g *OutputGenerator) generateJSON(file *os.File, output interface{}) error {
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return &MixError{Message: fmt.Sprintf("error encoding JSON: %v", err)}
	}
	return nil
}

func (g *OutputGenerator) generateYAML(file *os.File, output interface{}) error {
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(output); err != nil {
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
