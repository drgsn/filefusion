package core

import (
	"fmt"
	"path/filepath"
)

// Mixer orchestrates the complete file processing pipeline, coordinating between
// file finding, processing, and output generation components. It implements the main
// business logic for combining multiple files into a single output document.
type Mixer struct {

	// options contains the configuration settings for the mixing process
	options *MixOptions

	// finder handles file discovery based on patterns and exclusions
	finder *FileFinder

	// processor handles reading and processing of individual files
	processor *FileProcessor

	// generator manages output file creation in various formats
	generator *OutputGenerator
}

// NewMixer creates a new Mixer instance with the provided options.
// It initializes all necessary components (finder, processor, and generator)
// but does not validate the options until Mix is called.
//
// Parameters:
//   - options: Configuration settings for the mixing process
//
// Returns:
//   - A new Mixer instance ready for use
func NewMixer(options *MixOptions) *Mixer {
	m := &Mixer{
		options:   options,
		finder:    NewFileFinder(options),
		processor: NewFileProcessor(options),
		generator: NewOutputGenerator(options),
	}
	return m
}

// Mix performs the complete file mixing process in three main steps:
// 1. Finds all files matching the configured patterns
// 2. Processes each file to extract its content
// 3. Generates the final output file in the specified format
//
// The function handles path resolution, including symlinks, and ensures
// all operations are performed safely and in the correct order.
//
// Returns:
//   - error: nil if successful, otherwise an error describing what went wrong
func (m *Mixer) Mix() error {

	// Resolve any symlinks in the input path for consistent handling
	resolvedPath, err := filepath.EvalSymlinks(m.options.InputPath)
	if err != nil {
		return fmt.Errorf("error resolving input path: %w", err)
	}
	m.options.InputPath = resolvedPath

	// Find all files matching the configured patterns
	files, err := m.finder.FindFiles()
	if err != nil {
		return fmt.Errorf("error finding files: %w", err)
	}

	// Process discovered files to extract their content
	contents, err := m.processor.ProcessFiles(files)
	if err != nil {
		return fmt.Errorf("error processing files: %w", err)
	}

	// Generate the final output file
	if err := m.generator.Generate(contents); err != nil {
		return fmt.Errorf("error generating output: %w", err)
	}

	return nil
}

// GetFoundFiles returns the list of files that were found and processed.
// This method is primarily used for testing and verification purposes,
// allowing inspection of which files were discovered without generating output.
//
// Returns:
//   - []FileContent: Slice of processed file contents
//   - error: nil if successful, otherwise an error describing what went wrong
func (m *Mixer) GetFoundFiles() ([]FileContent, error) {
	files, err := m.finder.FindFiles()
	if err != nil {
		return nil, err
	}
	return m.processor.ProcessFiles(files)
}

// ValidateOptions performs comprehensive validation of the mixer configuration.
// It checks all required fields are set and have valid values before any
// processing begins.
//
// The validation includes checking:
// - Input and output paths are specified
// - File pattern is not empty
// - Size limits are positive values
// - Output format is supported
//
// Returns:
//   - error: nil if validation passes, otherwise a MixError describing the issue
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

	if m.options.MaxOutputSize <= 0 {
		return &MixError{Message: "max output size must be greater than 0"}
	}

	switch m.options.OutputType {
	case OutputTypeXML, OutputTypeJSON, OutputTypeYAML:
		// Valid output types
	default:
		return &MixError{Message: fmt.Sprintf("unsupported output type: %s", m.options.OutputType)}
	}

	return nil
}
