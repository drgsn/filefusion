// Package core provides the core functionality for file finding and pattern matching.
package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
)

// FileFinder handles file pattern matching and collection with support for
// glob patterns, symlinks, and parallel processing.
type FileFinder struct {
	includes       []string        // Glob patterns for files to include
	excludes       []string        // Glob patterns for files to exclude
	followSymlinks bool            // Whether to follow symbolic links
	seenPaths      map[string]bool // Track real paths we've seen to prevent duplicates
	seenLinks      map[string]bool // Track symlinks we've seen for reference
	mu             sync.Mutex      // Protects concurrent access to seen maps
}

// Result represents the outcome of a file finding operation.
// It can contain either a matched file path or an error.
type Result struct {
	Path string // The path of the matched file
	Err  error  // Any error encountered during processing
}

// NewFileFinder creates a new FileFinder with the specified include and exclude patterns.
// The followSymlinks parameter determines whether symbolic links should be followed.
func NewFileFinder(includes, excludes []string, followSymlinks bool) *FileFinder {
	return &FileFinder{
		includes:       includes,
		excludes:       excludes,
		followSymlinks: followSymlinks,
		seenPaths:      make(map[string]bool),
		seenLinks:      make(map[string]bool),
	}
}

// FindMatchingFiles returns all files that match the include patterns and don't match any exclude patterns.
// It processes directories in parallel using a worker pool for improved performance.
// Returns a slice of matched file paths and any error encountered during processing.
func (ff *FileFinder) FindMatchingFiles(basePaths []string) ([]string, error) {
	resultChan := make(chan Result)
	var wg sync.WaitGroup

	// Create a worker pool sized to the number of available CPUs
	numWorkers := runtime.GOMAXPROCS(0)
	pathChan := make(chan string, len(basePaths))

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go ff.worker(pathChan, resultChan, &wg)
	}

	// Feed paths to workers in a separate goroutine
	go func() {
		for _, path := range basePaths {
			pathChan <- path
		}
		close(pathChan)
	}()

	// Close result channel when all workers finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect and deduplicate results
	var matches []string
	seen := make(map[string]bool)
	var firstErr error

	for result := range resultChan {
		if result.Err != nil {
			// Store only the first error but continue processing
			if firstErr == nil {
				firstErr = result.Err
			}
			continue
		}

		// Deduplicate matches
		if !seen[result.Path] {
			matches = append(matches, result.Path)
			seen[result.Path] = true
		}
	}

	if firstErr != nil {
		return matches, fmt.Errorf("errors occurred while finding files: %w", firstErr)
	}

	return matches, nil
}

// processSymlink handles the processing of symbolic links, including cycle detection
// and pattern matching for the linked file.
func (ff *FileFinder) processSymlink(path string, resultChan chan<- Result) error {
	// Resolve the actual file path that the symlink points to
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		// For broken symlinks, just log and continue rather than returning an error
		fmt.Fprintf(os.Stderr, "Warning: Could not resolve symlink %q: %v\n", path, err)
		return nil
	}

	// Use mutex to safely check and update seen paths
	ff.mu.Lock()
	seenBefore := ff.seenPaths[realPath]
	ff.seenPaths[realPath] = true
	ff.seenLinks[path] = true
	ff.mu.Unlock()

	// Skip if we've seen this path before (prevents cycles)
	if seenBefore {
		return nil
	}

	// Get info about the real file
	realInfo, err := os.Stat(realPath)
	if err != nil {
		// Log warning and continue for stat errors
		fmt.Fprintf(os.Stderr, "Warning: Could not stat resolved symlink %q: %v\n", realPath, err)
		return nil
	}

	// For directory symlinks, add the path to be processed
	if realInfo.IsDir() {
		return filepath.WalkDir(realPath, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			return ff.handleEntry(p, d, resultChan)
		})
	}

	// Check if either the symlink or its target matches our patterns
	normalizedPath := filepath.ToSlash(path)
	normalizedRealPath := filepath.ToSlash(realPath)

	includeSymlink, err := ff.shouldIncludeFile(normalizedPath)
	if err != nil {
		return err
	}

	includeTarget, err := ff.shouldIncludeFile(normalizedRealPath)
	if err != nil {
		return err
	}

	if includeSymlink || includeTarget {
		resultChan <- Result{Path: path}
	}
	return nil
}

