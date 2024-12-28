package core

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestNewFileFinder(t *testing.T) {
	includes := []string{"*.txt"}
	excludes := []string{"temp*"}
	ff := NewFileFinder(includes, excludes, true)

	if ff == nil {
		t.Fatal("NewFileFinder returned nil")
	}
	if len(ff.includes) != 1 || ff.includes[0] != "*.txt" {
		t.Errorf("Expected includes to be [*.txt], got %v", ff.includes)
	}
	if len(ff.excludes) != 1 || ff.excludes[0] != "temp*" {
		t.Errorf("Expected excludes to be [temp*], got %v", ff.excludes)
	}
	if !ff.followSymlinks {
		t.Error("Expected followSymlinks to be true")
	}
}

func setupTestFiles(t *testing.T) (string, func()) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "filefinder_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create test file structure
	files := []string{
		"file1.txt",
		"file2.log",
		"temp.txt",
		"subdir/file3.txt",
		"subdir/file4.log",
		"subdir/temp.log",
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create directory structure: %v", err)
		}
		err = os.WriteFile(path, []byte("test content"), 0644)
		if err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a symlink for testing
	err = os.Symlink(
		filepath.Join(tempDir, "file1.txt"),
		filepath.Join(tempDir, "link1.txt"),
	)
	if err != nil && !os.IsExist(err) {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create symlink: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestFindMatchingFiles(t *testing.T) {
	tempDir, cleanup := setupTestFiles(t)
	defer cleanup()

	tests := []struct {
		name          string
		includes      []string
		excludes      []string
		followSymlink bool
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Match all txt files",
			includes:      []string{"**.txt"},
			excludes:      nil,
			followSymlink: false,
			expectedCount: 3, // file1.txt, temp.txt, subdir/file3.txt
		},
		{
			name:          "Match txt files excluding temp",
			includes:      []string{"**.txt"},
			excludes:      []string{"**/temp*"},
			followSymlink: false,
			expectedCount: 2, // file1.txt, subdir/file3.txt
		},
		{
			name:          "Match with symlinks",
			includes:      []string{"**.txt"},
			excludes:      []string{"**/temp*"},
			followSymlink: true,
			expectedCount: 3, // file1.txt, subdir/file3.txt, link1.txt
		},
		{
			name:          "Match only log files",
			includes:      []string{"**.log"},
			excludes:      nil,
			followSymlink: false,
			expectedCount: 3, // file2.log, subdir/file4.log, subdir/temp.log
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ff := NewFileFinder(tt.includes, tt.excludes, tt.followSymlink)
			matches, err := ff.FindMatchingFiles([]string{tempDir})

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(matches) != tt.expectedCount {
				t.Errorf("Expected %d matches, got %d: %v",
					tt.expectedCount, len(matches), matches)
			}
		})
	}
}

func TestProcessSymlink(t *testing.T) {
	tempDir, cleanup := setupTestFiles(t)
	defer cleanup()

	ff := NewFileFinder([]string{"**.txt"}, nil, true)
	resultChan := make(chan Result)

	go func() {
		err := ff.processSymlink(filepath.Join(tempDir, "link1.txt"), resultChan)
		if err != nil {
			t.Errorf("Unexpected error processing symlink: %v", err)
		}
		close(resultChan)
	}()

	results := []string{}
	for result := range resultChan {
		if result.Err != nil {
			t.Errorf("Unexpected error in result: %v", result.Err)
			continue
		}
		results = append(results, result.Path)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d: %v", len(results), results)
	}
}

func TestShouldIncludeFile(t *testing.T) {
	tests := []struct {
		name     string
		includes []string
		excludes []string
		path     string
		expected bool
	}{
		{
			name:     "Match include pattern",
			includes: []string{"*.txt"},
			excludes: nil,
			path:     "test.txt",
			expected: true,
		},
		{
			name:     "Match exclude pattern",
			includes: []string{"*.txt"},
			excludes: []string{"temp*"},
			path:     "temp.txt",
			expected: false,
		},
		{
			name:     "No include patterns",
			includes: nil,
			excludes: []string{"temp*"},
			path:     "file.log",
			expected: true,
		},
		{
			name:     "Path with directories",
			includes: []string{"**/test/*.txt"},
			excludes: nil,
			path:     "path/to/test/file.txt",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ff := NewFileFinder(tt.includes, tt.excludes, false)
			result, err := ff.shouldIncludeFile(tt.path)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Expected shouldIncludeFile to return %v for path %s, got %v",
					tt.expected, tt.path, result)
			}
		})
	}
}

func TestGetRealPath(t *testing.T) {
	tempDir, cleanup := setupTestFiles(t)
	defer cleanup()

	ff := NewFileFinder(nil, nil, true)
	linkPath := filepath.Join(tempDir, "link1.txt")
	realPath := filepath.Join(tempDir, "file1.txt")

	got, err := ff.GetRealPath(linkPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	// Normalize paths for comparison
	gotAbs, _ := filepath.Abs(got)
	realAbs, _ := filepath.Abs(realPath)
	if gotAbs != realAbs {
		t.Errorf("Expected real path %s, got %s", realAbs, gotAbs)
	}
}

func TestIsSymlink(t *testing.T) {
	tempDir, cleanup := setupTestFiles(t)
	defer cleanup()

	ff := NewFileFinder([]string{"**.txt"}, nil, true)
	linkPath := filepath.Join(tempDir, "link1.txt")

	// First, find all files to populate the seenLinks map
	_, err := ff.FindMatchingFiles([]string{tempDir})
	if err != nil {
		t.Fatalf("Failed to find files: %v", err)
	}

	isLink, err := ff.IsSymlink(linkPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if !isLink {
		t.Errorf("Expected %s to be recognized as symlink", linkPath)
	}

	nonLinkPath := filepath.Join(tempDir, "file1.txt")
	isLink, err = ff.IsSymlink(nonLinkPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if isLink {
		t.Errorf("Expected %s to not be recognized as symlink", nonLinkPath)
	}
}

func TestConcurrentFinding(t *testing.T) {
	tempDir, cleanup := setupTestFiles(t)
	defer cleanup()

	// Create multiple base paths for testing concurrency
	basePaths := []string{
		filepath.Join(tempDir, "subdir"),
		tempDir,
	}

	ff := NewFileFinder([]string{"**.txt"}, []string{"**/temp*"}, true)
	matches, err := ff.FindMatchingFiles(basePaths)
	if err != nil {
		t.Fatalf("Failed to find files concurrently: %v", err)
	}

	// Sort matches for consistent comparison
	sort.Strings(matches)

	// We expect file1.txt, subdir/file3.txt, and link1.txt (when following symlinks)
	expectedCount := 3
	if len(matches) != expectedCount {
		t.Errorf("Expected %d matches, got %d: %v", expectedCount, len(matches), matches)
	}
}
