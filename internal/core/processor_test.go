package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/drgsn/filefusion/internal/core/cleaner"
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
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-error-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a nested directory structure
	nestedDir := filepath.Join(tmpDir, "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// Create a test file in the nested directory
	testFile := filepath.Join(nestedDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create processor with minimal size to force errors
	processor := NewFileProcessor(&MixOptions{
		InputPath:   tmpDir,
		MaxFileSize: 1, // 1 byte max size to force size-related errors
	})

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "process directory as file",
			path:        nestedDir,
			wantErr:     true,
			errContains: "is a directory",
		},
		{
			name:        "process non-existent file",
			path:        filepath.Join(tmpDir, "nonexistent.txt"),
			wantErr:     true,
			errContains: "no such file",
		},
		{
			name:    "process file exceeding size limit",
			path:    testFile,
			wantErr: false, // This should not error as we handle large files gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.processFile(tt.path)

			if tt.wantErr {
				if result.Error == nil {
					t.Errorf("Expected error for %s", tt.name)
					return
				}
				if tt.errContains != "" && !strings.Contains(strings.ToLower(result.Error.Error()), strings.ToLower(tt.errContains)) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, result.Error)
				}
			} else {
				if result.Error != nil {
					t.Errorf("Unexpected error: %v", result.Error)
				}
			}
		})
	}
}

func TestProcessorConcurrency(t *testing.T) {
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

		// Set file permissions after creation
		if err := os.Chmod(path, f.permissions); err != nil {
			t.Fatalf("Failed to set permissions for %s: %v", f.name, err)
		}
	}

	processor := NewFileProcessor(&MixOptions{
		InputPath:   tmpDir,
		MaxFileSize: 1024 * 1024, // Increase size limit to 1MB to ensure files are processed
	})

	// Test concurrent processing
	contents, err := processor.ProcessFiles(filePaths)

	// We should get an error because one file is unreadable
	if err == nil {
		t.Error("Expected error due to unreadable file, got none")
	}

	// Verify that readable files were still processed
	// Count only files that were successfully processed
	readableCount := 0
	for _, content := range contents {
		if !strings.Contains(content.Path, "unreadable") {
			readableCount++
		}
	}

	expectedReadable := 2 // readable.txt and executable.txt
	if readableCount != expectedReadable {
		t.Errorf("Expected %d readable files, got %d", expectedReadable, readableCount)
	}
}

func TestProcessorEdgeCases(t *testing.T) {
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

func TestLanguageDetection(t *testing.T) {
	processor := NewFileProcessor(&MixOptions{})

	tests := []struct {
		path     string
		expected cleaner.Language
	}{
		{"test.go", cleaner.LangGo},
		{"test.java", cleaner.LangJava},
		{"test.py", cleaner.LangPython},
		{"test.js", cleaner.LangJavaScript},
		{"test.ts", cleaner.LangTypeScript},
		{"test.html", cleaner.LangHTML},
		{"test.css", cleaner.LangCSS},
		{"test.cpp", cleaner.LangCPP},
		{"test.cc", cleaner.LangCPP},
		{"test.h", cleaner.LangCPP},
		{"test.cs", cleaner.LangCSharp},
		{"test.php", cleaner.LangPHP},
		{"test.rb", cleaner.LangRuby},
		{"test.sh", cleaner.LangBash},
		{"test.bash", cleaner.LangBash},
		{"test.swift", cleaner.LangSwift},
		{"test.kt", cleaner.LangKotlin},
		{"test.sql", cleaner.LangSQL},
		{"test.txt", ""},                // Unsupported extension
		{"test", ""},                    // No extension
		{"test.JAVA", cleaner.LangJava}, // Test case insensitivity
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := processor.detectLanguage(tt.path)
			if got != tt.expected {
				t.Errorf("detectLanguage(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestCleanerCaching(t *testing.T) {
	processor := NewFileProcessor(&MixOptions{
		CleanerOptions: &cleaner.CleanerOptions{},
	})

	// First call should create a new cleaner
	c1, err := processor.getOrCreateCleaner(cleaner.LangGo)
	if err != nil {
		t.Fatalf("Failed to create first cleaner: %v", err)
	}

	// Second call should return the same cleaner
	c2, err := processor.getOrCreateCleaner(cleaner.LangGo)
	if err != nil {
		t.Fatalf("Failed to get cached cleaner: %v", err)
	}

	if c1 != c2 {
		t.Error("Expected same cleaner instance to be returned from cache")
	}

	// Test concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := processor.getOrCreateCleaner(cleaner.LangGo)
			if err != nil {
				t.Errorf("Concurrent cleaner access failed: %v", err)
			}
			if c != c1 {
				t.Error("Got different cleaner instance in concurrent access")
			}
		}()
	}
	wg.Wait()
}

func TestWorkerPoolBehavior(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filefusion-workers-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	fileCount := 20
	var paths []string
	for i := 0; i < fileCount; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		paths = append(paths, path)
	}

	processor := NewFileProcessor(&MixOptions{
		InputPath:   tmpDir,
		MaxFileSize: 1024,
	})

	start := time.Now()
	contents, err := processor.ProcessFiles(paths)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ProcessFiles failed: %v", err)
	}

	if len(contents) != fileCount {
		t.Errorf("Expected %d processed files, got %d", fileCount, len(contents))
	}

	// Verify that processing was actually concurrent
	// If it wasn't concurrent, it would take significantly longer
	// This is a rough check and might need adjustment based on system performance
	expectedMaxDuration := time.Second // Adjust based on reasonable expectations
	if duration > expectedMaxDuration {
		t.Errorf("Processing took too long (%v), suggesting lack of concurrency", duration)
	}
}

func TestRelativePathCreation(t *testing.T) {
	tests := []struct {
		name         string
		inputPath    string
		filePath     string
		expectedPath string
		expectError  bool
	}{
		{
			name:         "simple relative path",
			inputPath:    "/base/dir",
			filePath:     "/base/dir/file.txt",
			expectedPath: "file.txt",
		},
		{
			name:         "nested relative path",
			inputPath:    "/base/dir",
			filePath:     "/base/dir/nested/file.txt",
			expectedPath: "nested/file.txt",
		},
		{
			name:         "path with dots",
			inputPath:    "/base/dir",
			filePath:     "/base/dir/./nested/../file.txt",
			expectedPath: "file.txt",
		},
		{
			name:         "path outside input directory",
			inputPath:    "/base/dir",
			filePath:     "/other/dir/file.txt",
			expectedPath: "/other/dir/file.txt",
		},
		{
			name:         "input path with trailing slash",
			inputPath:    "/base/dir/",
			filePath:     "/base/dir/file.txt",
			expectedPath: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewFileProcessor(&MixOptions{
				InputPath: tt.inputPath,
			})

			got, err := processor.createRelativePath(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if got != tt.expectedPath {
				t.Errorf("createRelativePath() = %v, want %v", got, tt.expectedPath)
			}
		})
	}
}
