package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/drgsn/filefusion/internal/core/cleaner"
)

// FileResult represents the outcome of processing a single file.
// It can contain either the processed content or an error, but not both.
type FileResult struct {
	Content FileContent // Processed file content and metadata
	Error   error       // Error that occurred during processing, if any
}

// FileProcessor handles the concurrent processing of multiple files,
// including content cleaning when enabled. It manages a pool of cleaners
// for different languages and ensures proper resource cleanup.
type FileProcessor struct {
	options  *MixOptions
	cleaners map[cleaner.Language]*cleaner.Cleaner
	mu       sync.RWMutex
}

// NewFileProcessor creates a new FileProcessor instance with the specified options.
// It initializes the cleaner map if cleaning is enabled but defers actual cleaner
// creation until needed.
func NewFileProcessor(options *MixOptions) *FileProcessor {
	return &FileProcessor{
		options:  options,
		cleaners: make(map[cleaner.Language]*cleaner.Cleaner),
	}
}

// ProcessFiles processes multiple files concurrently using a worker pool pattern.
// It respects file size limits and handles errors gracefully, continuing to process
// files even if some fail.
//
// Parameters:
//   - paths: Slice of file paths to process
//
// Returns:
//   - []FileContent: Slice of successfully processed file contents
//   - error: First error encountered during processing, if any
func (p *FileProcessor) ProcessFiles(paths []string) ([]FileContent, error) {
	// Use reasonable number of workers
	numWorkers := min(len(paths), 10)
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
		if result.Content.Size > 0 {
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

// processFile handles the processing of a single file, including reading,
// cleaning (if enabled), and metadata collection.
func (p *FileProcessor) processFile(path string) FileResult {
	// Get file info and perform initial checks
	info, err := os.Stat(path)
	if err != nil {
		return FileResult{
			Error: &MixError{
				File:    path,
				Message: fmt.Sprintf("error getting file info: %v", err),
			},
		}
	}

	if info.IsDir() {
		return FileResult{
			Error: &MixError{
				File:    path,
				Message: "is a directory",
			},
		}
	}

	// Check size limit
	if info.Size() > p.options.MaxFileSize {
		fmt.Fprintf(os.Stderr, "Warning: Skipping %s (size %d bytes exceeds limit %d bytes)\n",
			path, info.Size(), p.options.MaxFileSize)
		return FileResult{}
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

	// Clean content if enabled and language is supported
	if p.options.CleanerOptions != nil {
		cleaned, err := p.cleanContent(path, content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to clean %s: %v\n", path, err)
			// Continue with original content instead of failing
		} else {
			content = cleaned
		}
	}

	// Create relative path
	relPath, err := p.createRelativePath(path)
	if err != nil {
		relPath = path
	}

	// Return successful result
	return FileResult{
		Content: FileContent{
			Path:      filepath.ToSlash(relPath),
			Name:      filepath.Base(path),
			Extension: strings.TrimPrefix(filepath.Ext(path), "."),
			Content:   string(content),
			Size:      int64(len(content)),
		},
	}
}

// cleanContent attempts to clean the content using the appropriate language cleaner
func (p *FileProcessor) cleanContent(path string, content []byte) ([]byte, error) {
	// Add defer/recover to prevent panics from crashing goroutines
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Recovered from panic in cleanContent for %s: %v\n", path, r)
		}
	}()

	lang := p.detectLanguage(path)
	if lang == "" {
		return content, nil
	}

	c, err := p.getOrCreateCleaner(lang)
	if err != nil {
		return nil, fmt.Errorf("failed to create cleaner: %w", err)
	}

	cleaned, err := c.Clean(content)
	if err != nil {
		return nil, fmt.Errorf("failed to clean content: %w", err)
	}

	return cleaned, nil
}

// getOrCreateCleaner safely gets or creates a cleaner for the given language
func (p *FileProcessor) getOrCreateCleaner(lang cleaner.Language) (*cleaner.Cleaner, error) {
	// Try to get existing cleaner
	p.mu.RLock()
	c, exists := p.cleaners[lang]
	p.mu.RUnlock()

	if exists {
		return c, nil
	}

	// Create new cleaner if needed
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check again in case another goroutine created it
	if c, exists = p.cleaners[lang]; exists {
		return c, nil
	}

	// Create new cleaner
	c, err := cleaner.NewCleaner(lang, p.options.CleanerOptions)
	if err != nil {
		return nil, err
	}

	p.cleaners[lang] = c
	return c, nil
}

// createRelativePath creates a path relative to the input directory
func (p *FileProcessor) createRelativePath(path string) (string, error) {
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

	return relPath, nil
}

// detectLanguage determines the language based on file extension
func (p *FileProcessor) detectLanguage(path string) cleaner.Language {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return cleaner.LangGo
	case ".java":
		return cleaner.LangJava
	case ".py":
		return cleaner.LangPython
	case ".js":
		return cleaner.LangJavaScript
	case ".ts":
		return cleaner.LangTypeScript
	case ".html":
		return cleaner.LangHTML
	case ".css":
		return cleaner.LangCSS
	case ".cpp", ".cc", ".h":
		return cleaner.LangCPP
	case ".cs":
		return cleaner.LangCSharp
	case ".php":
		return cleaner.LangPHP
	case ".rb":
		return cleaner.LangRuby
	case ".sh", ".bash":
		return cleaner.LangBash
	case ".swift":
		return cleaner.LangSwift
	case ".kt":
		return cleaner.LangKotlin
	case ".sql":
		return cleaner.LangSQL
	}
	return ""
}

// min returns the smaller of two integers.
// This helper function is used to limit the number of concurrent workers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
