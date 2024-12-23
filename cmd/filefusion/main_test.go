package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drgsn/filefusion/internal/core"
)

// Existing tests
func TestParseSize(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int64
		shouldErr bool
	}{
		{
			name:      "bytes",
			input:     "1024B",
			expected:  1024,
			shouldErr: false,
		},
		{
			name:      "kilobytes",
			input:     "1KB",
			expected:  1024,
			shouldErr: false,
		},
		{
			name:      "megabytes",
			input:     "1MB",
			expected:  1024 * 1024,
			shouldErr: false,
		},
		{
			name:      "gigabytes",
			input:     "1GB",
			expected:  1024 * 1024 * 1024,
			shouldErr: false,
		},
		{
			name:      "terabytes",
			input:     "1TB",
			expected:  1024 * 1024 * 1024 * 1024,
			shouldErr: false,
		},
		{
			name:      "with spaces",
			input:     " 5 MB ",
			expected:  5 * 1024 * 1024,
			shouldErr: false,
		},
		{
			name:      "invalid format",
			input:     "1XB",
			shouldErr: true,
		},
		{
			name:      "invalid number",
			input:     "abcMB",
			shouldErr: true,
		},
		{
			name:      "negative number",
			input:     "-1MB",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSize(tt.input)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %d bytes, got %d", tt.expected, result)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{
			name:     "bytes",
			input:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			input:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "megabytes",
			input:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "gigabytes",
			input:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "partial unit",
			input:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSize(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Helper function to capture stdout for testing
func captureOutput(fn func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := fn()

	// Restore stdout and get output
	os.Stdout = old
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String(), err
}

// New tests

func TestRootCommandFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "invalid max size format",
			args:          []string{"--max-size", "10Z"},
			expectedError: "invalid max-size value: invalid size format: must end with B, KB, MB, GB, or TB",
		},
		{
			name:          "negative max size",
			args:          []string{"--max-size", "-5MB", "--pattern", "*.go"},
			expectedError: "invalid max-size value: size must be a positive number",
		},
		{
			name:          "empty pattern",
			args:          []string{"--pattern", ""},
			expectedError: "pattern cannot be empty",
		},
		{
			name:          "invalid output extension",
			args:          []string{"--output", "output.txt", "--pattern", "*.go", "--max-size", "10MB"},
			expectedError: "invalid output file extension: must be .xml, .json, .yaml, or .yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			if err.Error() != tt.expectedError {
				t.Errorf("Expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestScanFiles(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-scan-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with specific sizes
	files := map[string]struct {
		size     int64
		content  string
		expected bool
	}{
		"small.go":     {size: 100, content: strings.Repeat("a", 100), expected: true},
		"medium.go":    {size: 1024, content: strings.Repeat("b", 1024), expected: true},
		"large.go":     {size: 2048, content: strings.Repeat("c", 2048), expected: false},
		"ignored.txt":  {size: 100, content: strings.Repeat("d", 100), expected: false},
		"test/nest.go": {size: 100, content: strings.Repeat("e", 100), expected: true},
		".git/hide.go": {size: 100, content: strings.Repeat("f", 100), expected: false},
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

	// Test scanning with various options
	options := &core.MixOptions{
		InputPath:   tmpDir,
		Pattern:     "*.go",
		MaxFileSize: 1500, // Will exclude large.go
	}

	result, totalSize, err := scanFiles(options)
	if err != nil {
		t.Fatalf("scanFiles failed: %v", err)
	}

	// Verify results
	var foundCount int
	for _, file := range result {
		info := filepath.Base(file.Path)
		fileInfo, exists := files[info]
		if !exists {
			// Check if it's in a subdirectory
			if subInfo, exists := files[file.Path]; exists {
				fileInfo = subInfo
			} else {
				t.Errorf("Unexpected file in results: %s", file.Path)
				continue
			}
		}

		if !fileInfo.expected {
			t.Errorf("File %s should not have been included", file.Path)
		}
		foundCount++
	}

	// Check if all expected files were found
	expectedCount := 0
	for _, info := range files {
		if info.expected {
			expectedCount++
		}
	}

	if foundCount != expectedCount {
		t.Errorf("Expected to find %d files, but found %d", expectedCount, foundCount)
	}

	// Verify total size calculation
	var expectedTotal int64
	for _, info := range files {
		if info.expected && info.size <= options.MaxFileSize {
			expectedTotal += info.size
		}
	}

	if totalSize != expectedTotal {
		t.Errorf("Expected total size %d, got %d", expectedTotal, totalSize)
	}
}

func TestExcludePatterns(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-exclude-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := []string{
		"src/main.go",
		"src/test/test.go",
		"build/output.go",
		"vendor/lib.go",
		"src/generated/auto.go",
		"docs/api.go",
	}

	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(path, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name           string
		excludePattern string
		expectedFiles  []string
	}{
		{
			name:           "exclude build directory",
			excludePattern: "build/**",
			expectedFiles:  []string{"src/main.go", "src/test/test.go", "vendor/lib.go", "src/generated/auto.go", "docs/api.go"},
		},
		{
			name:           "exclude multiple directories",
			excludePattern: "build/**,vendor/**",
			expectedFiles:  []string{"src/main.go", "src/test/test.go", "src/generated/auto.go", "docs/api.go"},
		},
		{
			name:           "exclude by pattern",
			excludePattern: "**/test/**",
			expectedFiles:  []string{"src/main.go", "build/output.go", "vendor/lib.go", "src/generated/auto.go", "docs/api.go"},
		},
		{
			name:           "exclude generated files",
			excludePattern: "**/generated/**,build/**",
			expectedFiles:  []string{"src/main.go", "src/test/test.go", "vendor/lib.go", "docs/api.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &core.MixOptions{
				InputPath:   tmpDir,
				Pattern:     "*.go",
				Exclude:     tt.excludePattern,
				MaxFileSize: 1024 * 1024,
			}

			files, _, err := scanFiles(options)
			if err != nil {
				t.Fatalf("scanFiles failed: %v", err)
			}

			// Convert expected files to map for easier checking
			expected := make(map[string]bool)
			for _, f := range tt.expectedFiles {
				expected[f] = true
			}

			// Check results
			if len(files) != len(tt.expectedFiles) {
				t.Errorf("Expected %d files, got %d", len(tt.expectedFiles), len(files))
			}

			for _, file := range files {
				if !expected[file.Path] {
					t.Errorf("Unexpected file in results: %s", file.Path)
				}
			}
		})
	}
}
