package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileProcessor handles file processing operations
type FileProcessor struct {
	options *MixOptions
}

// NewFileProcessor creates a new FileProcessor instance
func NewFileProcessor(options *MixOptions) *FileProcessor {
	return &FileProcessor{options: options}
}

// ProcessFiles processes multiple files concurrently
func (p *FileProcessor) ProcessFiles(paths []string) ([]FileContent, error) {
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
				result := p.processFile(path)
				results <- result
			}
		}()
	}

	// Wait for all workers to finish
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
		if result.Content.Size > 0 { // Skip empty results
			contents = append(contents, result.Content)
		}
	}

	// If any errors occurred, return the first one
	if len(errors) > 0 {
		return nil, errors[0]
	}

	return contents, nil
}

// processFile handles processing of a single file
func (p *FileProcessor) processFile(path string) FileResult {
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

	// Check if it's a directory
	if info.IsDir() {
		return FileResult{
			Error: &MixError{
				File:    path,
				Message: "is a directory",
			},
		}
	}

	// Skip if file is too large
	if info.Size() > p.options.MaxFileSize {
		fmt.Fprintf(os.Stderr, "Warning: Skipping %s (size %d bytes exceeds limit %d bytes)\n",
			path, info.Size(), p.options.MaxFileSize)
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
	baseDir := filepath.Clean(p.options.InputPath)
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

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FileResult represents the result of processing a single file
type FileResult struct {
	Content FileContent
	Error   error
}
