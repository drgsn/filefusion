package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestPatternValidator_ValidatePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
		errMsg  string
	}{
		// Basic patterns
		{
			name:    "valid simple pattern",
			pattern: "*.txt",
			wantErr: false,
		},
		{
			name:    "valid directory pattern",
			pattern: "src/**/*.go",
			wantErr: false,
		},

		// Length validation
		{
			name:    "pattern too long",
			pattern: strings.Repeat("a", 1001),
			wantErr: true,
			errMsg:  "pattern too long",
		},

		// Null byte validation
		{
			name:    "pattern with null byte",
			pattern: "test\x00.txt",
			wantErr: true,
			errMsg:  "pattern contains null bytes",
		},

		// Banned patterns
		{
			name:    "directory traversal",
			pattern: "/**/../config.json",
			wantErr: true,
			errMsg:  "contains banned pattern",
		},
		{
			name:    "hidden files",
			pattern: "**/.*/**",
			wantErr: true,
			errMsg:  "contains banned pattern",
		},

		// Brace validation
		{
			name:    "valid brace pattern",
			pattern: "{*.txt,*.md}",
			wantErr: false,
		},
		{
			name:    "valid brace pattern with space",
			pattern: "*{.go, .ts}",
			wantErr: false,
		},
		{
			name:    "unmatched opening brace",
			pattern: "{*.txt",
			wantErr: true,
			errMsg:  "unclosed brace",
		},
		{
			name:    "unmatched closing brace",
			pattern: "*.txt}",
			wantErr: true,
			errMsg:  "unmatched closing brace",
		},

		// Bracket validation
		{
			name:    "valid bracket pattern",
			pattern: "[a-z]*.txt",
			wantErr: false,
		},
		{
			name:    "unmatched opening bracket",
			pattern: "[a-z*.txt",
			wantErr: true,
			errMsg:  "unclosed bracket",
		},
		{
			name:    "unmatched closing bracket",
			pattern: "a-z]*.txt",
			wantErr: true,
			errMsg:  "unmatched closing bracket",
		},

		// Negation patterns
		{
			name:    "valid negation pattern",
			pattern: "!*.tmp",
			wantErr: false,
		},

		// Complex patterns
		{
			name:    "complex valid pattern",
			pattern: "src/**/{test,main}/*_[a-z]*.{go,ts}",
			wantErr: false,
		},
		{
			name:    "complex invalid pattern with traversal",
			pattern: "src/**/../{test,main}/*.go",
			wantErr: true,
			errMsg:  "contains banned pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPatternValidator()
			err := v.ValidatePattern(tt.pattern)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePattern() error = nil, want error containing %q", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePattern() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidatePattern() error = %v, want nil", err)
			}
		})
	}
}

func TestPatternValidator_SplitPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "simple comma-separated patterns",
			pattern: "*.txt,*.md",
			want:    []string{"*.txt", "*.md"},
		},
		{
			name:    "patterns with braces",
			pattern: "{*.txt,*.md}",
			want:    []string{"{*.txt,*.md}"},
		},
		{
			name:    "mixed patterns",
			pattern: "*.txt,{*.go,*.ts},*.md",
			want:    []string{"*.txt", "{*.go,*.ts}", "*.md"},
		},
		{
			name:    "pattern with escaped comma",
			pattern: `test\,file.txt`,
			want:    []string{`test\,file.txt`},
		},
		{
			name:    "multiple patterns with escaped comma",
			pattern: `*.txt,test\,file.txt,*.md`,
			want:    []string{"*.txt", `test\,file.txt`, "*.md"},
		},
		{
			name:    "empty pattern",
			pattern: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPatternValidator()
			got := v.splitPatterns(tt.pattern)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitPatterns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPatternValidator_ExpandPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    []string
		wantErr bool
		errMsg  string
	}{
		// Basic expansions
		{
			name:    "empty pattern",
			pattern: "",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "pattern without braces",
			pattern: "*.txt",
			want:    []string{"*.txt"},
			wantErr: false,
		},
		{
			name:    "simple brace expansion",
			pattern: "{a,b}.txt",
			want:    []string{"a.txt", "b.txt"},
			wantErr: false,
		},

		// Complex expansions
		{
			name:    "multiple brace expansions",
			pattern: "test.{txt,md}",
			want:    []string{"test.txt", "test.md"},
			wantErr: false,
		},
		{
			name:    "nested brace expansion",
			pattern: "{a,b}.txt",
			want:    []string{"a.txt", "b.txt"},
			wantErr: false,
		},
		{
			name:    "empty option in braces",
			pattern: "pre{,fix}*.txt",
			want:    []string{"pre*.txt", "prefix*.txt"},
			wantErr: false,
		},

		// Edge cases and escapes
		{
			name:    "pattern with escaped braces",
			pattern: `\{a,b\}.txt`,
			want:    []string{`\{a,b\}.txt`},
			wantErr: false,
		},
		{
			name:    "pattern with escaped comma",
			pattern: `{a\,b,c}.txt`,
			want:    []string{`a\,b.txt`, "c.txt"},
			wantErr: false,
		},

		// Error cases
		{
			name:    "unmatched brace",
			pattern: "{a,b.txt",
			wantErr: true,
			errMsg:  "unclosed brace",
		},
		{
			name:    "unmatched closing brace",
			pattern: "a,b}.txt",
			wantErr: true,
			errMsg:  "unmatched closing brace",
		},

		// Complex patterns
		{
			name:    "complex pattern with braces",
			pattern: "src/{test,main}/*.{js,ts}",
			want: []string{
				"src/test/*.js",
				"src/test/*.ts",
				"src/main/*.js",
				"src/main/*.ts",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPatternValidator()
			got, err := v.ExpandPattern(tt.pattern)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExpandPattern() error = nil, want error containing %q", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ExpandPattern() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ExpandPattern() error = %v, want nil", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ExpandPattern() got %v patterns, want %v patterns\nGot: %v\nWant: %v",
					len(got), len(tt.want), got, tt.want)
				return
			}

			// Create maps for easier comparison
			gotMap := make(map[string]bool)
			for _, pattern := range got {
				gotMap[pattern] = true
			}
			wantMap := make(map[string]bool)
			for _, pattern := range tt.want {
				wantMap[pattern] = true
			}

			for pattern := range wantMap {
				if !gotMap[pattern] {
					t.Errorf("ExpandPattern() missing pattern %q", pattern)
				}
			}
			for pattern := range gotMap {
				if !wantMap[pattern] {
					t.Errorf("ExpandPattern() unexpected pattern %q", pattern)
				}
			}
		})
	}
}
