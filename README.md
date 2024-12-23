# FileFusion

FileFusion is a powerful command-line tool designed to concatenate and process files in a format optimized for Large Language Models (LLMs). It automatically preserves file metadata and structures the output in XML, JSON, or YAML format.

[![Run tests and upload coverage](https://github.com/drgsn/filefusion/actions/workflows/test.yml/badge.svg)](https://github.com/drgsn/filefusion/actions/workflows/test.yml)
[![Release](https://github.com/drgsn/filefusion/actions/workflows/release.yml/badge.svg)](https://github.com/drgsn/filefusion/actions/workflows/release.yml)

## Features

- Multiple file pattern matching with support for exclusions
- Concurrent file processing for improved performance
- Size limit enforcement for individual files
- Support for XML, JSON, and YAML output formats
- Directory traversal with customizable pattern matching
- Automatic handling of hidden directories and files
- Progress reporting and error handling

## Installation

### Using Go Install

If you have Go installed, you can install FileFusion directly:

```bash
go install github.com/drgsn/filefusion/cmd/filefusion@latest
```

### From Releases

Download the latest binary for your platform from the [releases page](https://github.com/drgsn/filefusion/releases).

## Basic Usage

```bash
filefusion [flags] [paths...]
```

### Flags

- `-o, --output`: Output file path (optional)
- `-p, --pattern`: File patterns to match (comma-separated)
- `-e, --exclude`: Patterns to exclude (comma-separated)
- `--max-size`: Maximum file size (e.g., "10MB")

## Examples

### 1. Basic File Processing

Process all Go and JSON files in the current directory:

```bash
filefusion . -p "*.go,*.json" -o output.xml
```

### 2. Multiple Directories

Process multiple directories and generate separate outputs:

```bash
filefusion ./service1 ./service2 ./service3 -p "*.go,*.yaml"
```

This will create:
- service1.xml
- service2.xml
- service3.xml

### 3. Exclude Patterns

Process files while excluding specific patterns:

```bash
filefusion . -p "*.go" -e "vendor/**,**/*_test.go" -o output.xml
```

### 4. Size Limits

Set maximum file size limit:

```bash
filefusion . -p "*.go" --max-size 5MB -o output.xml
```

### 5. Different Output Formats

#### XML Output (Default)
```bash
filefusion . -p "*.go" -o output.xml
```

Example output:
```xml
<documents>
  <document index="1">
    <source>main.go</source>
    <document_content>package main...</document_content>
  </document>
</documents>
```

#### JSON Output
```bash
filefusion . -p "*.go" -o output.json
```

Example output:
```json
{
  "documents": [
    {
      "index": 1,
      "source": "main.go",
      "document_content": "package main..."
    }
  ]
}
```

#### YAML Output
```bash
filefusion . -p "*.go" -o output.yaml
```

Example output:
```yaml
documents:
  - index: 1
    source: main.go
    document_content: |
      package main...
```

### 6. Complex Pattern Matching

Process specific file types while excluding certain directories:

```bash
filefusion . \
  -p "*.go,*.proto,*.yaml" \
  -e "vendor/**,**/generated/**,**/*_test.go" \
  --max-size 2MB \
  -o project.xml
```

### 7. Processing Large Projects

For large projects with many files:

```bash
filefusion . \
  -p "*.go,*.js,*.ts,*.proto" \
  -e "node_modules/**,vendor/**,**/dist/**" \
  --max-size 5MB \
  -o project.xml
```

## Pattern Matching Rules

- Use `*` to match any sequence of characters in a filename
- Use `**` in exclude patterns to match any number of subdirectories
- Patterns are case-sensitive by default
- Multiple patterns can be separated by commas
- Exclude patterns take precedence over include patterns

### Pattern Examples

- `*.go`: All Go files
- `*.{go,proto}`: All Go and Proto files
- `src/**/*.js`: All JavaScript files under src directory and its subdirectories
- `!vendor/**`: Exclude all files in vendor directory and its subdirectories
- `**/*_test.go`: Exclude all Go test files in any directory

## Size Specifications

File size limits can be specified using the following units:
- B (Bytes)
- KB (Kilobytes)
- MB (Megabytes)
- GB (Gigabytes)
- TB (Terabytes)

Examples:
- `--max-size 500KB`
- `--max-size 10MB`
- `--max-size 1GB`

## Error Handling

FileFusion provides detailed error messages and warnings:
- Files exceeding size limits are skipped with warnings
- Invalid patterns generate appropriate error messages
- Permission issues are reported with specific details
- Missing directories or files are properly handled

## Best Practices

1. **Start Small**:
   - Begin with specific patterns and add more as needed
   - Test with smaller directories first

2. **Use Exclusions Wisely**:
   - Exclude build directories, dependencies, and generated files
   - Use `**` in exclude patterns to match nested directories

3. **Monitor Output Size**:
   - Use appropriate size limits for your use case
   - Consider splitting large projects into smaller chunks

4. **Choose Output Format**:
   - Use XML for better readability
   - Use JSON for better compatibility with other tools
   - Use YAML for fun of it

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Mozilla Public License Version 2.0
 - see the LICENSE file for details.
