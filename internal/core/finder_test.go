package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileFinder(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := map[string]string{
		"test1.go":           "package main\n",
		"test2.go":           "package main\n",
		"data.json":          `{"key": "value"}`,
		"config.yaml":        "name: test",
		"subfolder/test.go":  "package sub\n",
		"subfolder/data.txt": "ignored file",
		".hidden/test.go":    "package hidden",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name          string
		pattern       string
		exclude       string
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "find all go files",
			pattern:       "*.go",
			expectedCount: 4,
		},
		{
			name:          "find json and yaml",
			pattern:       "*.json,*.yaml",
			expectedCount: 2,
		},
		{
			name:          "exclude subfolder",
			pattern:       "*.go",
			exclude:       "subfolder/**,.hidden/**",
			expectedCount: 2,
		},
		{
			name:        "no matches",
			pattern:     "*.cpp",
			shouldError: true,
		},
		{
			name:        "invalid pattern",
			pattern:     "[",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finder := NewFileFinder(&MixOptions{
				InputPath: tmpDir,
				Pattern:   tt.pattern,
				Exclude:   tt.exclude,
			})

			files, err := finder.FindFiles()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(files) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(files))
			}
		})
	}
}
