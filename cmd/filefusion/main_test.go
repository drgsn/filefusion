package main

import (
	"testing"
)

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