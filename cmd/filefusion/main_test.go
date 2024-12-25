package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drgsn/filefusion/internal/core"
	"github.com/spf13/cobra"
)

func setupRootCmd() {
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
	rootCmd.PersistentFlags().StringVar(&maxFileSize, "max-file-size", "10MB", "maximum size for individual input files")
	rootCmd.PersistentFlags().StringVar(&maxOutputSize, "max-output-size", "50MB", "maximum size for output file")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show the list of files that will be processed")
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
			name:          "invalid max file size format",
			args:          []string{"--max-file-size", "10Z", "some/path"},
			expectedError: "invalid max-file-size value: invalid size format: must end with B, KB, MB, GB, or TB",
		},
		{
			name:          "invalid max output size format",
			args:          []string{"--max-output-size", "10Z", "some/path"},
			expectedError: "invalid max-output-size value: invalid size format: must end with B, KB, MB, GB, or TB",
		},
		{
			name:          "negative max file size",
			args:          []string{"--max-file-size", "-5MB", "--pattern", "*.go", "some/path"},
			expectedError: "invalid max-file-size value: size must be a positive number",
		},
		{
			name:          "negative max output size",
			args:          []string{"--max-output-size", "-5MB", "--pattern", "*.go", "some/path"},
			expectedError: "invalid max-output-size value: size must be a positive number",
		},
		{
			name:          "empty pattern",
			args:          []string{"--pattern", "", "some/path"},
			expectedError: "pattern cannot be empty",
		},
		{
			name:          "invalid output extension",
			args:          []string{"--output", "output.txt", "--pattern", "*.go", "some/path"},
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
		name           string
		outputFlag     string
		expectedStart  string
		notExpected    string
		outputContains string
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
			name:           "YAML output",
			outputFlag:     "output.yaml",
			outputContains: "documents:",
			notExpected:    "xml",
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
				"--max-file-size", "1MB",
				"--max-output-size", "10MB",
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

			if tt.expectedStart != "" {
				if !strings.HasPrefix(string(content), tt.expectedStart) {
					t.Errorf("Output should start with %q", tt.expectedStart)
				}
			}

			if tt.outputContains != "" {
				if !strings.Contains(string(content), tt.outputContains) {
					t.Errorf("Output should contain %q", tt.outputContains)
				}
			}

			if strings.Contains(string(content), tt.notExpected) {
				t.Errorf("Output should not contain %q", tt.notExpected)
			}
		})
	}
}

func TestSizeExceedCase(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filefusion-size-exceed-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := strings.Repeat("x", 2*1024*1024) // 2MB of content
	err = os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	setupRootCmd()
	rootCmd.SetArgs([]string{
		"--max-output-size", "1MB",
		"--pattern", "*.txt",
		tmpDir,
	})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
		t.Errorf("Expected error about exceeding size limit, got: %v", err)
	}
}

func TestSizeLimits(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filefusion-size-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	files := map[string]struct {
		size     int
		name     string
		included bool
	}{
		"small.go":  {size: 500 * 1024, name: "small.go", included: true},
		"medium.go": {size: 5 * 1024 * 1024, name: "medium.go", included: true},
		"large.go":  {size: 15 * 1024 * 1024, name: "large.go", included: false},
	}

	for _, file := range files {
		content := strings.Repeat("x", file.size)
		path := filepath.Join(tmpDir, file.name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name          string
		maxFileSize   string
		maxOutputSize string
		expectedFiles int
		shouldError   bool
		errorContains string
	}{
		{
			name:          "default limits",
			maxFileSize:   "10MB",
			maxOutputSize: "50MB",
			expectedFiles: 2,
			shouldError:   false,
		},
		{
			name:          "small file size limit",
			maxFileSize:   "1MB",
			maxOutputSize: "50MB",
			expectedFiles: 1,
			shouldError:   false,
		},
		{
			name:          "small output size limit",
			maxFileSize:   "10MB",
			maxOutputSize: "1MB",
			expectedFiles: 2,
			shouldError:   true,
			errorContains: "exceeds maximum allowed size",
		},
		{
			name:          "invalid max file size",
			maxFileSize:   "invalid",
			maxOutputSize: "50MB",
			shouldError:   true,
			errorContains: "invalid max-file-size value",
		},
		{
			name:          "invalid max output size",
			maxFileSize:   "10MB",
			maxOutputSize: "invalid",
			shouldError:   true,
			errorContains: "invalid max-output-size value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "output.xml")
			setupRootCmd()
			args := []string{
				"--output", outputPath,
				"--max-file-size", tt.maxFileSize,
				"--max-output-size", tt.maxOutputSize,
				tmpDir,
			}
			rootCmd.SetArgs(args)
			err := rootCmd.Execute()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			docCount := strings.Count(string(content), "<document ")
			if docCount != tt.expectedFiles {
				t.Errorf("Expected %d files in output, got %d", tt.expectedFiles, docCount)
			}
		})
	}
}

func TestDryRun(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{"test1.go", "package main"},
		{"test2.go", "package test"},
		{"ignore.txt", "ignored file"},
	}

	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, tf.name)
		if err := os.WriteFile(path, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Set up the output file path
	outputPath := filepath.Join(tmpDir, "output.xml")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set up command line arguments
	setupRootCmd()
	rootCmd.SetArgs([]string{
		"--pattern", "*.go",
		"--output", outputPath,
		"--dry-run=true",
		tmpDir,
	})

	// Run the command
	err = rootCmd.Execute()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify results
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check that output file was not created
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Error("Output file should not exist in dry-run mode")
	}

	// Verify output contains expected information
	expectedStrings := []string{
		"Found 2 files matching pattern",
		fmt.Sprintf("Processing %s:", tmpDir),
		"test1.go",
		"test2.go",
		"Matched files:",
		"Dry run complete",
	}

	for _, exp := range expectedStrings {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got output:\n%s", exp, output)
		}
	}

	// Instead of checking exact file sizes, verify size information is present
	for _, tf := range testFiles {
		if strings.HasSuffix(tf.name, ".go") {
			if !strings.Contains(output, fmt.Sprintf("- %s (", tf.name)) {
				t.Errorf("Expected output to contain file '%s' with size, got output:\n%s", tf.name, output)
			}
		}
	}

	// Verify ignored file is not mentioned
	if strings.Contains(output, "ignore.txt") {
		t.Error("Output should not contain ignored file")
	}
}
