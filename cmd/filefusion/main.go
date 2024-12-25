package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/drgsn/filefusion/internal/core"
	"github.com/drgsn/filefusion/internal/core/cleaner"
	"github.com/spf13/cobra"
)

// FileInfo represents basic information about a processed file
type FileInfo struct {
	Path string
	Size int64
}

// Command-line flags
var (
	// Core flags
	outputPath    string
	pattern       string
	exclude       string
	maxFileSize   string
	maxOutputSize string
	dryRun        bool

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
// In cmd/filefusion/main.go, modify the runMix function:

func runMix(cmd *cobra.Command, args []string) error {
    // Validate pattern first
    if pattern == "" {
        return fmt.Errorf("pattern cannot be empty")
    }

    // Add pattern validation
    if err := validatePattern(pattern); err != nil {
        return err
    }

    args, err := validateAndGetPaths(args)
    if err != nil {
        return err
    }

    sizeLimits, err := parseSizeLimits()
    if err != nil {
        return err
    }

    outputType, err := validateOutputType()
    if err != nil {
        return err
    }

    return processInputPaths(args, sizeLimits, outputType)
}

func validatePattern(pattern string) error {
    patterns := strings.Split(pattern, ",")
    for _, p := range patterns {
        p = strings.TrimSpace(p)
        if p == "" {
            continue
        }
        if _, err := filepath.Match(p, "test"); err != nil {
            return fmt.Errorf("syntax error in pattern %q: %w", p, err)
        }
    }
    return nil
}

// validateAndGetPaths validates input paths and returns them
func validateAndGetPaths(args []string) ([]string, error) {
	if len(args) == 0 {
		currentDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
		args = []string{currentDir}
	}
	return args, nil
}

// parseSizeLimits parses the size limit flags
func parseSizeLimits() (*struct{ maxFile, maxOutput int64 }, error) {
	maxFileSizeBytes, err := parseSize(maxFileSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max-file-size value: %w", err)
	}

	maxOutputSizeBytes, err := parseSize(maxOutputSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max-output-size value: %w", err)
	}

	return &struct{ maxFile, maxOutput int64 }{
		maxFile:   maxFileSizeBytes,
		maxOutput: maxOutputSizeBytes,
	}, nil
}

// validateOutputType validates and returns the output type
func validateOutputType() (core.OutputType, error) {
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

// processInputPaths processes each input path
func processInputPaths(args []string, sizes *struct{ maxFile, maxOutput int64 }, outputType core.OutputType) error {
	cleanerOpts := createCleanerOptions()

	for _, inputPath := range args {
		if err := processPath(inputPath, sizes, outputType, cleanerOpts); err != nil {
			return err
		}

		if outputPath != "" {
			fmt.Println("Note: Using specified output path. Additional inputs will be ignored.")
			break
		}
	}

	return nil
}

// createCleanerOptions creates cleaner options if cleaning is enabled
func createCleanerOptions() *cleaner.CleanerOptions {
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

// processPath processes a single input path
func processPath(inputPath string, sizes *struct{ maxFile, maxOutput int64 }, outputType core.OutputType, cleanerOpts *cleaner.CleanerOptions) error {
	currentOutputPath := getOutputPath(inputPath)

	options := &core.MixOptions{
		InputPath:      inputPath,
		OutputPath:     currentOutputPath,
		Pattern:        pattern,
		Exclude:        exclude,
		MaxFileSize:    sizes.maxFile,
		MaxOutputSize:  sizes.maxOutput,
		OutputType:     outputType,
		CleanerOptions: cleanerOpts,
	}

	files, totalSize, err := scanFiles(options)
	if err != nil {
		return fmt.Errorf("error processing %s: %w", inputPath, err)
	}

	if err := displaySummary(inputPath, files, totalSize); err != nil {
		return err
	}

	if dryRun {
		fmt.Println("\nDry run complete. No files will be processed.")
		return nil
	}

	return finalizeProcessing(options, files, totalSize)
}

// getOutputPath determines the output path for the current input
func getOutputPath(inputPath string) string {
	if outputPath != "" {
		return outputPath
	}
	return deriveOutputPath(inputPath)
}

// deriveOutputPath generates an output file path based on the input path
func deriveOutputPath(inputPath string) string {
	base := filepath.Base(strings.TrimSuffix(inputPath, string(os.PathSeparator)))
	if ext := filepath.Ext(base); ext != "" {
		return base + ".xml"
	}
	return base + ".xml"
}

// displaySummary shows the processing summary
func displaySummary(inputPath string, files []FileInfo, totalSize int64) error {
	fmt.Printf("Processing %s:\n", inputPath)
	fmt.Printf("Found %d files matching pattern\n", len(files))
	if cleanEnabled {
		fmt.Printf("Uncompressed size: %s\n", formatSize(totalSize))
		fmt.Printf("Final size (with --clean): will be calculated after processing\n")
	} else {
		fmt.Printf("Total size: %s\n", formatSize(totalSize))
	}

	fmt.Println("\nMatched files:")
	for _, file := range files {
		fmt.Printf("- %s (%s)\n", file.Path, formatSize(file.Size))
	}

	return nil
}

// finalizeProcessing performs the final processing steps
func finalizeProcessing(options *core.MixOptions, files []FileInfo, totalSize int64) error {
	if totalSize > options.MaxOutputSize {
		return fmt.Errorf("output size (%s) exceeds maximum allowed size (%s)",
			formatSize(totalSize), formatSize(options.MaxOutputSize))
	}

	mixer := core.NewMixer(options)
	if err := mixer.Mix(); err != nil {
		return fmt.Errorf("error mixing %s: %w", options.InputPath, err)
	}

	// Display final size if clean is enabled
	if cleanEnabled {
		if info, err := os.Stat(options.OutputPath); err == nil {
			fmt.Printf("\nFinal size (with --clean): %s\n", formatSize(info.Size()))
		}
	}

	return nil
}

// scanFiles discovers and validates files to be processed
func scanFiles(options *core.MixOptions) ([]FileInfo, int64, error) {
	var files []FileInfo
	var totalSize int64

	patterns := strings.Split(options.Pattern, ",")
	for i := range patterns {
		patterns[i] = strings.TrimSpace(patterns[i])
	}

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

		if info.IsDir() {
			if filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(options.InputPath, path)
		if err != nil {
			relPath = path
		}

		if shouldIncludeFile(relPath, patterns, excludePatterns, info, options.MaxFileSize) {
			files = append(files, FileInfo{
				Path: relPath,
				Size: info.Size(),
			})
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	if len(files) == 0 {
		return nil, 0, fmt.Errorf("no files found matching pattern(s) %q (excluding %q) in %s",
			options.Pattern, options.Exclude, options.InputPath)
	}

	return files, totalSize, nil
}

// shouldIncludeFile determines if a file should be included in processing
func shouldIncludeFile(relPath string, patterns, excludePatterns []string, info os.FileInfo, maxSize int64) bool {
	// Check exclusions first
	for _, pattern := range excludePatterns {
		if pattern == "" {
			continue
		}

		if matchesExcludePattern(pattern, relPath) {
			return false
		}
	}

	// Check size limit
	if info.Size() > maxSize {
		return false
	}

	// Check inclusion patterns
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}

		match, err := filepath.Match(pattern, filepath.Base(relPath))
		if err == nil && match {
			return true
		}
	}

	return false
}

// matchesExcludePattern checks if a path matches an exclude pattern
func matchesExcludePattern(pattern, path string) bool {
	if strings.Contains(pattern, "**") {
		pattern = strings.ReplaceAll(pattern, "**", "*")
		matched, _ := filepath.Match(pattern, path)
		return matched
	}

	matched, _ := filepath.Match(pattern, path)
	return matched
}

// parseSize converts a size string to bytes
func parseSize(size string) (int64, error) {
	size = strings.ToUpper(strings.ReplaceAll(size, " ", ""))
	if size == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}

	var multiplier int64 = 1
	var value string

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

	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number: %w", err)
	}

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
