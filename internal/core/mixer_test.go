package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestMixerFindFiles(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file structure
	files := map[string]string{
		"test1.go":           "package main\nfunc main() {}\n",
		"test2.go":           "package main\nfunc helper() {}\n",
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
			shouldError:   false,
		},
		{
			name:          "find json and yaml",
			pattern:       "*.json,*.yaml",
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:          "exclude subfolder",
			pattern:       "*.go",
			exclude:       "subfolder/**,.hidden/**", // Updated to exclude both subfolder and .hidden
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:          "exclude hidden folders",
			pattern:       "*.go",
			exclude:       ".hidden/**",
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name:          "exclude specific files",
			pattern:       "*.go,*.json",
			exclude:       "data.json",
			expectedCount: 4,
			shouldError:   false,
		},
		{
			name:          "no matches",
			pattern:       "*.cpp",
			expectedCount: 0,
			shouldError:   true,
		},
		{
			name:          "invalid pattern",
			pattern:       "[",
			expectedCount: 0,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mixer := NewMixer(&MixOptions{
				InputPath: tmpDir,
				Pattern:   tt.pattern,
				Exclude:   tt.exclude,
			})

			files, err := mixer.findFiles()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(files) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d. Files: %v",
					tt.expectedCount, len(files), files)
			}
		})
	}
}

func TestConcurrentFileProcessing(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple test files
	const numFiles = 20
	expectedContents := make(map[string]string)

	for i := 0; i < numFiles; i++ {
		content := fmt.Sprintf("content-%d", i)
		filename := fmt.Sprintf("test%d.txt", i)
		path := filepath.Join(tmpDir, filename)

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		expectedContents[filename] = content
	}

	mixer := NewMixer(&MixOptions{
		InputPath:   tmpDir,
		OutputPath:  filepath.Join(tmpDir, "output.xml"),
		Pattern:     "*.txt",
		MaxFileSize: 1024 * 1024,
	})

	// Find and process files
	files, err := mixer.findFiles()
	if err != nil {
		t.Fatalf("Failed to find files: %v", err)
	}

	// Test concurrent processing
	contents, err := mixer.readFilesConcurrently(files)
	if err != nil {
		t.Fatalf("Failed to read files concurrently: %v", err)
	}

	// Basic validation
	if len(contents) != numFiles {
		t.Errorf("Expected %d files, got %d", numFiles, len(contents))
	}

	// Verify content correctness
	for _, content := range contents {
		expected, exists := expectedContents[content.Name]
		if !exists {
			t.Errorf("Unexpected file in results: %s", content.Name)
			continue
		}
		if content.Content != expected {
			t.Errorf("Content mismatch for %s: expected %q, got %q",
				content.Name, expected, content.Content)
		}
	}

	// Optional performance test (benchmark-style)
	t.Run("PerformanceComparison", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping performance comparison in short mode")
		}

		// Run concurrent version multiple times to get more stable measurements
		var concurrentTimes []time.Duration
		for i := 0; i < 5; i++ {
			start := time.Now()
			contents, err := mixer.readFilesConcurrently(files)
			duration := time.Since(start)
			if err != nil {
				t.Errorf("Concurrent processing failed on iteration %d: %v", i, err)
				continue
			}
			if len(contents) != numFiles {
				t.Errorf("Concurrent processing returned wrong number of files on iteration %d: got %d, want %d",
					i, len(contents), numFiles)
				continue
			}
			concurrentTimes = append(concurrentTimes, duration)
		}

		// Run sequential version multiple times
		var sequentialTimes []time.Duration
		for i := 0; i < 5; i++ {
			start := time.Now()
			var seqContents []FileContent
			for _, file := range files {
				result := mixer.processFile(file)
				if result.Error == nil && result.Content.Size > 0 {
					seqContents = append(seqContents, result.Content)
				}
			}
			duration := time.Since(start)
			if len(seqContents) != numFiles {
				t.Errorf("Sequential processing returned wrong number of files on iteration %d: got %d, want %d",
					i, len(seqContents), numFiles)
				continue
			}
			sequentialTimes = append(sequentialTimes, duration)
		}

		// Calculate median times
		concurrentMedian := calculateMedian(concurrentTimes)
		sequentialMedian := calculateMedian(sequentialTimes)

		// Log the results for analysis
		t.Logf("Concurrent processing median time: %v", concurrentMedian)
		t.Logf("Sequential processing median time: %v", sequentialMedian)
		t.Logf("Concurrent/Sequential ratio: %.2f", float64(concurrentMedian)/float64(sequentialMedian))

		// Only fail if concurrent is extremely slow compared to sequential
		// This is a very conservative check that should only fail in extreme cases
		if numFiles >= 10 && concurrentMedian > sequentialMedian*3 {
			t.Errorf("Concurrent processing was unexpectedly slow: concurrent=%v, sequential=%v, ratio=%.2f",
				concurrentMedian, sequentialMedian, float64(concurrentMedian)/float64(sequentialMedian))
		}
	})
}

