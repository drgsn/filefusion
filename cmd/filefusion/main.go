package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"

    "github.com/spf13/cobra"
    "github.com/drgsn/filefusion/internal/core"
)

type FileInfo struct {
    Path string `json:"path"`
    Size int64  `json:"size"`
}

var (
    inputPath    string
    outputPath   string
    pattern      string
    exclude      string
    maxFileSize  string
    jsonOutput   bool
)

var rootCmd = &cobra.Command{
    Use:   "filefusion",
    Short: "Filefusion - File concatenation tool optimized for LLM usage",
    Long: `Filefusion concatenates files into a format optimized for Large Language Models (LLMs).
It preserves file metadata and structures the output in an XML-like or JSON format.
Complete documentation is available at https://github.com/drgsn/filefusion`,
    RunE: runMix,
}

func init() {
    currentDir, err := os.Getwd()
    if err != nil {
        currentDir = "."
    }

    rootCmd.PersistentFlags().StringVarP(&inputPath, "input", "i", currentDir, "input directory path (default: current directory)")
    rootCmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "output.xml", "output file path (default: output.xml)")
    rootCmd.PersistentFlags().StringVarP(&pattern, "pattern", "p", "*.go,*.json,*.yaml,*.yml", "comma-separated file patterns (e.g., '*.go,*.json')")
    rootCmd.PersistentFlags().StringVarP(&exclude, "exclude", "e", "", "comma-separated patterns to exclude (e.g., 'build/**,*.jar')")
    rootCmd.PersistentFlags().StringVar(&maxFileSize, "max-size", "10MB", "maximum size per file")
}

func main() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}


func runMix(cmd *cobra.Command, args []string) error {
    // Validate output path
    ext := strings.ToLower(filepath.Ext(outputPath))
    if ext != ".xml" && ext != ".json" && ext != ".yaml" && ext != ".yml" {
        return fmt.Errorf("output file must have .xml, .json, .yaml, or .yml extension")
    }
    jsonOutput = ext == ".json"
    yamlOutput := ext == ".yaml" || ext == ".yml"

    // Parse max file size
    maxBytes, err := parseSize(maxFileSize)
    if err != nil {
        return fmt.Errorf("invalid max-size value: %w", err)
    }

    // Create mixer options
    options := &core.MixOptions{
        InputPath:   inputPath,
        OutputPath:  outputPath,
        Pattern:     pattern,
        MaxFileSize: maxBytes,
        JsonOutput:  jsonOutput,
        YamlOutput:  yamlOutput,
    }

    // First, scan for files and check total size
    files, totalSize, err := scanFiles(options)
    if err != nil {
        return err
    }

    // Print summary before processing
    fmt.Printf("Found %d files matching pattern\n", len(files))
    fmt.Printf("Total size: %s\n", formatSize(totalSize))

    // Check if total size exceeds maximum
    if totalSize > maxBytes {
        fmt.Printf("\nError: Total size (%s) exceeds maximum allowed size (%s)\n", 
            formatSize(totalSize), maxFileSize)
        fmt.Println("\nMatching files:")
        
        // Sort files by size (largest first) and print details
        for _, file := range files {
            fmt.Printf("- %s (%s)\n", file.Path, formatSize(file.Size))
        }
        
        return fmt.Errorf("total size exceeds maximum allowed size")
    }

    // Create and run mixer
    mixer := core.NewMixer(options)
    if err := mixer.Mix(); err != nil {
        return err
    }

    fmt.Printf("\nSuccessfully created %s\n", outputPath)
    return nil
}

func scanFiles(options *core.MixOptions) ([]FileInfo, int64, error) {
    var files []FileInfo
    var totalSize int64

    patterns := strings.Split(options.Pattern, ",")
    for i := range patterns {
        patterns[i] = strings.TrimSpace(patterns[i])
    }

    err := filepath.Walk(options.InputPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }

        // Check if file matches any pattern
        for _, pattern := range patterns {
            match, err := filepath.Match(pattern, filepath.Base(path))
            if err != nil {
                return err
            }
            if match {
                relPath, err := filepath.Rel(options.InputPath, path)
                if err != nil {
                    relPath = path
                }
                files = append(files, FileInfo{
                    Path: relPath,
                    Size: info.Size(),
                })
                totalSize += info.Size()
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