// processRegularFile handles the processing of regular (non-symlink) files,
// including deduplication and pattern matching.
func (ff *FileFinder) processRegularFile(path string, resultChan chan<- Result) error {
	normalizedPath := filepath.ToSlash(path)

	// Check if we've seen this file before
	ff.mu.Lock()
	seenBefore := ff.seenPaths[path]
	ff.seenPaths[path] = true
	ff.mu.Unlock()

	if seenBefore {
		return nil
	}

	// Check if the file matches our patterns
	include, err := ff.shouldIncludeFile(normalizedPath)
	if err != nil {
		return err
	}

	if include {
		resultChan <- Result{Path: path}
	}
	return nil
}

// handleEntry processes a single filesystem entry, determining its type
// and delegating to the appropriate handler.
func (ff *FileFinder) handleEntry(path string, d fs.DirEntry, resultChan chan<- Result) error {
	info, err := d.Info()
	if err != nil {
		return fmt.Errorf("error getting file info for %q: %w", path, err)
	}

	// Handle symlinks specially if configured to follow them
	if info.Mode()&os.ModeSymlink != 0 {
		ff.mu.Lock()
		ff.seenLinks[path] = true
		ff.mu.Unlock()

		if !ff.followSymlinks {
			return nil
		}
		return ff.processSymlink(path, resultChan)
	}

	// Skip directories as they're handled by WalkDir
	if d.IsDir() {
		return nil
	}

	return ff.processRegularFile(path, resultChan)
}

// worker processes paths from pathChan, walking directories and sending results to resultChan.
// It's designed to run concurrently with other workers.
func (ff *FileFinder) worker(pathChan <-chan string, resultChan chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for basePath := range pathChan {
		// Convert to absolute path for consistent handling
		absPath, err := filepath.Abs(basePath)
		if err != nil {
			resultChan <- Result{Err: fmt.Errorf("error resolving path %q: %w", basePath, err)}
			continue
		}

		// Walk the directory tree
		err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// For broken symlinks and permission errors, log and continue
				if os.IsNotExist(err) || os.IsPermission(err) {
					fmt.Fprintf(os.Stderr, "Warning: Skipping %s: %v\n", path, err)
					return nil
				}
				return err
			}
			return ff.handleEntry(path, d, resultChan)
		})

		if err != nil {
			resultChan <- Result{Err: fmt.Errorf("error walking path %q: %w", basePath, err)}
		}
	}
}

// GetRealPath returns the real filesystem path for a file, resolving any symbolic links.
func (ff *FileFinder) GetRealPath(path string) (string, error) {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("error resolving symlink %q: %w", path, err)
	}

	// On macOS, handle /private prefix
	if runtime.GOOS == "darwin" && strings.HasPrefix(realPath, "/private") {
		realPath = realPath[8:] // Remove "/private" prefix
	}

	return realPath, nil
}

// IsSymlink checks if the given path has been seen as a symbolic link during processing.
func (ff *FileFinder) IsSymlink(path string) (bool, error) {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	return ff.seenLinks[path], nil
}

// matchPattern checks if a file matches a given pattern, handling both basename-only
// and full path patterns appropriately.
func (ff *FileFinder) matchPattern(pattern, path, basename string) (bool, error) {
	// For patterns without path separators, match against basename only
	if !strings.Contains(pattern, "/") {
		return doublestar.Match(pattern, basename)
	}
	// For patterns with path separators, match against full path
	return doublestar.Match(pattern, path)
}

// shouldIncludeFile determines whether a file should be included in the results
// based on the configured include and exclude patterns.
func (ff *FileFinder) shouldIncludeFile(path string) (bool, error) {
	basename := filepath.Base(path)

	// Check exclude patterns first - if any match, exclude the file
	for _, pattern := range ff.excludes {
		matched, err := ff.matchPattern(pattern, path, basename)
		if err != nil {
			return false, fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
		if matched {
			return false, nil
		}
	}

	// If no include patterns specified, include all files not explicitly excluded
	if len(ff.includes) == 0 {
		return true, nil
	}

	// Check if file matches any include pattern
	for _, pattern := range ff.includes {
		matched, err := ff.matchPattern(pattern, path, basename)
		if err != nil {
			return false, fmt.Errorf("invalid include pattern %q: %w", pattern, err)
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}