// Helper function to calculate median duration
func calculateMedian(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Sort the durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate median
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func TestConcurrentProcessingWithErrors(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some valid files
	validFiles := map[string]string{
		"valid1.txt": "content1",
		"valid2.txt": "content2",
	}

	for name, content := range validFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a directory that looks like a file
	if err := os.Mkdir(filepath.Join(tmpDir, "invalid.txt"), 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a file that exceeds size limit
	largeContent := strings.Repeat("a", 2*1024) // 2KB
	if err := os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(largeContent), 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	mixer := NewMixer(&MixOptions{
		InputPath:   tmpDir,
		OutputPath:  filepath.Join(tmpDir, "output.xml"),
		Pattern:     "*.txt",
		MaxFileSize: 1024, // 1KB limit
	})

	files, err := mixer.findFiles()
	if err != nil {
		t.Fatalf("Failed to find files: %v", err)
	}

	contents, err := mixer.readFilesConcurrently(files)

	// We should get valid contents, with invalid and large files skipped
	if err != nil {
		t.Errorf("Expected successful processing with skipped files, got error: %v", err)
	}

	if len(contents) != len(validFiles) {
		t.Errorf("Expected %d valid files, got %d", len(validFiles), len(contents))
	}

	// Verify valid contents were processed correctly
	for _, content := range contents {
		expected, exists := validFiles[content.Name]
		if !exists {
			t.Errorf("Unexpected file in results: %s", content.Name)
			continue
		}
		if content.Content != expected {
			t.Errorf("Content mismatch for %s: expected %q, got %q",
				content.Name, expected, content.Content)
		}
	}
}

func TestOutputFormats(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := map[string]string{
		"test1.txt": "content1",
		"test2.txt": "content2",
	}

	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name       string
		outputType OutputType
		validate   func(t *testing.T, output []byte)
	}{
		{
			name:       "XML output",
			outputType: OutputTypeXML,
			validate: func(t *testing.T, output []byte) {
				if !bytes.Contains(output, []byte("<documents>")) {
					t.Error("Expected XML output to contain <documents> tag")
				}
				if !bytes.Contains(output, []byte("<document index=\"1\">")) {
					t.Error("Expected XML output to contain indexed document tags")
				}
				if !bytes.Contains(output, []byte("<source>")) {
					t.Error("Expected XML output to contain source tags")
				}
				if !bytes.Contains(output, []byte("<document_content>")) {
					t.Error("Expected XML output to contain document_content tags")
				}
			},
		},
		{
			name:       "JSON output",
			outputType: OutputTypeJSON,
			validate: func(t *testing.T, output []byte) {
				var result struct {
					Documents []struct {
						Index           int    `json:"index"`
						Source          string `json:"source"`
						DocumentContent string `json:"document_content"`
					} `json:"documents"`
				}

				if err := json.Unmarshal(output, &result); err != nil {
					t.Errorf("Failed to parse JSON output: %v", err)
					return
				}

				if len(result.Documents) != 2 {
					t.Errorf("Expected 2 documents in JSON output, got %d", len(result.Documents))
				}

				// Verify document order and content
				for i, doc := range result.Documents {
					if doc.Index != i+1 {
						t.Errorf("Expected document index %d, got %d", i+1, doc.Index)
					}
					expectedContent := fmt.Sprintf("content%d", i+1)
					if doc.DocumentContent != expectedContent {
						t.Errorf("Expected content %q, got %q", expectedContent, doc.DocumentContent)
					}
				}
			},
		},
		{
			name:       "YAML output",
			outputType: OutputTypeYAML,
			validate: func(t *testing.T, output []byte) {
				var result struct {
					Documents []struct {
						Index           int    `yaml:"index"`
						Source          string `yaml:"source"`
						DocumentContent string `yaml:"document_content"`
					} `yaml:"documents"`
				}

				if err := yaml.Unmarshal(output, &result); err != nil {
					t.Errorf("Failed to parse YAML output: %v", err)
					return
				}

				if len(result.Documents) != 2 {
					t.Errorf("Expected 2 documents in YAML output, got %d", len(result.Documents))
				}

				// Verify document order and content
				for i, doc := range result.Documents {
					if doc.Index != i+1 {
						t.Errorf("Expected document index %d, got %d", i+1, doc.Index)
					}
					expectedContent := fmt.Sprintf("content%d", i+1)
					if doc.DocumentContent != expectedContent {
						t.Errorf("Expected content %q, got %q", expectedContent, doc.DocumentContent)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, fmt.Sprintf("output%s", tt.outputType))
			mixer := NewMixer(&MixOptions{
				InputPath:   tmpDir,
				OutputPath:  outputPath,
				Pattern:     "*.txt",
				MaxFileSize: 1024 * 1024,
				OutputType:  tt.outputType,
			})

			if err := mixer.Mix(); err != nil {
				t.Fatalf("Mix failed: %v", err)
			}

			output, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			tt.validate(t, output)
		})
	}
}

func TestMixerSizeLimit(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files of different sizes
	files := map[string]int{
		"small.txt":  512,  // 512 bytes
		"medium.txt": 1024, // 1KB
		"large.txt":  2048, // 2KB
		"huge.txt":   4096, // 4KB
	}

	// Create the test files
	for name, size := range files {
		content := strings.Repeat("a", size)
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	tests := []struct {
		name          string
		maxSize       int64
		expectedFiles int
		expectedNames []string // Add expected file names
	}{
		{
			name:          "all files under limit",
			maxSize:       5 * 1024, // 5KB
			expectedFiles: 4,
			expectedNames: []string{"small.txt", "medium.txt", "large.txt", "huge.txt"},
		},
		{
			name:          "medium size limit",
			maxSize:       2 * 1024, // 2KB
			expectedFiles: 3,
			expectedNames: []string{"small.txt", "medium.txt", "large.txt"},
		},
		{
			name:          "small size limit",
			maxSize:       1024, // 1KB
			expectedFiles: 2,
			expectedNames: []string{"small.txt", "medium.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mixer := NewMixer(&MixOptions{
				InputPath:   tmpDir,
				OutputPath:  filepath.Join(tmpDir, "output.xml"),
				Pattern:     "*.txt",
				MaxFileSize: tt.maxSize,
			})

			// Process files
			contents, err := mixer.readFilesConcurrently([]string{
				filepath.Join(tmpDir, "small.txt"),
				filepath.Join(tmpDir, "medium.txt"),
				filepath.Join(tmpDir, "large.txt"),
				filepath.Join(tmpDir, "huge.txt"),
			})

			if err != nil {
				t.Fatalf("Failed to process files: %v", err)
			}

			if len(contents) != tt.expectedFiles {
				t.Errorf("Expected %d files, got %d", tt.expectedFiles, len(contents))
			}

			// Create a map of expected files for easier checking
			expectedFiles := make(map[string]bool)
			for _, name := range tt.expectedNames {
				expectedFiles[name] = true
			}

			// Verify all included files are within size limit and expected
			for _, content := range contents {
				if content.Size > tt.maxSize {
					t.Errorf("File %s exceeds size limit: %d > %d",
						content.Name, content.Size, tt.maxSize)
				}
				if !expectedFiles[content.Name] {
					t.Errorf("Unexpected file in results: %s", content.Name)
				}
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	// Skip this test on Windows as permission handling is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test scenarios
	tests := []struct {
		name        string
		setup       func(dir string) error
		pattern     string
		shouldError bool
		errorCheck  func(error) bool
	}{
		{
			name: "non-existent directory",
			setup: func(dir string) error {
				return nil // Do nothing, we'll use a non-existent path
			},
			pattern:     "*.txt",
			shouldError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "no such file or directory")
			},
		},
		{
			name: "permission denied",
			setup: func(dir string) error {
				subdir := filepath.Join(dir, "noaccess")
				if err := os.Mkdir(subdir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(subdir, "test.txt"), []byte("test"), 0644); err != nil {
					return err
				}
				// Change permissions after creating the file
				return os.Chmod(subdir, 0000)
			},
			pattern:     "*.txt",
			shouldError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "permission denied") ||
					strings.Contains(err.Error(), "no files found")
			},
		},
		{
			name: "invalid output path",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "test.txt"), []byte("test"), 0644)
			},
			pattern:     "*.txt",
			shouldError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "permission denied") ||
					strings.Contains(err.Error(), "no such file or directory")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test scenario
			if err := tt.setup(tmpDir); err != nil {
				t.Fatalf("Failed to set up test: %v", err)
			}

			var inputPath string
			if tt.name == "non-existent directory" {
				inputPath = filepath.Join(tmpDir, "nonexistent")
			} else {
				inputPath = tmpDir
			}

			mixer := NewMixer(&MixOptions{
				InputPath:   inputPath,
				OutputPath:  "/nonexistent/output.xml", // Invalid output path
				Pattern:     tt.pattern,
				MaxFileSize: 1024 * 1024,
			})

			err := mixer.Mix()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if !tt.errorCheck(err) {
					t.Errorf("Expected specific error condition, got: %v", err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Clean up permissions so the deferred cleanup can work
			if tt.name == "permission denied" {
				os.Chmod(filepath.Join(tmpDir, "noaccess"), 0755)
			}
		})
	}
}
