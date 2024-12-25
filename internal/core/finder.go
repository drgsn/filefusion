package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileFinder is responsible for discovering files that match specified patterns
// while respecting exclusion rules. It handles complex pattern matching including
// glob patterns and directory-specific exclusions.
type FileFinder struct {
	options *MixOptions // Configuration options for file discovery
}

// NewFileFinder creates a new FileFinder instance with the specified options.
//
// Parameters:
//   - options: Configuration settings for file discovery
//
// Returns:
//   - A new FileFinder instance
func NewFileFinder(options *MixOptions) *FileFinder {
	return &FileFinder{options: options}
}

// FindFiles discovers all files in the input directory that match the configured patterns
// while respecting exclusion rules. It uses filepath.WalkDir for efficient directory traversal
// and implements sophisticated pattern matching for both inclusion and exclusion.
//
// The function handles:
// - Multiple inclusion patterns (comma-separated)
// - Multiple exclusion patterns (comma-separated)
// - Special patterns like "**" for recursive matching
// - .git directory exclusion
//
// Returns:
//   - []string: Slice of matched file paths
//   - error: Error if any occurs during file discovery
func (f *FileFinder) FindFiles() ([]string, error) {
	var matches []string

	err := filepath.WalkDir(f.options.InputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("error getting file info for %s: %w", path, err)
		}

		if info.IsDir() {
			// Skip .git directory entirely
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get path relative to input directory for pattern matching
		relPath, err := filepath.Rel(f.options.InputPath, path)
		if err != nil {
			relPath = path // Fallback to full path if relative path fails
		}

		// Check if file matches patterns
		match, err := f.matchesPattern(relPath, filepath.Base(path))
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
			f.options.Pattern, f.options.Exclude, f.options.InputPath)
	}

	return matches, nil
}

// matchesPattern checks if a file matches the inclusion patterns while not matching
// any exclusion patterns. It implements sophisticated pattern matching including:
// - Glob pattern support
// - Directory-specific exclusions using "**"
// - Both filename and full path matching
//
// Parameters:
//   - path: Full relative path to the file
//   - filename: Base name of the file
//
// Returns:
//   - bool: true if the file should be included, false otherwise
//   - error: Error if pattern matching fails
func (f *FileFinder) matchesPattern(path, filename string) (bool, error) {
    // Pattern syntax is already validated at this point
    if f.options.Exclude != "" {
        excludePatterns := strings.Split(f.options.Exclude, ",")
        for _, pattern := range excludePatterns {
            pattern = strings.TrimSpace(pattern)
            if pattern == "" {
                continue
            }
            
            pattern = filepath.FromSlash(pattern)
            pathToCheck := filepath.FromSlash(path)
            
            if strings.Contains(pattern, "**") {
                basePattern := strings.TrimSuffix(pattern, string(filepath.Separator)+"**")
                basePattern = strings.TrimSuffix(basePattern, "**")
                if strings.HasPrefix(pathToCheck, basePattern) {
                    return false, nil
                }
            } else if strings.Contains(pattern, string(filepath.Separator)) {
                matched, _ := filepath.Match(pattern, pathToCheck)
                if matched {
                    return false, nil
                }
            } else {
                matched, _ := filepath.Match(pattern, filename)
                if matched {
                    return false, nil
                }
            }
        }
    }

    patterns := strings.Split(f.options.Pattern, ",")
    for _, pattern := range patterns {
        pattern = strings.TrimSpace(pattern)
        if pattern == "" {
            continue
        }
        
        matched, _ := filepath.Match(pattern, filename)
        if matched {
            return true, nil
        }
    }
    
    return false, nil
}