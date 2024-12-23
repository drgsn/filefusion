package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileProcessor(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with specific sizes
	files := map[string]struct {
		size    int64
		content string
	}{
		"small.txt":     {size: 100, content: strings.Repeat("a", 100)},
		"medium.txt":    {size: 1024, content: strings.Repeat("b", 1024)},
		"large.txt":     {size: 2048, content: strings.Repeat("c", 2048)},
		"test/nest.txt": {size: 100, content: strings.Repeat("d", 100)},
	}

	// Create the files
	for name, info := range files {
		path := filepath.Join(tmpDir, name)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for %s: %v", name, err)
		}
		err = os.WriteFile(path, []byte(info.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	tests := []struct {
		name          string
		maxSize       int64
		expectedCount int
	}{
		{
			name:          "process all files",
			maxSize:       3000,
			expectedCount: 4,
		},
		{
			name:          "size limit excludes large file",
			maxSize:       1500,
			expectedCount: 3,
		},
		{
			name:          "small size limit",
			maxSize:       500,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewFileProcessor(&MixOptions{
				InputPath:   tmpDir,
				MaxFileSize: tt.maxSize,
			})

			var paths []string
			for name := range files {
				paths = append(paths, filepath.Join(tmpDir, name))
			}

			contents, err := processor.ProcessFiles(paths)
			if err != nil {
				t.Fatalf("ProcessFiles failed: %v", err)
			}

			if len(contents) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(contents))
			}

			// Verify content integrity
			for _, content := range contents {
				// Convert the content path to be relative to tmpDir for comparison
				relPath, err := filepath.Rel(tmpDir, filepath.Join(tmpDir, content.Path))
				if err != nil {
					t.Fatalf("Failed to get relative path: %v", err)
				}

				expectedInfo, exists := files[relPath]
				if !exists {
					t.Errorf("Unexpected file in results: %s (relative path: %s)", content.Path, relPath)
					continue
				}

				if content.Size != expectedInfo.size {
					t.Errorf("Size mismatch for %s: expected %d, got %d",
						relPath, expectedInfo.size, content.Size)
				}

				if content.Content != expectedInfo.content {
					t.Errorf("Content mismatch for %s", relPath)
				}
			}
		})
	}
}

func TestFileProcessorErrors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filefusion-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a directory that looks like a file
	invalidPath := filepath.Join(tmpDir, "not-a-file.txt")
	if err := os.Mkdir(invalidPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	processor := NewFileProcessor(&MixOptions{
		InputPath:   tmpDir,
		MaxFileSize: 1024,
	})

	_, err = processor.ProcessFiles([]string{invalidPath})
	if err == nil {
		t.Error("Expected error when processing directory as file")
	}
}
