package core

// FileContent represents a file's content and metadata
type FileContent struct {
    Path      string `json:"path"`      // Path to the file (relative to input directory)
    Name      string `json:"name"`      // Name of the file
    Content   string `json:"content"`   // Content of the file
    Extension string `json:"extension"` // File extension without dot
    Size      int64  `json:"size"`     // File size in bytes
}

// MixOptions represents configuration options for the mixer
type MixOptions struct {
    InputPath   string // Directory to process
    OutputPath  string // Output file path
    Pattern     string // Comma-separated file matching patterns
    Exclude     string   // Comma-separated exclusion patterns
    MaxFileSize int64  // Maximum total size in bytes
    JsonOutput  bool   // Whether to output JSON
    YamlOutput  bool   // Whether to output YAML
}

// MixError represents an error that occurred during mixing
type MixError struct {
    File    string // File where error occurred
    Message string // Error message
}

// Error implements the error interface for MixError
func (e *MixError) Error() string {
    if e.File != "" {
        return "file " + e.File + ": " + e.Message
    }
    return e.Message
}