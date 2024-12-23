package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"gopkg.in/yaml.v3"
)

// FileResult represents the result of processing a single file
type FileResult struct {
	Content FileContent
	Error   error
}

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
	// Resolve the input path in case it's a symlink
	resolvedPath, err := filepath.EvalSymlinks(m.options.InputPath)
	if err != nil {
		return fmt.Errorf("error resolving input path: %w", err)
	}
	m.options.InputPath = resolvedPath

	// Find all matching files
	files, err := m.findFiles()
	if err != nil {
		return fmt.Errorf("error finding files: %w", err)
	}

	// Read and process files concurrently
	contents, err := m.readFilesConcurrently(files)
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

	// Use filepath.WalkDir instead of filepath.Walk for better performance
	err := filepath.WalkDir(m.options.InputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// Get file info to properly handle symlinks
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("error getting file info for %s: %w", path, err)
		}

		// Skip directories themselves (but still traverse into them)
		if info.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
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

// readFilesConcurrently reads all files concurrently using worker pool
func (m *Mixer) readFilesConcurrently(paths []string) ([]FileContent, error) {
	numWorkers := min(len(paths), 10) // Limit max number of concurrent workers
	results := make(chan FileResult, len(paths))
	var wg sync.WaitGroup

	// Create a channel for distributing work
	jobs := make(chan string, len(paths))
	for _, path := range paths {
		jobs <- path
	}
	close(jobs)

	// Start worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				result := m.processFile(path)
				results <- result
			}
		}()
	}

	// Wait for all workers to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and handle errors
	var contents []FileContent
	var errors []error

	for result := range results {
		if result.Error != nil {
			errors = append(errors, result.Error)
			continue
		}
		if result.Content.Size > 0 { // Skip empty results (from skipped files)
			contents = append(contents, result.Content)
		}
	}

	// If any errors occurred, return the first one
	if len(errors) > 0 {
		return nil, errors[0]
	}

	return contents, nil
}

// processFile handles reading and processing of a single file
func (m *Mixer) processFile(path string) FileResult {
	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return FileResult{
			Error: &MixError{
				File:    path,
				Message: fmt.Sprintf("error getting file info: %v", err),
			},
		}
	}

	// Skip if file is too large
	if info.Size() > m.options.MaxFileSize {
		fmt.Fprintf(os.Stderr, "Warning: Skipping %s (size %d bytes exceeds limit %d bytes)\n",
			path, info.Size(), m.options.MaxFileSize)
		return FileResult{} // Return empty result for skipped files
	}

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return FileResult{
			Error: &MixError{
				File:    path,
				Message: fmt.Sprintf("error reading file: %v", err),
			},
		}
	}

	// Get the base directory from the input path
	baseDir := filepath.Clean(m.options.InputPath)
	cleanPath := filepath.Clean(path)

	// Convert both paths to slashes for consistent handling
	baseDir = filepath.ToSlash(baseDir)
	cleanPath = filepath.ToSlash(cleanPath)

	relPath := cleanPath
	if strings.HasPrefix(cleanPath, baseDir) {
		relPath = cleanPath[len(baseDir):]
		// Remove leading slash if present
		relPath = strings.TrimPrefix(relPath, "/")
	}

	return FileResult{
		Content: FileContent{
			Path:      relPath,
			Name:      filepath.Base(path),
			Extension: strings.TrimPrefix(filepath.Ext(path), "."),
			Content:   string(content),
			Size:      info.Size(),
		},
	}
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

	switch m.options.OutputType {
	case OutputTypeJSON:
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return &MixError{Message: fmt.Sprintf("error encoding JSON: %v", err)}
		}
		return nil

	case OutputTypeYAML:
		encoder := yaml.NewEncoder(file)
		encoder.SetIndent(2)
		if err := encoder.Encode(output); err != nil {
			return &MixError{Message: fmt.Sprintf("error encoding YAML: %v", err)}
		}
		return nil

	case OutputTypeXML:
		// XML output
		const xmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<documents>{{range $index, $file := .}}
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

	default:
		return &MixError{Message: fmt.Sprintf("unsupported output type: %s", m.options.OutputType)}
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
