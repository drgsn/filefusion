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

// Mixer handles file concatenation
type Mixer struct {
    options *MixOptions
}

// NewMixer creates a new Mixer instance
func NewMixer(options *MixOptions) *Mixer {
    if options.MaxFileSize == 0 {
        options.MaxFileSize = 10 * 1024 * 1024 // Default 10MB
    }
    return &Mixer{options: options}
}

// Mix processes and concatenates files
func (m *Mixer) Mix() error {
    // Find all matching files
    files, err := m.findFiles()
    if err != nil {
        return fmt.Errorf("error finding files: %w", err)
    }

    // Read and process files
    contents, err := m.readFiles(files)
    if err != nil {
        return fmt.Errorf("error reading files: %w", err)
    }

    // Generate LLM-optimized output
    return m.generateLLMOutput(contents)
}

// matchesAnyPattern checks if a file matches any of the provided patterns
func (m *Mixer) matchesAnyPattern(path, filename string) (bool, error) {
    // First check exclusions
    if m.options.Exclude != "" {
        excludePatterns := strings.Split(m.options.Exclude, ",")
        for _, pattern := range excludePatterns {
            pattern = strings.TrimSpace(pattern)
            if pattern == "" {
                continue
            }

            // Convert all slashes to platform-specific separator
            pattern = filepath.FromSlash(pattern)
            pathToCheck := filepath.FromSlash(path)

            // Handle directory-based exclusions with **
            if strings.Contains(pattern, "**") {
                basePattern := strings.TrimSuffix(pattern, string(filepath.Separator)+"**")
                basePattern = strings.TrimSuffix(basePattern, "**")

                // Check if the path starts with the base pattern (excluding the **)
                if strings.HasPrefix(pathToCheck, basePattern) {
                    return false, nil
                }
            } else if strings.Contains(pattern, string(filepath.Separator)) {
                // Handle path-based exclusions (contains path separator)
                if match, err := filepath.Match(pattern, pathToCheck); err != nil {
                    return false, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
                } else if match {
                    return false, nil
                }
            } else {
                // Handle file-based exclusions (no path separator)
                if match, err := filepath.Match(pattern, filename); err != nil {
                    return false, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
                } else if match {
                    return false, nil
                }
            }
        }
    }

    // Then check inclusion patterns
    patterns := strings.Split(m.options.Pattern, ",")
    for _, pattern := range patterns {
        pattern = strings.TrimSpace(pattern)
        if pattern == "" {
            continue
        }

        // For inclusion patterns, we only match against the filename
        match, err := filepath.Match(pattern, filename)
        if err != nil {
            return false, fmt.Errorf("invalid pattern %q: %w", pattern, err)
        }
        if match {
            return true, nil
        }
    }

    return false, nil
}

// findFiles finds all files matching any of the patterns recursively
func (m *Mixer) findFiles() ([]string, error) {
    var matches []string
    
    err := filepath.Walk(m.options.InputPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return fmt.Errorf("error accessing path %s: %w", path, err)
        }
        
        // Skip directories themselves
        if info.IsDir() {
            return nil
        }

        // Get relative path for pattern matching
        relPath, err := filepath.Rel(m.options.InputPath, path)
        if err != nil {
            relPath = path
        }
        
        // Check if file matches patterns
        match, err := m.matchesAnyPattern(relPath, filepath.Base(path))
        if err != nil {
            return err
        }
        
        if match {
            matches = append(matches, path)
        }
        return nil
    })

    if err != nil {
        return nil, err
    }
    
    if len(matches) == 0 {
        return nil, fmt.Errorf("no files found matching pattern(s) %q (excluding %q) in %s", 
            m.options.Pattern, m.options.Exclude, m.options.InputPath)
    }
    
    return matches, nil
}

// readFiles reads all files and their contents
func (m *Mixer) readFiles(paths []string) ([]FileContent, error) {
    var contents []FileContent

    for _, path := range paths {
        // Get file info
        info, err := os.Stat(path)
        if err != nil {
            return nil, &MixError{
                File:    path,
                Message: fmt.Sprintf("error getting file info: %v", err),
            }
        }

        // Skip if file is too large
        if info.Size() > m.options.MaxFileSize {
            fmt.Fprintf(os.Stderr, "Warning: Skipping %s (size %d bytes exceeds limit %d bytes)\n",
                path, info.Size(), m.options.MaxFileSize)
            continue
        }

        // Read file content
        content, err := os.ReadFile(path)
        if err != nil {
            return nil, &MixError{
                File:    path,
                Message: fmt.Sprintf("error reading file: %v", err),
            }
        }

        // Get relative path from input directory
        relPath, err := filepath.Rel(m.options.InputPath, path)
        if err != nil {
            relPath = path // Fallback to full path if relative path cannot be determined
        }

        contents = append(contents, FileContent{
            Path:      relPath,
            Name:      filepath.Base(path),
            Extension: strings.TrimPrefix(filepath.Ext(path), "."),
            Content:   string(content),
            Size:      info.Size(),
        })
    }

    return contents, nil
}

// generateLLMOutput generates output optimized for LLM consumption
func (m *Mixer) generateLLMOutput(contents []FileContent) error {
    file, err := os.Create(m.options.OutputPath)
    if err != nil {
        return &MixError{
            File:    m.options.OutputPath,
            Message: fmt.Sprintf("error creating output file: %v", err),
        }
    }
    defer file.Close()

    // Common structure for JSON and YAML output
    type Document struct {
        Index          int    `json:"index" yaml:"index"`
        Source         string `json:"source" yaml:"source"`
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
            Index:          i + 1,
            Source:         content.Path,
            DocumentContent: content.Content,
        }
    }

    if m.options.JsonOutput {
        encoder := json.NewEncoder(file)
        encoder.SetIndent("", "  ")
        if err := encoder.Encode(output); err != nil {
            return &MixError{Message: fmt.Sprintf("error encoding JSON: %v", err)}
        }
        return nil
    }

    if m.options.YamlOutput {
        encoder := yaml.NewEncoder(file)
        encoder.SetIndent(2)  // Set YAML indentation to 2 spaces
        if err := encoder.Encode(output); err != nil {
            return &MixError{Message: fmt.Sprintf("error encoding YAML: %v", err)}
        }
        return nil
    }

    // XML output (unchanged)
    const xmlTemplate = `<documents>{{range $index, $file := .}}
<document index="{{add $index 1}}">
<source>{{.Path}}</source>
<document_content>{{.Content}}</document_content>
</document>{{end}}
</documents>`

    // Create template with custom functions
    t, err := template.New("llm").Funcs(template.FuncMap{
        "add": func(a, b int) int { return a + b },
    }).Parse(xmlTemplate)
    if err != nil {
        return &MixError{Message: fmt.Sprintf("error parsing template: %v", err)}
    }

    // Execute template
    if err := t.Execute(file, contents); err != nil {
        return &MixError{Message: fmt.Sprintf("error executing template: %v", err)}
    }

    return nil
}