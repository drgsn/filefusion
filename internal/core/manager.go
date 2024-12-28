package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FileManager handles file organization, path management, and size calculations
type FileManager struct {
	maxFileSize   int64
	maxOutputSize int64
	outputType    OutputType
}

// FileGroup represents a collection of files destined for the same output
type FileGroup struct {
	OutputPath string
	Files      []string
}

// NewFileManager creates a new FileManager with the specified size limits and output type
func NewFileManager(maxFileSize, maxOutputSize int64, outputType OutputType) *FileManager {
	return &FileManager{
		maxFileSize:   maxFileSize,
		maxOutputSize: maxOutputSize,
		outputType:    outputType,
	}
}

// ValidateFiles checks files against size limits and returns valid ones
func (fm *FileManager) ValidateFiles(files []string) ([]string, error) {
	var validFiles []string
	var totalSize int64
	var ignoredCount int

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			return nil, fmt.Errorf("error getting file info: %w", err)
		}

		if info.Size() > fm.maxFileSize {
			ignoredCount++
			fmt.Printf("%s⚠️  IGNORED: %s%s\n", ColorRed, file, ColorReset)
			fmt.Printf("   Size: %s (exceeds limit of %s)\n", formatSize(info.Size()), formatSize(fm.maxFileSize))
			fmt.Println()
			continue
		}

		fmt.Printf("%s✓ INCLUDED: %s (%s)%s\n", ColorGreen, file, formatSize(info.Size()), ColorReset)
		validFiles = append(validFiles, file)
		totalSize += info.Size()
	}

	if ignoredCount > 0 {
		fmt.Printf("\n%s⚠️  Warning: %d file(s) were ignored due to size limits%s\n\n", ColorRed, ignoredCount, ColorReset)
	}

	if len(validFiles) == 0 {
		return nil, fmt.Errorf("no valid files found matching patterns")
	}

	if totalSize > fm.maxOutputSize {
		return nil, fmt.Errorf("total size of valid files (%s) exceeds maximum output size (%s)",
			formatSize(totalSize), formatSize(fm.maxOutputSize))
	}

	return validFiles, nil
}

// GroupFilesByOutput organizes files into groups based on their output destinations
func (fm *FileManager) GroupFilesByOutput(files []string, outputPaths []string) ([]FileGroup, error) {
	// If only one output path, group all files there
	if len(outputPaths) == 1 {
		return []FileGroup{{
			OutputPath: outputPaths[0],
			Files:      files,
		}}, nil
	}

	// Map for paths without extensions
	pathsWithoutExt := make(map[string]string)
	for _, outPath := range outputPaths {
		pathWithoutExt := strings.TrimSuffix(outPath, filepath.Ext(outPath))
		pathsWithoutExt[pathWithoutExt] = outPath
	}

	// Group files by their best matching output path
	groups := make(map[string][]string)
	var unmatchedFiles []string

	for _, file := range files {
		filePathWithoutExt := strings.TrimSuffix(file, filepath.Ext(file))
		bestMatch := fm.findBestMatchingPath(filePathWithoutExt, pathsWithoutExt)

		if bestMatch == "" {
			unmatchedFiles = append(unmatchedFiles, file)
			continue
		}

		outputPath := pathsWithoutExt[bestMatch]
		groups[outputPath] = append(groups[outputPath], file)
	}

	// Handle unmatched files
	if len(unmatchedFiles) > 0 {
		firstOutputWithoutExt := strings.TrimSuffix(outputPaths[0], filepath.Ext(outputPaths[0]))
		unmatchedPath := firstOutputWithoutExt + "_unmatched.xml"
		groups[unmatchedPath] = unmatchedFiles
	}

	// Convert map to slice of FileGroups
	var result []FileGroup
	for outputPath, files := range groups {
		result = append(result, FileGroup{
			OutputPath: outputPath,
			Files:      files,
		})
	}

	return result, nil
}

// DeriveOutputPaths generates output paths for the given input paths
func (fm *FileManager) DeriveOutputPaths(inputPaths []string, customOutputPath string) ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if customOutputPath != "" {
		return []string{filepath.Join(currentDir, customOutputPath)}, nil
	}

	result := make([]string, len(inputPaths))
	for i, path := range inputPaths {
		if path == "." {
			dirName := filepath.Base(currentDir)
			result[i] = filepath.Join(currentDir, fm.addDefaultExtension(dirName))
		} else {
			result[i] = filepath.Join(currentDir, fm.deriveOutputName(path))
		}
	}

	return result, nil
}

// ParseSize converts a size string (e.g., "10MB") to bytes
func (fm *FileManager) ParseSize(size string) (int64, error) {
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

// Helper methods

func (fm *FileManager) findBestMatchingPath(filePath string, dirPaths map[string]string) string {
	var bestMatch string
	var maxCommonSegments int

	fileComponents := strings.Split(filepath.Clean(filePath), string(os.PathSeparator))

	for dirPath := range dirPaths {
		dirComponents := strings.Split(filepath.Clean(dirPath), string(os.PathSeparator))
		commonSegments := fm.countCommonSegments(fileComponents, dirComponents)

		if commonSegments > maxCommonSegments || bestMatch == "" {
			maxCommonSegments = commonSegments
			bestMatch = dirPath
		}
	}

	if maxCommonSegments > 0 {
		return bestMatch
	}
	return ""
}

func (fm *FileManager) countCommonSegments(a, b []string) int {
	count := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
		count++
	}
	return count
}

func (fm *FileManager) deriveOutputName(path string) string {
	base := filepath.Base(strings.TrimSuffix(path, string(os.PathSeparator)))
	return fm.addDefaultExtension(base)
}

func (fm *FileManager) addDefaultExtension(name string) string {
	switch fm.outputType {
	case OutputTypeJSON:
		return name + ".json"
	case OutputTypeYAML:
		return name + ".yaml"
	default:
		return name + ".xml"
	}
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
