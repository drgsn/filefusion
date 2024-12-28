package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileManager(t *testing.T) {
	fm := NewFileManager(100, 1000, OutputTypeXML)
	if fm.maxFileSize != 100 {
		t.Errorf("Expected maxFileSize to be 100, got %d", fm.maxFileSize)
	}
	if fm.maxOutputSize != 1000 {
		t.Errorf("Expected maxOutputSize to be 1000, got %d", fm.maxOutputSize)
	}
	if fm.outputType != OutputTypeXML {
		t.Errorf("Expected outputType to be XML, got %v", fm.outputType)
	}
}

func TestValidateFiles(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()

	smallFile := filepath.Join(tempDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("small"), 0644); err != nil {
		t.Fatal(err)
	}

	largeFile := filepath.Join(tempDir, "large.txt")
	if err := os.WriteFile(largeFile, []byte("large content here"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		files         []string
		maxFileSize   int64
		maxOutputSize int64
		wantLen       int
		wantErr       bool
	}{
		{
			name:          "valid files within limits",
			files:         []string{smallFile},
			maxFileSize:   100,
			maxOutputSize: 200,
			wantLen:       1,
			wantErr:       false,
		},
		{
			name:          "file exceeds max file size",
			files:         []string{largeFile},
			maxFileSize:   5,
			maxOutputSize: 200,
			wantLen:       0,
			wantErr:       true,
		},
		{
			name:          "total size exceeds max output size",
			files:         []string{smallFile, largeFile},
			maxFileSize:   100,
			maxOutputSize: 10,
			wantLen:       0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := NewFileManager(tt.maxFileSize, tt.maxOutputSize, OutputTypeXML)
			got, err := fm.ValidateFiles(tt.files)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("ValidateFiles() got %v files, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestGroupFilesByOutput(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		outputPaths []string
		wantGroups  int
		wantErr     bool
	}{
		{
			name:        "single output path",
			files:       []string{"file1.txt", "file2.txt"},
			outputPaths: []string{"output.xml"},
			wantGroups:  1,
			wantErr:     false,
		},
		{
			name:        "multiple output paths",
			files:       []string{"dir1/file1.txt", "dir2/file2.txt"},
			outputPaths: []string{"dir1/out.xml", "dir2/out.xml"},
			wantGroups:  2,
			wantErr:     false,
		},
		{
			name:        "unmatched files",
			files:       []string{"dir1/file1.txt", "other/file2.txt"},
			outputPaths: []string{"dir1/out.xml", "dir2/out.xml"},
			wantGroups:  2,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := NewFileManager(1000, 10000, OutputTypeXML)
			groups, err := fm.GroupFilesByOutput(tt.files, tt.outputPaths)

			if (err != nil) != tt.wantErr {
				t.Errorf("GroupFilesByOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(groups) != tt.wantGroups {
				t.Errorf("GroupFilesByOutput() got %v groups, want %v", len(groups), tt.wantGroups)
			}
		})
	}
}

func TestDeriveOutputPaths(t *testing.T) {
	
	tests := []struct {
		name           string
		inputPaths     []string
		customOutput   string
		outputType     OutputType
		expectedSuffix string
		wantPathsCount int
	}{
		{
			name:           "current directory",
			inputPaths:     []string{"."},
			customOutput:   "",
			outputType:     OutputTypeXML,
			expectedSuffix: ".xml",
			wantPathsCount: 1,
		},
		{
			name:           "custom output path",
			inputPaths:     []string{"file1.txt"},
			customOutput:   "custom.json",
			outputType:     OutputTypeJSON,
			expectedSuffix: ".json",
			wantPathsCount: 1,
		},
		{
			name:           "multiple input paths",
			inputPaths:     []string{"file1.txt", "file2.txt"},
			customOutput:   "",
			outputType:     OutputTypeYAML,
			expectedSuffix: ".yaml",
			wantPathsCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := NewFileManager(1000, 10000, tt.outputType)
			paths, err := fm.DeriveOutputPaths(tt.inputPaths, tt.customOutput)

			if err != nil {
				t.Errorf("DeriveOutputPaths() error = %v", err)
				return
			}

			if len(paths) != tt.wantPathsCount {
				t.Errorf("DeriveOutputPaths() got %v paths, want %v", len(paths), tt.wantPathsCount)
			}

			for _, path := range paths {
				if !filepath.IsAbs(path) {
					t.Errorf("Expected absolute path, got %v", path)
				}
				if tt.customOutput == "" && !strings.HasSuffix(path, tt.expectedSuffix) {
					t.Errorf("Expected path to end with %v, got %v", tt.expectedSuffix, path)
				}
			}
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		want    int64
		wantErr bool
	}{
		{
			name:    "bytes",
			size:    "100B",
			want:    100,
			wantErr: false,
		},
		{
			name:    "kilobytes",
			size:    "1KB",
			want:    1024,
			wantErr: false,
		},
		{
			name:    "megabytes",
			size:    "1MB",
			want:    1024 * 1024,
			wantErr: false,
		},
		{
			name:    "gigabytes",
			size:    "1GB",
			want:    1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "terabytes",
			size:    "1TB",
			want:    1024 * 1024 * 1024 * 1024,
			wantErr: false,
		},
		{
			name:    "invalid format",
			size:    "100",
			want:    0,
			wantErr: true,
		},
		{
			name:    "negative size",
			size:    "-1MB",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			size:    "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid unit",
			size:    "100XB",
			want:    0,
			wantErr: true,
		},
	}

	fm := NewFileManager(1000, 10000, OutputTypeXML)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fm.ParseSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelperMethods(t *testing.T) {
	t.Run("findBestMatchingPath", func(t *testing.T) {
		fm := NewFileManager(1000, 10000, OutputTypeXML)
		dirPaths := map[string]string{
			"dir1/subdir": "dir1/subdir/out.xml",
			"dir2":        "dir2/out.xml",
		}

		tests := []struct {
			filePath  string
			wantMatch string
		}{
			{"dir1/subdir/file.txt", "dir1/subdir"},
			{"dir2/file.txt", "dir2"},
			{"dir3/file.txt", ""},
		}

		for _, tt := range tests {
			got := fm.findBestMatchingPath(tt.filePath, dirPaths)
			if got != tt.wantMatch {
				t.Errorf("findBestMatchingPath(%v) = %v, want %v", tt.filePath, got, tt.wantMatch)
			}
		}
	})

	t.Run("countCommonSegments", func(t *testing.T) {
		fm := NewFileManager(1000, 10000, OutputTypeXML)
		tests := []struct {
			a    []string
			b    []string
			want int
		}{
			{[]string{"dir1", "subdir"}, []string{"dir1", "subdir"}, 2},
			{[]string{"dir1", "subdir"}, []string{"dir1", "other"}, 1},
			{[]string{"dir1"}, []string{"dir2"}, 0},
		}

		for _, tt := range tests {
			got := fm.countCommonSegments(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("countCommonSegments(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		}
	})
}
