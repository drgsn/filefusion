package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"gopkg.in/yaml.v3"
)

func TestMixerFindFiles(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file structure
	files := map[string]string{
		"test1.go":           "package main\nfunc main() {}\n",
		"test2.go":           "package main\nfunc helper() {}\n",
		"data.json":          `{"key": "value"}`,
		"config.yaml":        "name: test",
		"subfolder/test.go":  "package sub\n",
		"subfolder/data.txt": "ignored file",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	tests := []struct {
		name          string
		pattern       string
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "find all go files",
			pattern:       "*.go",
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name:          "find json and yaml",
			pattern:       "*.json,*.yaml",
			expectedCount: 2,
			shouldError:   false,
		},
		{
			name:          "no matches",
			pattern:       "*.cpp",
			expectedCount: 0,
			shouldError:   true,
		},
		{
			name:          "invalid pattern",
			pattern:       "[",
			expectedCount: 0,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mixer := NewMixer(&MixOptions{
				InputPath: tmpDir,
				Pattern:  tt.pattern,
			})

			files, err := mixer.findFiles()
			
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(files) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(files))
			}
		})
	}
}


func TestMixerOutput(t *testing.T) {
    // Create temporary test directory
    tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    // Create test files
    inputFiles := map[string]string{
        "test1.go": "package main\n",
        "test2.go": "package test\n",
    }

    for name, content := range inputFiles {
        path := filepath.Join(tmpDir, name)
        if err := os.WriteFile(path, []byte(content), 0644); err != nil {
            t.Fatalf("Failed to create test file: %v", err)
        }
    }

    tests := []struct {
        name       string
        jsonOutput bool
        yamlOutput bool  // Add yamlOutput field
        validate   func(t *testing.T, output []byte)
    }{
        {
            name:       "XML output",
            jsonOutput: false,
            yamlOutput: false,
            validate: func(t *testing.T, output []byte) {
                content := string(output)
                if !strings.Contains(content, "<documents>") {
                    t.Error("Expected XML output to contain <documents> tag")
                }
                if !strings.Contains(content, "<document index=\"1\">") {
                    t.Error("Expected XML output to contain indexed document tags")
                }
            },
        },
        {
            name:       "JSON output",
            jsonOutput: true,
            yamlOutput: false,
            validate: func(t *testing.T, output []byte) {
                var result struct {
                    Documents []struct {
                        Index          int    `json:"index"`
                        Source         string `json:"source"`
                        DocumentContent string `json:"document_content"`
                    } `json:"documents"`
                }

                if err := json.Unmarshal(output, &result); err != nil {
                    t.Errorf("Failed to parse JSON output: %v", err)
                    return
                }

                if len(result.Documents) != 2 {
                    t.Errorf("Expected 2 documents in JSON output, got %d", len(result.Documents))
                }
            },
        },
        {
            name:       "YAML output",
            jsonOutput: false,
            yamlOutput: true,
            validate: func(t *testing.T, output []byte) {
                var result struct {
                    Documents []struct {
                        Index          int    `yaml:"index"`
                        Source         string `yaml:"source"`
                        DocumentContent string `yaml:"document_content"`
                    } `yaml:"documents"`
                }

                if err := yaml.Unmarshal(output, &result); err != nil {
                    t.Errorf("Failed to parse YAML output: %v", err)
                    return
                }

                if len(result.Documents) != 2 {
                    t.Errorf("Expected 2 documents in YAML output, got %d", len(result.Documents))
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            outputPath := filepath.Join(tmpDir, "output")
            switch {
            case tt.jsonOutput:
                outputPath += ".json"
            case tt.yamlOutput:
                outputPath += ".yaml"
            default:
                outputPath += ".xml"
            }

            mixer := NewMixer(&MixOptions{
                InputPath:   tmpDir,
                OutputPath:  outputPath,
                Pattern:    "*.go",
                MaxFileSize: 1024 * 1024,
                JsonOutput: tt.jsonOutput,
                YamlOutput: tt.yamlOutput,
            })

            if err := mixer.Mix(); err != nil {
                t.Fatalf("Mix failed: %v", err)
            }

            output, err := os.ReadFile(outputPath)
            if err != nil {
                t.Fatalf("Failed to read output file: %v", err)
            }

            tt.validate(t, output)
        })
    }
}

func TestMixerSizeLimit(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file larger than the limit
	largeContent := strings.Repeat("a", 1024*1024) // 1MB
	if err := os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(largeContent), 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	mixer := NewMixer(&MixOptions{
		InputPath:   tmpDir,
		OutputPath:  filepath.Join(tmpDir, "output.txt"),
		Pattern:    "*.txt",
		MaxFileSize: 1024, // 1KB limit
	})

	contents, err := mixer.readFiles([]string{filepath.Join(tmpDir, "large.txt")})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(contents) != 0 {
		t.Error("Expected file to be skipped due to size limit")
	}
}

func TestMixerExcludePatterns(t *testing.T) {
    // Create temporary test directory
    tmpDir, err := os.MkdirTemp("", "filefusion-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    // Create test file structure
    files := map[string]string{
        "src/main.go":              "package main\n",
        "src/util.go":              "package main\n",
        "build/main.go":            "package main\n",
        "build/app.jar":            "jar content",
        "services/build/app.jar":   "jar content",
        "services/lib/util.jar":    "jar content",
        "test/test.go":            "package test\n",
    }

    for path, content := range files {
        fullPath := filepath.Join(tmpDir, path)
        err := os.MkdirAll(filepath.Dir(fullPath), 0755)
        if err != nil {
            t.Fatalf("Failed to create directory: %v", err)
        }
        err = os.WriteFile(fullPath, []byte(content), 0644)
        if err != nil {
            t.Fatalf("Failed to create test file: %v", err)
        }
    }

    tests := []struct {
        name          string
        pattern       string
        exclude       string
        expectedFiles int
        shouldError   bool
    }{
        {
            name:          "exclude build directory",
            pattern:       "*.go",
            exclude:       "build/**",
            expectedFiles: 3, // src/main.go, src/util.go, test/test.go
            shouldError:   false,
        },
        {
            name:          "exclude all jar files",
            pattern:       "*.jar,*.go",
            exclude:       "*.jar",
            expectedFiles: 4, // all .go files
            shouldError:   false,
        },
        {
            name:          "exclude specific path",
            pattern:       "*.jar",
            exclude:       "services/build/*.jar",
            expectedFiles: 2, // build/app.jar, services/lib/util.jar
            shouldError:   false,
        },
        {
            name:          "exclude multiple patterns",
            pattern:       "*.go,*.jar",
            exclude:       "build/**,services/**",
            expectedFiles: 3, // src/main.go, src/util.go, test/test.go
            shouldError:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mixer := NewMixer(&MixOptions{
                InputPath: tmpDir,
                Pattern:  tt.pattern,
                Exclude:  tt.exclude,
            })

            files, err := mixer.findFiles()
            
            if tt.shouldError {
                if err == nil {
                    t.Error("Expected error but got none")
                }
                return
            }

            if err != nil {
                t.Errorf("Unexpected error: %v", err)
                return
            }

            if len(files) != tt.expectedFiles {
                t.Errorf("Expected %d files, got %d files. Files found: %v", 
                    tt.expectedFiles, len(files), files)
            }
        })
    }
}