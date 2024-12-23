package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drgsn/filefusion/internal/core"
	"github.com/spf13/cobra"
)

func setupRootCmd() {
	// Reset the command and its flags completely
	rootCmd = &cobra.Command{
		Use:   "filefusion [paths...]",
		Short: "Filefusion - File concatenation tool optimized for LLM usage",
		Long: `Filefusion concatenates files into a format optimized for Large Language Models (LLMs).
It preserves file metadata and structures the output in an XML-like or JSON format.`,
		RunE: runMix,
	}

	// Re-initialize all flags
	rootCmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "", "output file path")
	rootCmd.PersistentFlags().StringVarP(&pattern, "pattern", "p", "*.go,*.json,*.yaml,*.yml", "file patterns")
	rootCmd.PersistentFlags().StringVarP(&exclude, "exclude", "e", "", "exclude patterns")
	rootCmd.PersistentFlags().StringVar(&maxFileSize, "max-size", "10MB", "maximum size per file")
}

func TestDeriveOutputPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "directory path",
			input:    "/service/user-service",
			expected: "user-service.xml",
		},
		{
			name:     "directory path with trailing slash",
			input:    "/service/user-service/",
			expected: "user-service.xml",
		},
		{
			name:     "file path",
			input:    "/service/user-service/openapi.json",
			expected: "openapi.json.xml",
		},
		{
			name:     "simple directory name",
			input:    "config",
			expected: "config.xml",
		},
		{
			name:     "simple file name",
			input:    "config.yaml",
			expected: "config.yaml.xml",
		},
		{
			name:     "complex path with file",
			input:    "/very/long/path/to/some/config.json",
			expected: "config.json.xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveOutputPath(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMultipleInputPaths(t *testing.T) {
	// Create temporary test directories
	tmpDir1, err := os.MkdirTemp("", "filefusion-test1-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "filefusion-test2-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tmpDir2)

	// Create test files in first directory
	files1 := map[string]string{
		"test1.go":  "package main\nfunc main() {}\n",
		"data.json": `{"key": "value"}`,
	}

	// Create test files in second directory
	files2 := map[string]string{
		"test2.go":    "package other\nfunc helper() {}\n",
		"config.yaml": "name: test",
		"ignore.txt":  "ignored file",
	}

	// Create files in first directory
	for name, content := range files1 {
		path := filepath.Join(tmpDir1, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create files in second directory
	for name, content := range files2 {
		path := filepath.Join(tmpDir2, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		expectedFiles []string
		shouldError   bool
		errorContains string
	}{
		{
			name:          "multiple directories without output flag",
			args:          []string{tmpDir1, tmpDir2},
			flags:         map[string]string{"pattern": "*.go,*.json,*.yaml"},
			expectedFiles: []string{filepath.Base(tmpDir1) + ".xml", filepath.Base(tmpDir2) + ".xml"},
			shouldError:   false,
		},
		{
			name:          "multiple directories with output flag",
			args:          []string{tmpDir1, tmpDir2},
			flags:         map[string]string{"pattern": "*.go,*.json,*.yaml", "output": "combined.json"},
			expectedFiles: []string{"combined.json"},
			shouldError:   false,
		},
		{
			name:          "invalid pattern",
			args:          []string{tmpDir1, tmpDir2},
			flags:         map[string]string{"pattern": "["},
			shouldError:   true,
			errorContains: "syntax error in pattern",
		},
		{
			name:          "no matching files",
			args:          []string{tmpDir1, tmpDir2},
			flags:         map[string]string{"pattern": "*.cpp"},
			shouldError:   true,
			errorContains: "no files found matching pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current working directory
			wd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get working directory: %v", err)
			}

			// Create temporary directory for output files
			outputDir, err := os.MkdirTemp("", "filefusion-output-*")
			if err != nil {
				t.Fatalf("Failed to create output directory: %v", err)
			}
			defer os.RemoveAll(outputDir)

			// Change to output directory
			if err := os.Chdir(outputDir); err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Ensure we change back to the original directory
			defer func() {
				if err := os.Chdir(wd); err != nil {
					t.Errorf("Failed to change back to original directory: %v", err)
				}
			}()

			// Reset and reinitialize command for each test
			setupRootCmd()

			// Build command arguments
			var cmdArgs []string

			// Add flags first
			for flag, value := range tt.flags {
				cmdArgs = append(cmdArgs, "--"+flag, value)
			}

			// Add positional arguments
			cmdArgs = append(cmdArgs, tt.args...)

			// Set the command args
			rootCmd.SetArgs(cmdArgs)

			// Execute command
			err = rootCmd.Execute()

			// Check error status
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check output files
			files, err := os.ReadDir(outputDir)
			if err != nil {
				t.Fatalf("Failed to read output directory: %v", err)
			}

			// Create a map of expected files
			expectedFiles := make(map[string]bool)
			for _, f := range tt.expectedFiles {
				expectedFiles[f] = true
			}

			// Check that all expected files exist
			for _, file := range files {
				if !expectedFiles[file.Name()] {
					t.Errorf("Unexpected file created: %s", file.Name())
				}
				delete(expectedFiles, file.Name())
			}

			// Check if any expected files are missing
			for f := range expectedFiles {
				t.Errorf("Expected file not created: %s", f)
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

	tests := []struct {
		name          string
		pattern       string
		exclude       string
		maxSize       int64
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "standard scan",
			pattern:       "*.go",
			maxSize:       1500,
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name:          "exclude nested",
			pattern:       "*.go",
			exclude:       "test/**",
			maxSize:       1500,
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:          "small size limit",
			pattern:       "*.go",
			maxSize:       500,
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:          "multiple patterns",
			pattern:       "*.go,*.txt",
			maxSize:       1500,
			expectedCount: 4,
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &core.MixOptions{
				InputPath:   tmpDir,
				Pattern:     tt.pattern,
				Exclude:     tt.exclude,
				MaxFileSize: tt.maxSize,
			}

			files, totalSize, err := scanFiles(options)
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
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(files))
			}

			// Verify total size calculation
			var expectedTotal int64
			for _, file := range files {
				expectedTotal += file.Size
			}

			if totalSize != expectedTotal {
				t.Errorf("Expected total size %d, got %d", expectedTotal, totalSize)
			}
		})
	}
}

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

func TestRootCommandFlags(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "invalid max size format",
			args:          []string{"--max-size", "10Z", "some/path"},
			expectedError: "invalid max-size value: invalid size format: must end with B, KB, MB, GB, or TB",
		},
		{
			name:          "negative max size",
			args:          []string{"--max-size", "-5MB", "--pattern", "*.go", "some/path"},
			expectedError: "invalid max-size value: size must be a positive number",
		},
		{
			name:          "empty pattern",
			args:          []string{"--pattern", "", "some/path"},
			expectedError: "pattern cannot be empty",
		},
		{
			name:          "invalid output extension",
			args:          []string{"--output", "output.txt", "--pattern", "*.go", "--max-size", "10MB", "some/path"},
			expectedError: "invalid output file extension: must be .xml, .json, .yaml, or .yml",
		},
		{
			name:          "multiple inputs with invalid output",
			args:          []string{"--output", "out.txt", "path1", "path2"},
			expectedError: "invalid output file extension: must be .xml, .json, .yaml, or .yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupRootCmd()
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

func TestRootCommandWithNoArgs(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get and save the current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to the temporary directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Ensure we change back to the original directory
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to change back to original directory: %v", err)
		}
	}()

	// Reset and setup the root command
	setupRootCmd()
	rootCmd.SetArgs([]string{})

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("Command execution failed: %v", err)
	}

	// Get the base name of the directory for the expected output file
	baseName := filepath.Base(tmpDir)
	expectedOutput := filepath.Join(tmpDir, baseName+".xml")

	// Check if output file exists and has content
	fileInfo, err := os.Stat(expectedOutput)
	if os.IsNotExist(err) {
		t.Errorf("Expected output file %s was not created", expectedOutput)
	} else if err != nil {
		t.Errorf("Error checking output file: %v", err)
	} else if fileInfo.Size() == 0 {
		t.Error("Output file was created but is empty")
	}

	// Verify the content of the output file
	content, err := os.ReadFile(expectedOutput)
	if err != nil {
		t.Errorf("Failed to read output file: %v", err)
	} else if !bytes.Contains(content, []byte("package main")) {
		t.Error("Output file doesn't contain expected content")
	}
}

