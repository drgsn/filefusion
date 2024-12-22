# FileFusion

FileFusion is a powerful file concatenation tool designed specifically for Large Language Model (LLM) applications. 
It combines multiple files into a single structured output file while preserving metadata 
and maintaining a format that's optimal for LLM processing.

## Features

- Concatenates multiple files into XML, JSON, or YAML formats
- Supports flexible file pattern matching
- Excludes unwanted files or directories
- Enforces file size limits
- Preserves file metadata and structure
- Handles recursive directory scanning
- Platform-independent path handling

## Installation

```bash
go install github.com/drgsn/filefusion@latest
```

## Usage

```bash
filefusion [flags]
```

### Flags

- `-i, --input`: Input directory path (default: current directory)
- `-o, --output`: Output file path (default: output.xml)
- `-p, --pattern`: Comma-separated file patterns (default: "*.go,*.json,*.yaml,*.yml")
- `-e, --exclude`: Comma-separated patterns to exclude
- `--max-size`: Maximum size of the output file (default: 10MB)

## Examples

### Basic Usage

1. Process all Go files in the current directory:
```bash
filefusion -p "*.go"
```

2. Process multiple file types:
```bash
filefusion -p "*.go,*.js,*.py" -o output.xml
```

3. Process files from a specific directory:
```bash
filefusion -i /path/to/project -p "*.go"
```

### Output Formats

1. Generate XML output (default):
```bash
filefusion -p "*.go" -o output.xml
```

2. Generate JSON output:
```bash
filefusion -p "*.go" -o output.json
```

3. Generate YAML output:
```bash
filefusion -p "*.go" -o output.yaml
```

### File Size Limits

1. Set a 5MB limit for the output file:
```bash
filefusion -p "*.go" --max-size 5MB
```

2. Use different size units:
```bash
filefusion -p "*.go" --max-size 500KB
filefusion -p "*.go" --max-size 1GB
```

### Exclusion Patterns

1. Exclude specific directories:
```bash
filefusion -p "*.go" -e "vendor/**,build/**"
```

2. Exclude specific files:
```bash
filefusion -p "*.go" -e "*_test.go"
```

3. Complex exclusion patterns:
```bash
filefusion -p "*.go,*.js" -e "vendor/**,**/*_test.go,**/node_modules/**"
```

## Output Format Examples

### XML Output
```xml
<documents>
  <document index="1">
    <source>main.go</source>
    <document_content>package main
    
func main() {
    // ...
}</document_content>
  </document>
  <!-- Additional documents... -->
</documents>
```

### JSON Output
```json
{
  "documents": [
    {
      "index": 1,
      "source": "main.go",
      "document_content": "package main\n\nfunc main() {\n    // ...\n}"
    }
  ]
}
```

### YAML Output
```yaml
documents:
  - index: 1
    source: main.go
    document_content: |
      package main
      
      func main() {
          // ...
      }
```

## Size Units

The `--max-size` flag supports the following units:
- B (Bytes)
- KB (Kilobytes)
- MB (Megabytes)
- GB (Gigabytes)
- TB (Terabytes)

Examples:
- `--max-size 500B`
- `--max-size 1KB`
- `--max-size 10MB`
- `--max-size 1GB`

## File Pattern Syntax

FileFusion uses standard glob patterns:
- `*`: Matches any sequence of characters except path separators
- `**`: Matches any sequence of characters including path separators (for directory exclusions)
- `?`: Matches any single character
- `[abc]`: Matches one character given in the bracket
- `[a-z]`: Matches one character from the range given in the bracket

## Error Handling

FileFusion provides clear error messages for common issues:
- File size exceeding limits
- Invalid patterns
- Missing files
- Permission issues
- Invalid output formats

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.