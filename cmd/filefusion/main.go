package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drgsn/filefusion/internal/core"
	"github.com/drgsn/filefusion/internal/core/cleaner"
	"github.com/spf13/cobra"
)

// Command-line flags
var (
	// Core flags
	outputPath     string
	pattern        string
	exclude        string
	maxFileSize    string
	maxOutputSize  string
	dryRun         bool
	ignoreSymlinks bool

	// Cleaner flags
	cleanEnabled         bool
	removeComments       bool
	preserveDocComments  bool
	removeImports        bool
	removeLogging        bool
	removeGettersSetters bool
	optimizeWhitespace   bool
	removeEmptyLines     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "filefusion [paths...]",
	Short: "Filefusion - File concatenation tool optimized for LLM usage",
	Long: `Filefusion concatenates files into a format optimized for Large Language Models (LLMs).
It preserves file metadata and structures the output in an XML-like or JSON format.
Complete documentation is available at https://github.com/drgsn/filefusion`,
	RunE: runMix,
}

func init() {
	initCoreFlags()
	initCleanerFlags()
}

// initCoreFlags initializes the core command-line flags
func initCoreFlags() {
	rootCmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "", "output file path")
	rootCmd.PersistentFlags().StringVarP(&pattern, "pattern", "p", "*.go,*.json,*.yaml,*.yml", "file patterns")
	rootCmd.PersistentFlags().StringVarP(&exclude, "exclude", "e", "", "exclude patterns")
	rootCmd.PersistentFlags().StringVar(&maxFileSize, "max-file-size", "10MB", "maximum size for individual input files")
	rootCmd.PersistentFlags().StringVar(&maxOutputSize, "max-output-size", "50MB", "maximum size for output file")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show the list of files that will be processed")
	rootCmd.PersistentFlags().BoolVar(&ignoreSymlinks, "ignore-symlinks", false, "Ignore symbolic links when processing files")
}

// initCleanerFlags initializes the code cleaner flags
func initCleanerFlags() {
	rootCmd.PersistentFlags().BoolVar(&cleanEnabled, "clean", false, "enable code cleaning")
	rootCmd.PersistentFlags().BoolVar(&removeComments, "clean-remove-comments", true, "remove comments during cleaning")
	rootCmd.PersistentFlags().BoolVar(&preserveDocComments, "clean-preserve-doc-comments", true, "preserve documentation comments")
	rootCmd.PersistentFlags().BoolVar(&removeImports, "clean-remove-imports", false, "remove import statements")
	rootCmd.PersistentFlags().BoolVar(&removeLogging, "clean-remove-logging", true, "remove logging statements")
	rootCmd.PersistentFlags().BoolVar(&removeGettersSetters, "clean-remove-getters-setters", true, "remove getter/setter methods")
	rootCmd.PersistentFlags().BoolVar(&optimizeWhitespace, "clean-optimize-whitespace", true, "optimize whitespace")
	rootCmd.PersistentFlags().BoolVar(&removeEmptyLines, "clean-remove-empty-lines", true, "remove empty lines")
}

