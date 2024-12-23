package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FileFinder handles file discovery operations
type FileFinder struct {
	options *MixOptions
}

// NewFileFinder creates a new FileFinder instance
func NewFileFinder(options *MixOptions) *FileFinder {
	return &FileFinder{options: options}
}

// FindFiles finds all files matching the configured patterns
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
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(f.options.InputPath, path)
		if err != nil {
			relPath = path
		}

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

// matchesPattern checks if a file matches inclusion/exclusion patterns
func (f *FileFinder) matchesPattern(path, filename string) (bool, error) {
	// Check exclusions first
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
				if matched, err := filepath.Match(pattern, pathToCheck); err != nil {
					return false, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
				} else if matched {
					return false, nil
				}
			} else {
				if matched, err := filepath.Match(pattern, filename); err != nil {
					return false, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
				} else if matched {
					return false, nil
				}
			}
		}
	}

	// Check inclusion patterns
	patterns := strings.Split(f.options.Pattern, ",")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

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
