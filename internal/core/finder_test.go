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

func TestInvalidPatternHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filefusion-pattern-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invalidPatterns := []struct {
		pattern string
		exclude string
	}{
		{pattern: "["},
		{pattern: "*.go", exclude: "["},
		{pattern: "***/invalid"},
		{pattern: "", exclude: "[]"},
	}

	for _, p := range invalidPatterns {
		finder := NewFileFinder(&MixOptions{
			InputPath: tmpDir,
			Pattern:   p.pattern,
			Exclude:   p.exclude,
		})

		_, err := finder.FindFiles()
		if err == nil {
			t.Errorf("Expected error for invalid pattern %q (exclude: %q), got none",
				p.pattern, p.exclude)
		}
	}
}

// TestProcessorConcurrencyAndErrorHandling tests the concurrent processing
// behavior and error handling of the FileProcessor
func TestProcessorConcurrencyAndErrorHandling(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filefusion-concurrent-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple test files with varying permissions
	files := []struct {
		name        string
		content     string
		permissions os.FileMode
		shouldError bool
	}{
		{"readable.txt", "test content", 0644, false},
		{"unreadable.txt", "test content", 0000, true},
		{"executable.txt", "test content", 0755, false},
	}

	var filePaths []string
	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(path, []byte(f.content), f.permissions); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f.name, err)
		}
		filePaths = append(filePaths, path)
	}

	processor := NewFileProcessor(&MixOptions{
		InputPath:   tmpDir,
		MaxFileSize: 1024,
	})

	// Test concurrent processing
	contents, err := processor.ProcessFiles(filePaths)
	if err == nil {
		t.Error("Expected error due to unreadable file, got none")
	}

	// Verify that readable files were still processed
	expectedReadable := 2 // readable.txt and executable.txt
	if len(contents) != expectedReadable {
		t.Errorf("Expected %d readable files, got %d", expectedReadable, len(contents))
	}
}

// TestOutputGeneratorSymlinkHandling tests how the OutputGenerator handles
// symlinks in the output path
func TestOutputGeneratorSymlinkHandling(t *testing.T) {
	if os.Getenv("SKIP_SYMLINK_TESTS") != "" {
		t.Skip("Skipping symlink tests")
	}

	tmpDir, err := os.MkdirTemp("", "filefusion-symlink-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory for output
	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Create a symlink to the output directory
	symlinkPath := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(outputDir, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Create test content
	contents := []FileContent{
		{
			Path:    "test.go",
			Content: "package main\n",
			Size:    12,
		},
	}

	// Test writing through symlink
	outputPath := filepath.Join(symlinkPath, "output.xml")
	generator := NewOutputGenerator(&MixOptions{
		OutputPath:    outputPath,
		MaxOutputSize: 1024,
		OutputType:    OutputTypeXML,
	})

	if err := generator.Generate(contents); err != nil {
		t.Fatalf("Failed to generate output through symlink: %v", err)
	}

	// Verify file exists in actual directory
	realPath := filepath.Join(outputDir, "output.xml")
	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		t.Error("Output file not created in real directory")
	}
}

// TestEdgeCasePaths tests handling of unusual file paths
func TestEdgeCasePaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filefusion-paths-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	edgeCases := []struct {
		path    string
		content string
	}{
		{"file with spaces.go", "content"},
		{"file#with#hashes.go", "content"},
		{"file_with_漢字.go", "content"},
		{"../outside/attempt.go", "content"},
		{"./inside/./path.go", "content"},
	}

	for _, ec := range edgeCases {
		path := filepath.Join(tmpDir, ec.path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			continue // Skip if directory creation fails (e.g., for ../outside)
		}
		if err := os.WriteFile(path, []byte(ec.content), 0644); err != nil {
			continue // Skip if file creation fails
		}
	}

	processor := NewFileProcessor(&MixOptions{
		InputPath:   tmpDir,
		MaxFileSize: 1024,
	})

	paths := []string{
		filepath.Join(tmpDir, "file with spaces.go"),
		filepath.Join(tmpDir, "file#with#hashes.go"),
		filepath.Join(tmpDir, "file_with_漢字.go"),
	}

	contents, err := processor.ProcessFiles(paths)
	if err != nil {
		t.Fatalf("Failed to process edge case paths: %v", err)
	}

	if len(contents) != 3 {
		t.Errorf("Expected 3 processed files, got %d", len(contents))
	}
}