// main is the entry point of the application
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runMix implements the main program logic
func runMix(cmd *cobra.Command, args []string) error {
	// Validate and get initial configuration
	config, err := validateAndGetConfig(args)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error getting current working directory: %w", err)
		}
		args = []string{currentDir}
	}

	// Create file manager
	fileManager := core.NewFileManager(config.MaxFileSize, config.MaxOutputSize, config.OutputType)

	// Get list of files using FileFinder
	finder := core.NewFileFinder(config.IncludePatterns, config.ExcludePatterns, !ignoreSymlinks)
	files, err := finder.FindMatchingFiles(args)
	if err != nil {
		return fmt.Errorf("error finding files: %w", err)
	}

	// Validate files against size limits
	validFiles, err := fileManager.ValidateFiles(files)
	if err != nil {
		return err
	}

	if dryRun {
		fmt.Println("\nDry run complete. No files will be processed.")
		return nil
	}

	// Get output paths
	outputPaths, err := fileManager.DeriveOutputPaths(args, outputPath)
	if err != nil {
		return err
	}

	// Group files by output path
	fileGroups, err := fileManager.GroupFilesByOutput(validFiles, outputPaths)
	if err != nil {
		return err
	}

	// Process each group and generate output
	for _, group := range fileGroups {
		// Create processor for this group
		processor := core.NewFileProcessor(&core.MixOptions{
			MaxFileSize:    config.MaxFileSize,
			MaxOutputSize:  config.MaxOutputSize,
			OutputType:     config.OutputType,
			CleanerOptions: config.CleanerOptions,
		})

		// Process files
		contents, err := processor.ProcessFiles(group.Files)
		if err != nil {
			return fmt.Errorf("error processing files for %s: %w", group.OutputPath, err)
		}

		// Generate output
		generator, err := core.NewOutputGenerator(&core.MixOptions{
			OutputPath:    group.OutputPath,
			OutputType:    config.OutputType,
			MaxOutputSize: config.MaxOutputSize,
		})
		if err != nil {
			return fmt.Errorf("error creating output: %w", err)
		}

		if err := generator.Generate(contents); err != nil {
			return fmt.Errorf("error generating output for %s: %w", group.OutputPath, err)
		}

		fmt.Printf("Generated output: %s\n", group.OutputPath)
	}

	return nil
}

// Config holds the validated configuration for processing
type Config struct {
	IncludePatterns []string
	ExcludePatterns []string
	MaxFileSize     int64
	MaxOutputSize   int64
	OutputType      core.OutputType
	CleanerOptions  *cleaner.CleanerOptions
}

// validateAndGetConfig validates inputs and returns a Config struct
func validateAndGetConfig(args []string) (*Config, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	validator := core.NewPatternValidator()
	includePatterns, err := validator.ExpandPattern(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid include pattern: %w", err)
	}

	var excludePatterns []string
	if exclude != "" {
		excludePatterns, err = validator.ExpandPattern(exclude)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern: %w", err)
		}
	}

	fileManager := core.NewFileManager(0, 0, core.OutputTypeXML) // Temporary instance for parsing
	maxFileSizeBytes, err := fileManager.ParseSize(maxFileSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max-file-size value: %w", err)
	}

	maxOutputSizeBytes, err := fileManager.ParseSize(maxOutputSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max-output-size value: %w", err)
	}

	outputType, err := validateAndGetOutputType(outputPath)
	if err != nil {
		return nil, err
	}

	cleanerOpts := getCleanerOptions()

	return &Config{
		IncludePatterns: includePatterns,
		ExcludePatterns: excludePatterns,
		MaxFileSize:     maxFileSizeBytes,
		MaxOutputSize:   maxOutputSizeBytes,
		OutputType:      outputType,
		CleanerOptions:  cleanerOpts,
	}, nil
}

// getCleanerOptions creates cleaner options based on command-line flags
func getCleanerOptions() *cleaner.CleanerOptions {
	if !cleanEnabled {
		return nil
	}

	return &cleaner.CleanerOptions{
		RemoveComments:       removeComments,
		PreserveDocComments:  preserveDocComments,
		RemoveImports:        removeImports,
		RemoveLogging:        removeLogging,
		RemoveGettersSetters: removeGettersSetters,
		OptimizeWhitespace:   optimizeWhitespace,
		RemoveEmptyLines:     removeEmptyLines,
	}
}

// validateOutputType validates and returns the output type
func validateAndGetOutputType(outputPath string) (core.OutputType, error) {
	if outputPath == "" {
		return core.OutputTypeXML, nil
	}

	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".json":
		return core.OutputTypeJSON, nil
	case ".yaml", ".yml":
		return core.OutputTypeYAML, nil
	case ".xml":
		return core.OutputTypeXML, nil
	default:
		return "", fmt.Errorf("invalid output file extension: must be .xml, .json, .yaml, or .yml")
	}
}
