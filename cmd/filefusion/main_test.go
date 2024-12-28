package main

import (
	"os"
	"testing"

	"github.com/drgsn/filefusion/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestValidateAndGetConfig(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		exclude       string
		maxFileSize   string
		maxOutputSize string
		outputPath    string
		expectError   bool
	}{
		{
			name:          "Valid config",
			pattern:       "*.go,*.json",
			exclude:       "vendor/*",
			maxFileSize:   "10MB",
			maxOutputSize: "50MB",
			outputPath:    "output.xml",
			expectError:   false,
		},
		{
			name:          "Empty pattern",
			pattern:       "",
			maxFileSize:   "10MB",
			maxOutputSize: "50MB",
			expectError:   true,
		},
		{
			name:          "Invalid max file size",
			pattern:       "*.go",
			maxFileSize:   "invalid",
			maxOutputSize: "50MB",
			expectError:   true,
		},
		{
			name:          "Invalid max output size",
			pattern:       "*.go",
			maxFileSize:   "10MB",
			maxOutputSize: "invalid",
			expectError:   true,
		},
		{
			name:          "Invalid output extension",
			pattern:       "*.go",
			maxFileSize:   "10MB",
			maxOutputSize: "50MB",
			outputPath:    "output.invalid",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test flags
			pattern = tt.pattern
			exclude = tt.exclude
			maxFileSize = tt.maxFileSize
			maxOutputSize = tt.maxOutputSize
			outputPath = tt.outputPath

			config, err := validateAndGetConfig([]string{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
			}
		})
	}
}

func TestValidateAndGetOutputType(t *testing.T) {
	tests := []struct {
		name        string
		outputPath  string
		expectType  core.OutputType
		expectError bool
	}{
		{
			name:        "XML output",
			outputPath:  "output.xml",
			expectType:  core.OutputTypeXML,
			expectError: false,
		},
		{
			name:        "JSON output",
			outputPath:  "output.json",
			expectType:  core.OutputTypeJSON,
			expectError: false,
		},
		{
			name:        "YAML output",
			outputPath:  "output.yaml",
			expectType:  core.OutputTypeYAML,
			expectError: false,
		},
		{
			name:        "YML output",
			outputPath:  "output.yml",
			expectType:  core.OutputTypeYAML,
			expectError: false,
		},
		{
			name:        "Empty path defaults to XML",
			outputPath:  "",
			expectType:  core.OutputTypeXML,
			expectError: false,
		},
		{
			name:        "Invalid extension",
			outputPath:  "output.txt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputType, err := validateAndGetOutputType(tt.outputPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectType, outputType)
			}
		})
	}
}

func TestGetCleanerOptions(t *testing.T) {
	tests := []struct {
		name         string
		cleanEnabled bool
		wantNil      bool
	}{
		{
			name:         "Cleaner disabled",
			cleanEnabled: false,
			wantNil:      true,
		},
		{
			name:         "Cleaner enabled",
			cleanEnabled: true,
			wantNil:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanEnabled = tt.cleanEnabled
			opts := getCleanerOptions()

			if tt.wantNil {
				assert.Nil(t, opts)
			} else {
				assert.NotNil(t, opts)
			}
		})
	}
}

func TestRunMix(t *testing.T) {
	// Save current working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create a test file
	testFile := "test.go"
	if err := os.WriteFile(testFile, []byte("package test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create output directory
	if err := os.MkdirAll("output", 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		args        []string
		pattern     string
		outputPath  string
		dryRun      bool
		expectError bool
	}{
		{
			name:        "Dry run",
			args:        []string{"."},
			pattern:     "*.go",
			dryRun:      true,
			expectError: false,
		},
		{
			name:        "Valid run with output",
			args:        []string{"."},
			pattern:     "*.go",
			outputPath:  "output/output.xml",
			expectError: false,
		},
		{
			name:        "Invalid pattern",
			args:        []string{"."},
			pattern:     "[",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test flags
			pattern = tt.pattern
			outputPath = tt.outputPath
			dryRun = tt.dryRun

			err := runMix(rootCmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.dryRun && tt.outputPath != "" {
					// Verify output file exists and is not empty
					info, err := os.Stat(tt.outputPath)
					assert.NoError(t, err)
					assert.Greater(t, info.Size(), int64(0))
				}
			}
		})
	}
}
