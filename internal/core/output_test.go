package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name           string
		workDir        string
		inputPath      string
		expectedOutput string
	}{
		{
			name:           "Simple path",
			workDir:        "/home/user/project",
			inputPath:      "/home/user/project/file.go",
			expectedOutput: "project/file.go",
		},
		{
			name:           "Nested path",
			workDir:        "/home/user/project",
			inputPath:      "/home/user/project/src/main.go",
			expectedOutput: "project/src/main.go",
		},
		{
			name:           "Path outside workdir",
			workDir:        "/home/user/project",
			inputPath:      "/other/path/file.go",
			expectedOutput: "project/other/path/file.go",
		},
		{
			name:           "Relative path",
			workDir:        "/home/user/project",
			inputPath:      "src/file.go",
			expectedOutput: "project/src/file.go",
		},
		{
			name:           "Single directory workdir",
			workDir:        "project",
			inputPath:      "file.go",
			expectedOutput: "project/file.go",
		},
		// Added new test cases
		{
			name:           "Empty input path",
			workDir:        "/home/user/project",
			inputPath:      "",
			expectedOutput: "project",
		},
		{
			name:           "Path with dots",
			workDir:        "/home/user/project",
			inputPath:      "/home/user/project/../project/file.go",
			expectedOutput: "project/file.go",
		},
		{
			name:           "Windows style paths",
			workDir:        "C:\\Users\\user\\project",
			inputPath:      "C:\\Users\\user\\project\\file.go",
			expectedOutput: "project/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := &OutputGenerator{
				workDir: tt.workDir,
			}
			result := generator.normalizePath(tt.inputPath)
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

func TestOutputGenerator_Generate(t *testing.T) {
	// Save current working directory
	origWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(origWd)
		require.NoError(t, err)
	}()

	// Create a temporary directory and change to it
	tmpDir, err := os.MkdirTemp("", "output-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Rename the temporary directory to a fixed name to make tests predictable
	fixedDir := filepath.Join(filepath.Dir(tmpDir), "test-output")
	if err := os.RemoveAll(fixedDir); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Rename(tmpDir, fixedDir); err != nil {
		t.Fatal(err)
	}
	tmpDir = fixedDir

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create test directories
	err = os.MkdirAll(filepath.Join(tmpDir, "test"), 0755)
	require.NoError(t, err)

	// Create test content
	contents := []FileContent{
		{
			Path:      "test/file1.go",
			Name:      "file1.go",
			Content:   "package main\n\nfunc main() {}\n",
			Extension: ".go",
			Size:      30,
		},
		{
			Path:      "test/file2.go",
			Name:      "file2.go",
			Content:   "package test\n\nfunc Test() {}\n",
			Extension: ".go",
			Size:      29,
		},
	}

	tests := []struct {
		name        string
		outputType  OutputType
		maxSize     int64
		verifyFunc  func(t *testing.T, content []byte)
		expectError bool
	}{
		{
			name:       "XML Output",
			outputType: OutputTypeXML,
			maxSize:    1024,
			verifyFunc: func(t *testing.T, content []byte) {
				s := string(content)

				// Get the directory name that will be prefixed
				dirName := filepath.Base(tmpDir)
				expectedPath := filepath.Join(dirName, "test/file1.go")
				expectedPath = filepath.ToSlash(expectedPath)

				assert.Contains(t, s, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
				assert.Contains(t, s, "<documents>")
				assert.Contains(t, s, "<document index=\"1\">")
				assert.Contains(t, s, fmt.Sprintf("<source>%s</source>", expectedPath))
				assert.Contains(t, s, "<document_content>package main")
				assert.Contains(t, s, "</document>")
				assert.Contains(t, s, "</documents>")
				assert.NotContains(t, s, "<package")
			},
		},
		{
			name:       "JSON Output",
			outputType: OutputTypeJSON,
			maxSize:    1024,
			verifyFunc: func(t *testing.T, content []byte) {
				var output struct {
					Documents []struct {
						Index           int    `json:"index"`
						Source          string `json:"source"`
						DocumentContent string `json:"document_content"`
					} `json:"documents"`
				}
				err := json.Unmarshal(content, &output)
				require.NoError(t, err)
				assert.Len(t, output.Documents, 2)

				dirName := filepath.Base(tmpDir)
				expectedPath := filepath.Join(dirName, "test/file1.go")
				expectedPath = filepath.ToSlash(expectedPath)

				assert.Equal(t, expectedPath, output.Documents[0].Source)
				assert.Equal(t, 1, output.Documents[0].Index)
				assert.Equal(t, "package main\n\nfunc main() {}\n", output.Documents[0].DocumentContent)
			},
		},
		{
			name:       "YAML Output",
			outputType: OutputTypeYAML,
			maxSize:    1024,
			verifyFunc: func(t *testing.T, content []byte) {
				var output struct {
					Documents []struct {
						Index           int    `yaml:"index"`
						Source          string `yaml:"source"`
						DocumentContent string `yaml:"document_content"`
					} `yaml:"documents"`
				}
				err := yaml.Unmarshal(content, &output)
				require.NoError(t, err)
				assert.Len(t, output.Documents, 2)

				dirName := filepath.Base(tmpDir)
				expectedPath := filepath.Join(dirName, "test/file1.go")
				expectedPath = filepath.ToSlash(expectedPath)

				assert.Equal(t, expectedPath, output.Documents[0].Source)
				assert.Equal(t, 1, output.Documents[0].Index)
				assert.Equal(t, "package main\n\nfunc main() {}\n", output.Documents[0].DocumentContent)
			},
		},
		{
			name:        "Size Limit Exceeded",
			outputType:  OutputTypeXML,
			maxSize:     10, // Very small size limit
			expectError: true,
		},
		{
			name:        "Invalid Output Type",
			outputType:  "invalid",
			maxSize:     1024,
			expectError: true,
		},
		{
			name:       "Empty Content List",
			outputType: OutputTypeXML,
			maxSize:    1024,
			verifyFunc: func(t *testing.T, content []byte) {
				expected := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<documents>\n</documents>"
				assert.Equal(t, expected, string(content))
			},
		},
		{
			name:       "Special Characters",
			outputType: OutputTypeXML,
			maxSize:    1024,
			verifyFunc: func(t *testing.T, content []byte) {
				specialContent := []FileContent{
					{
						Path:    "test/special.go",
						Name:    "special.go",
						Content: "package main\n\n// Special chars: <>&'\"\n",
					},
				}

				generator, err := NewOutputGenerator(&MixOptions{
					OutputPath:    filepath.Join(tmpDir, "special.xml"),
					OutputType:    OutputTypeXML,
					MaxOutputSize: 1024,
				})
				require.NoError(t, err)

				err = generator.Generate(specialContent)
				require.NoError(t, err)

				content, err = os.ReadFile(filepath.Join(tmpDir, "special.xml"))
				require.NoError(t, err)

				s := string(content)
				assert.Contains(t, s, "&lt;")
				assert.Contains(t, s, "&gt;")
				assert.Contains(t, s, "&amp;")
				assert.Contains(t, s, "&apos;")
				assert.Contains(t, s, "&quot;")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "output"+string(tt.outputType))

			var testContents []FileContent
			if tt.name == "Empty Content List" {
				testContents = []FileContent{}
			} else if tt.name != "Special Characters" {
				testContents = contents
			}

			generator, err := NewOutputGenerator(&MixOptions{
				OutputPath:    outputPath,
				OutputType:    tt.outputType,
				MaxOutputSize: tt.maxSize,
			})
			require.NoError(t, err)

			// Generate output
			err = generator.Generate(testContents)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Read and verify output
			content, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, content)
			}
		})
	}
}