func TestOutputContents(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-output-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"test.go":   "package main\nfunc main() {}\n",
		"data.json": `{"key": "value"}`,
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name          string
		outputFlag    string
		expectedStart string
		notExpected   string
	}{
		{
			name:          "XML output",
			outputFlag:    "output.xml",
			expectedStart: "<?xml",
			notExpected:   "yaml",
		},
		{
			name:          "JSON output",
			outputFlag:    "output.json",
			expectedStart: "{",
			notExpected:   "xml",
		},
		{
			name:          "YAML output",
			outputFlag:    "output.yaml",
			expectedStart: "documents:",
			notExpected:   "xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, tt.outputFlag)

			// Reset and set up command
			setupRootCmd()

			rootCmd.SetArgs([]string{
				"--output", outputPath,
				"--pattern", "*.go,*.json",
				tmpDir,
			})

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			// Read and check output file
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			// Check content starts with expected string
			if !strings.Contains(string(content), tt.expectedStart) {
				t.Errorf("Output should contain %q", tt.expectedStart)
			}

			// Check content doesn't contain unexpected string
			if strings.Contains(string(content), tt.notExpected) {
				t.Errorf("Output should not contain %q", tt.notExpected)
			}
		})
	}
}

func TestSizeExceedCase(t *testing.T) {
	// Create temp dir with files exceeding max size
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with specific sizes
	files := map[string]struct {
		name    string
		content string
	}{
		"test1.go": {
			name:    "test1.go",
			content: strings.Repeat("a", 1024*1024), // 1MB
		},
		"test2.go": {
			name:    "test2.go",
			content: strings.Repeat("b", 1024*1024), // 1MB
		},
	}

	// Create the files
	for _, file := range files {
		path := filepath.Join(tmpDir, file.name)
		if err := os.WriteFile(path, []byte(file.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Set up command
	setupRootCmd()
	rootCmd.SetArgs([]string{
		"--max-size", "1MB", // Set max size to 1MB
		"--pattern", "*.go",
		tmpDir,
	})

	// Execute command
	err = rootCmd.Execute()

	// Verify we get an error about exceeding size
	if err == nil {
		t.Fatal("Expected error for exceeding max size")
	}

	// Check that the error message contains the expected text
	if !strings.Contains(err.Error(), "total size exceeds maximum allowed size") {
		t.Errorf("Expected error about exceeding size limit, got: %v", err)
	}
}
