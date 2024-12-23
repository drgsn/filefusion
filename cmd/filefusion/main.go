package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/drgsn/filefusion/internal/core"
	"github.com/spf13/cobra"
)

// FileInfo represents basic information about a processed file.
// It is used for reporting and validation purposes before the actual processing begins.
type FileInfo struct {
	Path string `json:"path"` // Relative path to the file
	Size int64  `json:"size"` // File size in bytes
}

// Command-line flags
var (
	outputPath    string // Path where the output file will be written
	pattern       string // File patterns to include (comma-separated)
	exclude       string // Patterns to exclude (comma-separated)
	maxFileSize   string // Maximum size limit for individual files
	maxOutputSize string // Maximum size limit for the output file
)

// rootCmd represents the base command when called without any subcommands.
// It configures the CLI interface and defines the main program behavior.
var rootCmd = &cobra.Command{
	Use:   "filefusion [paths...]",
	Short: "Filefusion - File concatenation tool optimized for LLM usage",
	Long: `Filefusion concatenates files into a format optimized for Large Language Models (LLMs).
It preserves file metadata and structures the output in an XML-like or JSON format.
Complete documentation is available at https://github.com/drgsn/filefusion`,
	RunE: runMix,
}

// init initializes the command-line flags with their default values.
// This function is called automatically by Go during package initialization.
func init() {
	rootCmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "", "output file path (if not specified, generates files based on input paths)")
	rootCmd.PersistentFlags().StringVarP(&pattern, "pattern", "p", "*.go,*.json,*.yaml,*.yml", "comma-separated file patterns (e.g., '*.go,*.json')")
	rootCmd.PersistentFlags().StringVarP(&exclude, "exclude", "e", "", "comma-separated patterns to exclude (e.g., 'build/**,*.jar')")
	rootCmd.PersistentFlags().StringVar(&maxFileSize, "max-file-size", "10MB", "maximum size for individual input files")
	rootCmd.PersistentFlags().StringVar(&maxOutputSize, "max-output-size", "50MB", "maximum size for the output file")
}

// main is the entry point of the application.
// It executes the root command and handles any errors that occur.
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// deriveOutputPath generates an output file path based on the input path.
// For directories, it uses the directory name with .xml extension.
// For files, it appends .xml to the existing filename.
//
// Parameters:
//   - inputPath: Path to the input file or directory
//
// Returns:
//   - A derived output path with .xml extension
func deriveOutputPath(inputPath string) string {
	// Get the last component of the path
	base := filepath.Base(strings.TrimSuffix(inputPath, string(os.PathSeparator)))

	// If it's a file, use its name with .xml extension
	if ext := filepath.Ext(base); ext != "" {
		return base + ".xml"
	}

	// For directories, append .xml
	return base + ".xml"
}

// runMix implements the main program logic when executing the root command.
// It processes command-line arguments, validates inputs, and orchestrates
// the file mixing process.
//
// Parameters:
//   - cmd: The Cobra command being run
//   - args: Command-line arguments after the command name
//
// Returns:
//   - error: nil if successful, otherwise an error describing what went wrong
func runMix(cmd *cobra.Command, args []string) error {

	// Use current directory if no paths specified
	if len(args) == 0 {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		args = []string{currentDir}
	}

	// Validate pattern first
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	// Parse max file size
	maxFileSizeBytes, err := parseSize(maxFileSize)
	if err != nil {
		return fmt.Errorf("invalid max-file-size value: %w", err)
	}

	// Parse max output size
	maxOutputSizeBytes, err := parseSize(maxOutputSize)
	if err != nil {
		return fmt.Errorf("invalid max-output-size value: %w", err)
	}

	// If output path is specified, validate and determine output type
	var globalOutputType core.OutputType
	if outputPath != "" {
		ext := strings.ToLower(filepath.Ext(outputPath))
		switch ext {
		case ".json":
			globalOutputType = core.OutputTypeJSON
		case ".yaml", ".yml":
			globalOutputType = core.OutputTypeYAML
		case ".xml":
			globalOutputType = core.OutputTypeXML
		default:
			return fmt.Errorf("invalid output file extension: must be .xml, .json, .yaml, or .yml")
		}
	}

	// Process each input path
	for _, inputPath := range args {
		// Determine output path and type for this input
		var currentOutputPath string
		var outputType core.OutputType

		if outputPath != "" {
			currentOutputPath = outputPath
			outputType = globalOutputType
		} else {
			currentOutputPath = deriveOutputPath(inputPath)
			outputType = core.OutputTypeXML // Default to XML for auto-generated paths
		}

		// Create mixer options
		options := &core.MixOptions{
			InputPath:     inputPath,
			OutputPath:    currentOutputPath,
			Pattern:       pattern,
			Exclude:       exclude,
			MaxFileSize:   maxFileSizeBytes,
			MaxOutputSize: maxOutputSizeBytes,
			OutputType:    outputType,
		}

		// First, scan for files and check total size
		files, totalSize, err := scanFiles(options)
		if err != nil {
			return fmt.Errorf("error processing %s: %w", inputPath, err)
		}

		// Print summary information
		fmt.Printf("Processing %s:\n", inputPath)
		fmt.Printf("Found %d files matching pattern\n", len(files))
		fmt.Printf("Total size: %s\n", formatSize(totalSize))

		// Validate total size against limit
		if totalSize > maxOutputSizeBytes {
			fmt.Printf("\nError: Total output size (%s) exceeds maximum allowed size (%s)\n",
				formatSize(totalSize), formatSize(maxOutputSizeBytes))
			fmt.Println("\nMatching files:")

			for _, file := range files {
				fmt.Printf("- %s (%s)\n", file.Path, formatSize(file.Size))
			}

			return fmt.Errorf("output size exceeds maximum allowed size")
		}

		// Create and run mixer
		mixer := core.NewMixer(options)
		if err := mixer.Mix(); err != nil {
			if strings.Contains(err.Error(), "output size exceeds maximum") {
				fmt.Println("\nMatching files:")
				for _, file := range files {
					fmt.Printf("- %s (%s)\n", file.Path, formatSize(file.Size))
				}
			}
			return fmt.Errorf("error mixing %s: %w", inputPath, err)
		}

		fmt.Printf("Successfully created %s\n\n", currentOutputPath)

		// If using a global output path, only process the first input
		if outputPath != "" {
			fmt.Println("Note: Using specified output path. Additional inputs will be ignored.")
			break
		}
	}

	return nil
}