func TestNewOutputGenerator(t *testing.T) {
	tests := []struct {
		name        string
		options     *MixOptions
		expectError bool
	}{
		{
			name: "Valid Options",
			options: &MixOptions{
				OutputPath:    "output.xml",
				OutputType:    OutputTypeXML,
				MaxOutputSize: 1024,
			},
			expectError: false,
		},
		{
			name:        "Nil Options",
			options:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := NewOutputGenerator(tt.options)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, generator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, generator)
				assert.NotEmpty(t, generator.workDir)
			}
		})
	}
}

func TestOutputFormatting(t *testing.T) {
	// Save current working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// Create a temporary directory and change to it
	tmpDir, err := os.MkdirTemp("", "format-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Test content with special characters and formatting
	contents := []FileContent{
		{
			Path:    "test/special.go",
			Name:    "special.go",
			Content: "package main\n\n// Special chars: <>&'\"\n",
		},
		{
			Path:    "test/multiline.go",
			Name:    "multiline.go",
			Content: "line1\nline2\nline3",
		},
	}

	tests := []struct {
		name       string
		outputType OutputType
		verify     func(t *testing.T, content []byte)
	}{
		{
			name:       "XML Special Characters",
			outputType: OutputTypeXML,
			verify: func(t *testing.T, content []byte) {
				s := string(content)
				// XML should escape special characters
				assert.Contains(t, s, "&lt;")
				assert.Contains(t, s, "&gt;")
				assert.Contains(t, s, "&amp;")
				// Check structure
				assert.True(t, strings.HasPrefix(s, "<?xml"))
				assert.Contains(t, s, "<documents>")
			},
		},
		{
			name:       "JSON Special Characters",
			outputType: OutputTypeJSON,
			verify: func(t *testing.T, content []byte) {
				s := string(content)
				// JSON should escape special characters
				assert.Contains(t, s, `\"`)
				assert.Contains(t, s, `\n`)
				// Check structure
				var output map[string]interface{}
				err := json.Unmarshal(content, &output)
				assert.NoError(t, err)
			},
		},
		{
			name:       "YAML Special Characters",
			outputType: OutputTypeYAML,
			verify: func(t *testing.T, content []byte) {
				s := string(content)
				// YAML should handle special characters
				assert.Contains(t, s, "Special chars")
				// Check structure
				var output map[string]interface{}
				err := yaml.Unmarshal(content, &output)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, "output"+string(tt.outputType))
			generator, err := NewOutputGenerator(&MixOptions{
				OutputPath:    outputPath,
				OutputType:    tt.outputType,
				MaxOutputSize: 1024 * 1024,
			})
			require.NoError(t, err)

			err = generator.Generate(contents)
			assert.NoError(t, err)

			content, err := os.ReadFile(outputPath)
			assert.NoError(t, err)
			tt.verify(t, content)
		})
	}
}
