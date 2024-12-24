# FileFusion

FileFusion is a powerful command-line tool designed to concatenate and process files in a format optimized for Large Language Models (LLMs). It automatically preserves file metadata and structures the output in XML, JSON, or YAML format.

[![Test Coverage](https://codecov.io/gh/drgsn/filefusion/branch/main/graph/badge.svg)](https://codecov.io/gh/drgsn/filefusion)
[![Release](https://github.com/drgsn/filefusion/actions/workflows/release.yml/badge.svg)](https://github.com/drgsn/filefusion/actions/workflows/release.yml)

## Features

-   Combines multiple files into a single structured output (XML, JSON, or YAML)
-   Powerful file pattern matching and exclusion
-   Concurrent file processing for better performance
-   Size limits for both individual files and total output
-   Preserves file metadata and structure
-   Safe file handling with atomic writes
-   Cross-platform compatibility

## Installation

### Quick Install (Recommended)

Using curl:

```bash
curl -fsSL https://raw.githubusercontent.com/drgsn/filefusion/main/install.sh | bash
```

Using wget:

```bash
wget -qO- https://raw.githubusercontent.com/drgsn/filefusion/main/install.sh | bash
```

For users who prefer to inspect the script before running (recommended security practice):

```bash
curl -fsSL https://raw.githubusercontent.com/drgsn/filefusion/main/install.sh > install.sh
chmod +x install.sh
./install.sh
```

### Alternative Installation Methods

If you have Go installed, you can install FileFusion directly:

```bash
go install github.com/drgsn/filefusion/cmd/filefusion@latest
```

Or download the latest binary for your platform from the [releases page](https://github.com/drgsn/filefusion/releases).

## Default Values

-   **Pattern**: `*.go,*.json,*.yaml,*.yml`
-   **Max File Size**: 10MB
-   **Max Output Size**: 50MB
-   **Output Format**: XML (when not specified)
-   **Exclude**: none by default

## Basic Usage

### No Parameters (Current Directory)

Running FileFusion without any parameters processes the current directory:

```bash
filefusion
```

This will:

-   Process the current directory
-   Use default patterns (_.go,_.json,_.yaml,_.yml)
-   Generate an XML output file named after the directory
-   Use default size limits (10MB per file, 50MB total)

### Specific Directory

Process a specific directory:

```bash
filefusion /path/to/project
```

### Multiple Directories

Process multiple directories:

```bash
filefusion /path/to/project1 /path/to/project2
```

Each directory will get its own output file unless -o is specified.

## Flag Examples

### Output Path (-o, --output)

Specify the output file location and format:

```bash
# Generate XML output
filefusion -o output.xml /path/to/project

# Generate JSON output
filefusion -o output.json /path/to/project

# Generate YAML output
filefusion -o output.yaml /path/to/project
```

The output format is determined by the file extension.

### Pattern Matching Rules

-   Use `*` to match any sequence of characters in a filename
-   Use `**` in exclude patterns to match any number of subdirectories
-   Patterns are case-sensitive by default
-   Multiple patterns can be separated by commas
-   Exclude patterns take precedence over include patterns

### Pattern Examples

-   `*.go`: All Go files
-   `*.{go,proto}`: All Go and Proto files
-   `src/**/*.js`: All JavaScript files under src directory and its subdirectories
-   `!vendor/**`: Exclude all files in vendor directory and its subdirectories
-   `**/*_test.go`: Exclude all Go test files in any directory

### File Patterns (-p, --pattern)

Specify which files to include:

```bash
# Process only Python and JavaScript files
filefusion --pattern "*.py,*.js" /path/to/project

# Process all source files
filefusion -p "*.go,*.rs,*.js,*.py,*.java" /path/to/project

# Include configuration files
filefusion -p "*.yaml,*.json,*.toml,*.ini" /path/to/project
```

Patterns are comma-separated glob patterns. They match against file names, not paths.

### Exclusions (-e, --exclude)

Exclude specific files or directories:

```bash
# Exclude test files
filefusion --exclude "*_test.go,test/**" /path/to/project

# Exclude build and vendor directories
filefusion -e "build/**,vendor/**,node_modules/**" /path/to/project

# Complex exclusion
filefusion -e "**/*.test.js,**/__tests__/**,**/dist/**" /path/to/project
```

Exclusion patterns support:

-   File name patterns (\*.test.js)
-   Directory patterns (test/\*\*)
-   Full path patterns (**/dist/**)
-   Multiple patterns (comma-separated)

### Size Limits

Control file size limits:

```bash
# Increase individual file size limit to 20MB
filefusion --max-file-size 20MB /path/to/project

# Increase total output size limit to 100MB
filefusion --max-output-size 100MB /path/to/project

# Set both limits
filefusion --max-file-size 20MB --max-output-size 100MB /path/to/project
```

Size limits accept suffixes:

-   B (bytes)
-   KB (kilobytes)
-   MB (megabytes)
-   GB (gigabytes)
-   TB (terabytes)

## Advanced Examples

### Processing a Go Project

```bash
filefusion \
  --pattern "*.go" \
  --exclude "*_test.go,vendor/**" \
  --output project.json \
  --max-file-size 5MB \
  /path/to/go/project
```

### Processing Web Project Files

```bash
filefusion \
  --pattern "*.js,*.ts,*.jsx,*.tsx,*.css,*.html" \
  --exclude "node_modules/**,dist/**,build/**" \
  --output web-project.xml \
  /path/to/web/project
```

### Processing Documentation

```bash
filefusion \
  --pattern "*.md,*.txt,*.rst" \
  --exclude "node_modules/**,vendor/**" \
  --max-file-size 1MB \
  --output docs.yaml \
  /path/to/docs
```

### Complex Multi-Language Project

```bash
filefusion \
  --pattern "*.go,*.py,*.js,*.java,*.json,*.yaml" \
  --exclude "**/*_test.go,**/test/**,**/tests/**,vendor/**,node_modules/**" \
  --max-file-size 10MB \
  --max-output-size 100MB \
  --output project-analysis.xml \
  /path/to/project
```

## Output Format Examples

### XML Output Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<documents>
  <document index="1">
    <source>main.go</source>
    <document_content>
      package main
      ...
    </document_content>
  </document>
  <document index="2">
    <source>config.json</source>
    <document_content>
      {
        "key": "value"
      }
    </document_content>
  </document>
</documents>
```

### JSON Output Structure

```json
{
    "documents": [
        {
            "index": 1,
            "source": "main.go",
            "document_content": "package main\n..."
        },
        {
            "index": 2,
            "source": "config.json",
            "document_content": "{\n  \"key\": \"value\"\n}"
        }
    ]
}
```

### YAML Output Structure

```yaml
documents:
    - index: 1
      source: main.go
      document_content: |
          package main
          ...
    - index: 2
      source: config.json
      document_content: |
          {
            "key": "value"
          }
```

## Tips and Best Practices

1. **Start Small**: Begin with specific patterns and add more as needed
2. **Use Exclusions**: Always exclude build directories and dependency folders
3. **Monitor Size**: Check the reported total size before processing
4. **Format Choice**:
    - Use XML for most LLM interactions
    - Use JSON for programmatic processing
    - Use YAML for human readability
5. **Path Handling**: Use relative paths when possible for portability

## Common Issues and Solutions

1. **"no files found matching pattern"**

    - Check if patterns match your file extensions
    - Verify files exist in the specified directory
    - Make sure patterns don't conflict with exclusions

2. **"output size exceeds maximum"**

    - Increase --max-output-size
    - Use more specific patterns to reduce included files
    - Split processing into multiple runs

3. **"error processing files"**
    - Check file permissions
    - Verify file encodings (UTF-8 recommended)
    - Ensure sufficient disk space

## License

[Mozilla Public License Version 2.0](LICENSE)
