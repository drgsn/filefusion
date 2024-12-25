package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMixOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		options MixOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid options",
			options: MixOptions{
				InputPath:     "/input",
				OutputPath:    "/output",
				Pattern:       "*.txt",
				MaxFileSize:   1024,
				MaxOutputSize: 2048,
				OutputType:    OutputTypeJSON,
			},
			wantErr: false,
		},
		{
			name: "empty input path",
			options: MixOptions{
				OutputPath:    "/output",
				Pattern:       "*.txt",
				MaxFileSize:   1024,
				MaxOutputSize: 2048,
				OutputType:    OutputTypeJSON,
			},
			wantErr: true,
			errMsg:  "input path is required",
		},
		{
			name: "empty output path",
			options: MixOptions{
				InputPath:     "/input",
				Pattern:       "*.txt",
				MaxFileSize:   1024,
				MaxOutputSize: 2048,
				OutputType:    OutputTypeJSON,
			},
			wantErr: true,
			errMsg:  "output path is required",
		},
		{
			name: "invalid pattern",
			options: MixOptions{
				InputPath:     "/input",
				OutputPath:    "/output",
				Pattern:       "[", // Invalid pattern
				MaxFileSize:   1024,
				MaxOutputSize: 2048,
				OutputType:    OutputTypeJSON,
			},
			wantErr: true,
			errMsg:  `syntax error in pattern "["`,
		},
		{
			name: "invalid exclude pattern",
			options: MixOptions{
				InputPath:     "/input",
				OutputPath:    "/output",
				Pattern:       "*.txt",
				Exclude:       "[", // Invalid pattern
				MaxFileSize:   1024,
				MaxOutputSize: 2048,
				OutputType:    OutputTypeJSON,
			},
			wantErr: true,
			errMsg:  `invalid exclusion pattern "["`,
		},
		{
			name: "zero max file size",
			options: MixOptions{
				InputPath:     "/input",
				OutputPath:    "/output",
				Pattern:       "*.txt",
				MaxFileSize:   0,
				MaxOutputSize: 2048,
				OutputType:    OutputTypeJSON,
			},
			wantErr: true,
			errMsg:  "max file size must be greater than 0",
		},
		{
			name: "zero max output size",
			options: MixOptions{
				InputPath:     "/input",
				OutputPath:    "/output",
				Pattern:       "*.txt",
				MaxFileSize:   1024,
				MaxOutputSize: 0,
				OutputType:    OutputTypeJSON,
			},
			wantErr: true,
			errMsg:  "max output size must be greater than 0",
		},
		{
			name: "unsupported output type",
			options: MixOptions{
				InputPath:     "/input",
				OutputPath:    "/output",
				Pattern:       "*.txt",
				MaxFileSize:   1024,
				MaxOutputSize: 2048,
				OutputType:    "INVALID",
			},
			wantErr: true,
			errMsg:  "unsupported output type: INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{
			name:    "valid single pattern",
			pattern: "*.txt",
			wantErr: false,
		},
		{
			name:    "valid multiple patterns",
			pattern: "*.txt,*.go,*.md",
			wantErr: false,
		},
		{
			name:    "empty pattern",
			pattern: "",
			wantErr: true,
		},
		{
			name:    "invalid pattern",
			pattern: "[",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePattern(tt.pattern)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateExcludePatterns(t *testing.T) {
	tests := []struct {
		name    string
		exclude string
		wantErr bool
	}{
		{
			name:    "empty exclude",
			exclude: "",
			wantErr: false,
		},
		{
			name:    "valid single pattern",
			exclude: "*.tmp",
			wantErr: false,
		},
		{
			name:    "valid multiple patterns",
			exclude: "*.tmp,*.log,*.bak",
			wantErr: false,
		},
		{
			name:    "glob pattern with **",
			exclude: "**/vendor/**",
			wantErr: false,
		},
		{
			name:    "invalid pattern",
			exclude: "[",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExcludePatterns(tt.exclude)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMixErrorString(t *testing.T) {
	tests := []struct {
		name     string
		err      MixError
		expected string
	}{
		{
			name: "error with file",
			err: MixError{
				File:    "test.txt",
				Message: "file not found",
			},
			expected: "file test.txt: file not found",
		},
		{
			name: "error without file",
			err: MixError{
				Message: "invalid operation",
			},
			expected: "invalid operation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}
