package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/drgsn/filefusion/internal/core/cleaner"
)

type FileContent struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	Extension string `json:"extension"`
	Size      int64  `json:"size"`
}

type OutputType string

const (
	ColorRed   = "\033[1;31m" // bold red
	ColorGreen = "\033[0;32m" // green
	ColorReset = "\033[0m"
)

const (
	OutputTypeXML  OutputType = "XML"
	OutputTypeJSON OutputType = "JSON"
	OutputTypeYAML OutputType = "YAML"
)

type MixOptions struct {
	InputPath      string
	OutputPath     string
	Pattern        string
	Exclude        string
	MaxFileSize    int64
	MaxOutputSize  int64
	OutputType     OutputType
	CleanerOptions *cleaner.CleanerOptions
	IgnoreSymlinks bool
}

func validatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	patterns := strings.Split(pattern, ",")
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Try to match against a test string to verify pattern syntax
		if _, err := filepath.Match(p, "test"); err != nil {
			return fmt.Errorf("syntax error in pattern %q: %w", p, err)
		}
	}
	return nil
}

func validateExcludePatterns(exclude string) error {
	if exclude == "" {
		return nil
	}

	patterns := strings.Split(exclude, ",")
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// Skip validation for glob patterns with **
		if strings.Contains(p, "**") {
			continue
		}

		// Try to match against a test string to verify pattern syntax
		if _, err := filepath.Match(p, "test"); err != nil {
			return fmt.Errorf("invalid exclusion pattern %q: %w", p, err)
		}
	}
	return nil
}

func (m *MixOptions) Validate() error {
	if m.InputPath == "" {
		return &MixError{Message: "input path is required"}
	}
	if m.OutputPath == "" {
		return &MixError{Message: "output path is required"}
	}
	if err := validatePattern(m.Pattern); err != nil {
		return err
	}
	if err := validateExcludePatterns(m.Exclude); err != nil {
		return err
	}
	if m.MaxFileSize <= 0 {
		return &MixError{Message: "max file size must be greater than 0"}
	}
	if m.MaxOutputSize <= 0 {
		return &MixError{Message: "max output size must be greater than 0"}
	}
	switch m.OutputType {
	case OutputTypeXML, OutputTypeJSON, OutputTypeYAML:
	default:
		return &MixError{Message: fmt.Sprintf("unsupported output type: %s", m.OutputType)}
	}
	return nil
}

type MixError struct {
	File    string
	Message string
}

func (e *MixError) Error() string {
	if e.File != "" {
		return "file " + e.File + ": " + e.Message
	}
	return e.Message
}
