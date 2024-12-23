package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMixer(t *testing.T) {
	// Skip symlink tests on Windows
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink tests on Windows")
	}

	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-mixer-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file structure
	files := map[string]string{
		"test1.go":           "package main\nfunc main() {}\n",
		"test2.go":           "package test\nfunc helper() {}\n",
		"data.json":          `{"key": "value"}`,
		"config.yaml":        "name: test",
		"subfolder/test.go":  "package sub\n",
		"subfolder/data.txt": "ignored file",
	}

	// Create the files
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

	// Create a symlink for testing symlink handling
	symDir := filepath.Join(tmpDir, "symlink")
	realDir := filepath.Join(tmpDir, "subfolder")
	if err := os.Symlink(realDir, symDir); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	tests := []struct {
		name          string
		options       *MixOptions
		expectedFiles int
		shouldError   bool
	}{
		{
			name: "process go files",
			options: &MixOptions{
				InputPath:     tmpDir,
				OutputPath:    filepath.Join(tmpDir, "output.xml"),
				Pattern:       "*.go",
				MaxFileSize:   1024 * 1024,
				MaxOutputSize: 10 * 1024 * 1024,
				OutputType:    OutputTypeXML,
			},
			expectedFiles: 3,
			shouldError:   false,
		},
		{
			name: "process through symlink",
			options: &MixOptions{
				InputPath:     symDir,
				OutputPath:    filepath.Join(tmpDir, "symlink-output.xml"),
				Pattern:       "*.go",
				MaxFileSize:   1024 * 1024,
				MaxOutputSize: 10 * 1024 * 1024,
				OutputType:    OutputTypeXML,
			},
			expectedFiles: 1,
			shouldError:   false,
		},
		{
			name: "invalid output type",
			options: &MixOptions{
				InputPath:     tmpDir,
				OutputPath:    filepath.Join(tmpDir, "output.xml"),
				Pattern:       "*.go",
				MaxFileSize:   1024 * 1024,
				MaxOutputSize: 10 * 1024 * 1024,
				OutputType:    "invalid",
			},
			shouldError: true,
		},
		{
			name: "empty pattern",
			options: &MixOptions{
				InputPath:     tmpDir,
				OutputPath:    filepath.Join(tmpDir, "output.xml"),
				Pattern:       "",
				MaxFileSize:   1024 * 1024,
				MaxOutputSize: 10 * 1024 * 1024,
				OutputType:    OutputTypeXML,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mixer := NewMixer(tt.options)

			// First validate options
			err := mixer.ValidateOptions()
			if tt.shouldError {
				if err == nil {
					t.Error("Expected validation error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected validation error: %v", err)
			}

			// Then test the mixing process
			err = mixer.Mix()
			if err != nil {
				t.Fatalf("Mix failed: %v", err)
			}

			// Verify output file exists
			if _, err := os.Stat(tt.options.OutputPath); os.IsNotExist(err) {
				t.Error("Output file was not created")
			}

			// Verify found files count
			foundFiles, err := mixer.GetFoundFiles()
			if err != nil {
				t.Fatalf("GetFoundFiles failed: %v", err)
			}

			if len(foundFiles) != tt.expectedFiles {
				t.Errorf("Expected %d files, got %d", tt.expectedFiles, len(foundFiles))
			}
		})
	}
}

func TestMixerValidation(t *testing.T) {
	tests := []struct {
		name       string
		options    *MixOptions
		errMessage string
	}{
		{
			name: "empty input path",
			options: &MixOptions{
				OutputPath:    "output.xml",
				Pattern:       "*.go",
				MaxFileSize:   1024,
				MaxOutputSize: 1024,
				OutputType:    OutputTypeXML,
			},
			errMessage: "input path is required",
		},
		{
			name: "empty output path",
			options: &MixOptions{
				InputPath:     "input",
				Pattern:       "*.go",
				MaxFileSize:   1024,
				MaxOutputSize: 1024,
				OutputType:    OutputTypeXML,
			},
			errMessage: "output path is required",
		},
		{
			name: "empty pattern",
			options: &MixOptions{
				InputPath:     "input",
				OutputPath:    "output.xml",
				MaxFileSize:   1024,
				MaxOutputSize: 1024,
				OutputType:    OutputTypeXML,
			},
			errMessage: "pattern cannot be empty",
		},
		{
			name: "invalid max file size",
			options: &MixOptions{
				InputPath:     "input",
				OutputPath:    "output.xml",
				Pattern:       "*.go",
				MaxFileSize:   0,
				MaxOutputSize: 1024,
				OutputType:    OutputTypeXML,
			},
			errMessage: "max file size must be greater than 0",
		},
		{
			name: "invalid max output size",
			options: &MixOptions{
				InputPath:     "input",
				OutputPath:    "output.xml",
				Pattern:       "*.go",
				MaxFileSize:   1024,
				MaxOutputSize: 0,
				OutputType:    OutputTypeXML,
			},
			errMessage: "max output size must be greater than 0",
		},
		{
			name: "invalid output type",
			options: &MixOptions{
				InputPath:     "input",
				OutputPath:    "output.xml",
				Pattern:       "*.go",
				MaxFileSize:   1024,
				MaxOutputSize: 1024,
				OutputType:    "invalid",
			},
			errMessage: "unsupported output type: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mixer := NewMixer(tt.options)
			err := mixer.ValidateOptions()

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if err.Error() != tt.errMessage {
				t.Errorf("Expected error message %q, got %q", tt.errMessage, err.Error())
			}
		})
	}
}

func TestMixError(t *testing.T) {
	tests := []struct {
		name     string
		err      *MixError
		expected string
	}{
		{
			name: "with file and message",
			err: &MixError{
				File:    "test.go",
				Message: "failed to read",
			},
			expected: "file test.go: failed to read",
		},
		{
			name: "with message only",
			err: &MixError{
				Message: "general error",
			},
			expected: "general error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("MixError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}
