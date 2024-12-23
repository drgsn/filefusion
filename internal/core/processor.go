package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileResult represents the outcome of processing a single file.
// It can contain either the processed content or an error, but not both.
type FileResult struct {
	Content FileContent // Processed file content and metadata
	Error   error       // Error that occurred during processing, if any
}

// FileProcessor handles the concurrent processing of multiple files,
// reading their contents and collecting metadata while respecting size limits
// and other constraints specified in the options.
type FileProcessor struct {
	options *MixOptions
}

// NewFileProcessor creates a new FileProcessor instance with the specified options.
//
// Parameters:
//   - options: Configuration settings for file processing
//
// Returns:
//   - A new FileProcessor instance
func NewFileProcessor(options *MixOptions) *FileProcessor {
	return &FileProcessor{options: options}
}

// ProcessFiles processes multiple files concurrently using a worker pool pattern.
// It respects file size limits and handles errors gracefully, continuing to process
// files even if some fail.
//
// The function implements a concurrent processing model where:
// - Multiple worker goroutines process files simultaneously
// - Results are collected in order of completion
// - Errors are collected but don't stop the overall processing
//
// Parameters:
//   - paths: Slice of file paths to process
//
// Returns:
//   - []FileContent: Slice of successfully processed file contents
//   - error: First error encountered during processing, if any
func (p *FileProcessor) ProcessFiles(paths []string) ([]FileContent, error) {
	numWorkers := min(len(paths), 10) // Limit concurrent workers
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

	// Wait for all workers to finish and close results channel
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
		if result.Content.Size > 0 && result.Content.Size <= p.options.MaxFileSize {
			contents = append(contents, result.Content)
		}
	}

	// Return the first error encountered, but still return processed files
	var firstError error
	if len(errors) > 0 {
		firstError = errors[0]
	}

	return contents, firstError
}

// processFile handles the processing of a single file, including:
// - Reading file content
// - Collecting metadata
// - Handling paths relative to the input directory
// - Validating file size constraints
//
// Parameters:
//   - path: Path to the file to process
//
// Returns:
//   - FileResult containing either the processed content or an error
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

	// Calculate relative path from input directory
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

	// Return successful result with file content and metadata
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

// min returns the smaller of two integers.
// This helper function is used to limit the number of concurrent workers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
