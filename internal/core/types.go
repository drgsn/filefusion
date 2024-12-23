package core

// FileContent encapsulates both the content and metadata of a processed file.
// This structure is used to pass file information between different components
// of the system and ultimately to generate the output.
type FileContent struct {
	// Path is the file path relative to the input directory.
	// This ensures consistent path representation across different platforms.
	Path string `json:"path"`

	// Name is the base name of the file without path information.
	// Used for display and logging purposes.
	Name string `json:"name"`

	// Content holds the actual file contents as a string.
	// Large files are handled according to MaxFileSize limits.
	Content string `json:"content"`

	// Extension is the file extension without the leading dot.
	// Used for format detection and filtering.
	Extension string `json:"extension"`

	// Size is the file size in bytes.
	// Used for size limit validation and reporting.
	Size int64 `json:"size"`
}

// OutputType represents the format in which the output file should be generated.
// This is determined by the output file extension or explicit configuration.
type OutputType string

// Supported output format constants.
// These determine how the final output file will be structured.
const (
	// OutputTypeXML generates output in XML format with proper declaration
	// and nested document structure.
	OutputTypeXML OutputType = "XML"

	// OutputTypeJSON generates output in JSON format with consistent
	// document structure and proper indentation.
	OutputTypeJSON OutputType = "JSON"

	// OutputTypeYAML generates output in YAML format, maintaining
	// the same document structure as JSON for consistency.
	OutputTypeYAML OutputType = "YAML"
)

// MixOptions encapsulates all configuration options for the file mixing process.
// This structure is passed to various components to control their behavior.
type MixOptions struct {
	// InputPath is the root directory or file to process.
	// All relative paths are calculated from this location.
	InputPath string

	// OutputPath is where the final mixed file will be written.
	// The extension of this path determines the output format.
	OutputPath string

	// Pattern is a comma-separated list of glob patterns for matching files.
	// For example: "*.go,*.json,*.yaml"
	Pattern string

	// Exclude is a comma-separated list of patterns for excluding files.
	// For example: "test/**,vendor/**"
	Exclude string

	// MaxFileSize is the maximum size in bytes for individual input files.
	// Files larger than this limit are skipped with a warning.
	MaxFileSize int64

	// MaxOutputSize is the maximum size in bytes for the output file.
	// The process fails if the combined output would exceed this limit.
	MaxOutputSize int64

	// OutputType determines the format of the output file.
	// Should be one of OutputTypeXML, OutputTypeJSON, or OutputTypeYAML.
	OutputType OutputType
}

// MixError represents an error that occurred during the mixing process.
// It provides context about which file (if any) caused the error.
type MixError struct {
	// File is the path to the file where the error occurred.
	// May be empty if the error is not specific to a file.
	File string

	// Message is a descriptive error message explaining what went wrong.
	Message string
}

// Error implements the error interface for MixError.
// It formats the error message to include the file path if available.
//
// Returns:
//   - A formatted error message string
func (e *MixError) Error() string {
	if e.File != "" {
		return "file " + e.File + ": " + e.Message
	}
	return e.Message
}
