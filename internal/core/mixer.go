package core

import (
	"fmt"
	"path/filepath"
)

// Mixer handles the complete file mixing process
type Mixer struct {
	options   *MixOptions
	finder    *FileFinder
	processor *FileProcessor
	generator *OutputGenerator
}

// NewMixer creates a new Mixer instance with the given options
func NewMixer(options *MixOptions) *Mixer {
	m := &Mixer{
		options:   options,
		finder:    NewFileFinder(options),
		processor: NewFileProcessor(options),
		generator: NewOutputGenerator(options),
	}

	// Only set default after validation has been performed
	return m
}

// Mix performs the complete file mixing process
func (m *Mixer) Mix() error {
	// Resolve the input path in case it's a symlink
	resolvedPath, err := filepath.EvalSymlinks(m.options.InputPath)
	if err != nil {
		return fmt.Errorf("error resolving input path: %w", err)
	}
	m.options.InputPath = resolvedPath

	// Find all matching files
	files, err := m.finder.FindFiles()
	if err != nil {
		return fmt.Errorf("error finding files: %w", err)
	}

	// Process files
	contents, err := m.processor.ProcessFiles(files)
	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	// Generate output
	if err := m.generator.Generate(contents); err != nil {
		return fmt.Errorf("error generating output: %w", err)
	}

	return nil
}

// GetFoundFiles returns the list of files that were found and processed
// This is useful for testing and verification purposes
func (m *Mixer) GetFoundFiles() ([]FileContent, error) {
	files, err := m.finder.FindFiles()
	if err != nil {
		return nil, err
	}
	return m.processor.ProcessFiles(files)
}

// ValidateOptions checks if the provided options are valid
func (m *Mixer) ValidateOptions() error {
	if m.options.InputPath == "" {
		return &MixError{Message: "input path is required"}
	}

	if m.options.OutputPath == "" {
		return &MixError{Message: "output path is required"}
	}

	if m.options.Pattern == "" {
		return &MixError{Message: "pattern cannot be empty"}
	}

	if m.options.MaxFileSize <= 0 {
		return &MixError{Message: "max file size must be greater than 0"}
	}

	switch m.options.OutputType {
	case OutputTypeXML, OutputTypeJSON, OutputTypeYAML:
		// Valid output types
	default:
		return &MixError{Message: fmt.Sprintf("unsupported output type: %s", m.options.OutputType)}
	}

	// Set default MaxFileSize only after validation
	if m.options.MaxFileSize == 0 {
		m.options.MaxFileSize = 10 * 1024 * 1024 // Default 10MB
	}

	return nil
}