func scanFiles(options *core.MixOptions) ([]FileInfo, int64, error) {
	var files []FileInfo
	var totalSize int64

	// Prepare patterns
	patterns := strings.Split(options.Pattern, ",")
	for i := range patterns {
		patterns[i] = strings.TrimSpace(patterns[i])
	}

	// Prepare exclude patterns
	var excludePatterns []string
	if options.Exclude != "" {
		excludePatterns = strings.Split(options.Exclude, ",")
		for i := range excludePatterns {
			excludePatterns[i] = strings.TrimSpace(excludePatterns[i])
		}
	}

	err := filepath.Walk(options.InputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip .git directory
			if filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path for pattern matching
		relPath, err := filepath.Rel(options.InputPath, path)
		if err != nil {
			relPath = path
		}

		// Check exclusions first
		for _, pattern := range excludePatterns {
			if pattern == "" {
				continue
			}

			// Handle directory wildcards
			if strings.Contains(pattern, "**") {
				pattern = strings.ReplaceAll(pattern, "**", "*")
				if matched, _ := filepath.Match(pattern, relPath); matched {
					return nil
				}
			} else {
				if matched, _ := filepath.Match(pattern, relPath); matched {
					return nil
				}
			}
		}

		// Check if file matches any inclusion pattern
		for _, pattern := range patterns {
			match, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return err
			}
			if match {
				// Check file size
				if info.Size() <= options.MaxFileSize {
					files = append(files, FileInfo{
						Path: relPath,
						Size: info.Size(),
					})
					totalSize += info.Size()
				}
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return files, totalSize, nil
}

// parseSize converts a size string (e.g., "10MB") to bytes

func parseSize(size string) (int64, error) {
	// Remove all spaces and convert to uppercase
	size = strings.ToUpper(strings.ReplaceAll(size, " ", ""))

	if size == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}

	var multiplier int64 = 1
	var value string

	// Check for valid suffix first
	switch {
	case strings.HasSuffix(size, "TB"):
		multiplier = 1024 * 1024 * 1024 * 1024
		value = strings.TrimSuffix(size, "TB")
	case strings.HasSuffix(size, "GB"):
		multiplier = 1024 * 1024 * 1024
		value = strings.TrimSuffix(size, "GB")
	case strings.HasSuffix(size, "MB"):
		multiplier = 1024 * 1024
		value = strings.TrimSuffix(size, "MB")
	case strings.HasSuffix(size, "KB"):
		multiplier = 1024
		value = strings.TrimSuffix(size, "KB")
	case strings.HasSuffix(size, "B"):
		value = strings.TrimSuffix(size, "B")
	default:
		return 0, fmt.Errorf("invalid size format: must end with B, KB, MB, GB, or TB")
	}

	// Parse the numeric value
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number: %w", err)
	}

	// Check for negative numbers
	if num <= 0 {
		return 0, fmt.Errorf("size must be a positive number")
	}

	return num * multiplier, nil
}

// formatSize converts bytes to a human-readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
