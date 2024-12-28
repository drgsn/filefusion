# FileFusion 🚀

<div align="center">

[![Test Coverage](https://codecov.io/gh/drgsn/filefusion/branch/main/graph/badge.svg)](https://codecov.io/gh/drgsn/filefusion)
[![Release](https://github.com/drgsn/filefusion/actions/workflows/release.yml/badge.svg)](https://github.com/drgsn/filefusion/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/drgsn/filefusion)](https://goreportcard.com/report/github.com/drgsn/filefusion)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)

**FileFusion is a powerful command-line tool designed to concatenate and process files in a format optimized for Large Language Models (LLMs).**

[Installation](#installation) • [Features](#features) • [Usage](#basic-usage) • [Examples](#examples) • [Documentation](#documentation)

</div>

## ✨ Features

-   📦 **Multiple Output Formats** - XML, JSON, or YAML with preserved metadata
-   🎯 **Smart Pattern Matching** - Powerful glob patterns for precise file selection ([see patterns guide](docs/patterns.md))
-   ⚡️ **Concurrent Processing** - Parallel file processing with safety limits
-   📊 **Size Control** - File and output size limits with detailed reporting
-   🧹 **Code Cleaning** - Remove comments and optimize while preserving docs
-   🔒 **Safe Operations** - Atomic writes and thorough error checking

## 🚀 Installation

### Quick Install (Recommended)

Using curl:

```bash
curl -fsSL https://raw.githubusercontent.com/drgsn/filefusion/main/install.sh | bash
```

Using wget:

```bash
wget -qO- https://raw.githubusercontent.com/drgsn/filefusion/main/install.sh | bash
```

### Safe Install (Recommended Security Practice)

```bash
# Download and inspect the script first
curl -fsSL https://raw.githubusercontent.com/drgsn/filefusion/main/install.sh > install.sh
chmod +x install.sh
./install.sh
```

### Alternative Methods

Using Go:

```bash
go install github.com/drgsn/filefusion/cmd/filefusion@latest
```

Or download the latest binary for your platform from the [releases page](https://github.com/drgsn/filefusion/releases).

## 📋 Default Values

| Setting         | Default Value              | Description                |
| --------------- | -------------------------- | -------------------------- |
| Pattern         | `*.go,*.json,*.yaml,*.yml` | Default file patterns      |
| Max File Size   | 10MB                       | Individual file size limit |
| Max Output Size | 50MB                       | Total output size limit    |
| Output Format   | XML                        | When not specified         |
| Exclude         | none                       | No default exclusions      |
| Dry Run         | disabled                   | Show files to be processed |

## 🎯 Basic Usage

### No Parameters (Current Directory)

```bash
filefusion
```

This will:

-   Process the current directory
-   Use default patterns (_.go,_.json,_.yaml,_.yml)
-   Generate an XML output file named after the directory
-   Use default size limits (10MB per file, 50MB total)

### Specific Directory

```bash
filefusion /path/to/project
```

### Multiple Directories

```bash
filefusion /path/to/project1 /path/to/project2
```

## 🛠️ Flag Examples

### Output Path (-o, --output)

```bash
# Generate XML output
filefusion -o output.xml /path/to/project

# Generate JSON output
filefusion -o output.json /path/to/project

# Generate YAML output
filefusion -o output.yaml /path/to/project
```

### Pattern Matching Rules

For detailed pattern matching examples and rules, please refer to our [Pattern Guide](docs/patterns.md).

Here are some common patterns:

| Pattern        | Description                    |
| -------------- | ------------------------------ |
| `*.go`         | All Go files                   |
| `*.{go,proto}` | All Go and Proto files         |
| `src/**/*.js`  | All JavaScript files under src |
| `!vendor/**`   | Exclude vendor directory       |
| `**/*_test.go` | All Go test files              |

### File Patterns (-p, --pattern)

```bash
# Process only Python and JavaScript files
filefusion --pattern "*.py,*.js" /path/to/project

# Process all source files
filefusion -p "*.go,*.rs,*.js,*.py,*.java" /path/to/project

# Include configuration files
filefusion -p "*.yaml,*.json,*.toml,*.ini" /path/to/project
```

### Exclusions (-e, --exclude)

```bash
# Exclude test files
filefusion --exclude "*_test.go,test/**" /path/to/project

# Exclude build and vendor directories
filefusion -e "build/**,vendor/**,node_modules/**" /path/to/project

# Complex exclusion
filefusion -e "**/*.test.js,**/*tests*/**,**/dist/**" /path/to/project
```

### Size Limits

```bash
# Increase individual file size limit to 20MB
filefusion --max-file-size 20MB /path/to/project

# Increase total output size limit to 100MB
filefusion --max-output-size 100MB /path/to/project

# Set both limits and enable cleaning
filefusion --max-file-size 20MB --max-output-size 100MB --clean /path/to/project
```

Size limits accept suffixes: `B`, `KB`, `MB`, `GB`, `TB`

## 📚 Code Cleaning

FileFusion includes a powerful code cleaning engine that optimizes files for LLM processing while preserving functionality. The cleaner supports multiple programming languages and offers various optimization options.

### Supported Languages

-   Go, Java, Python, Swift, Kotlin
-   JavaScript, TypeScript, HTML, CSS
-   C++, C#, PHP, Ruby
-   SQL, Bash

### Cleaning Options

| Option                           | Description                  | Default |
| -------------------------------- | ---------------------------- | ------- |
| `--clean`                        | Enable code cleaning         | false   |
| `--clean-remove-comments`        | Remove all comments          | true    |
| `--clean-preserve-doc-comments`  | Keep documentation comments  | true    |
| `--clean-remove-imports`         | Remove import statements     | false   |
| `--clean-remove-logging`         | Remove logging statements    | true    |
| `--clean-remove-getters-setters` | Remove getter/setter methods | true    |
| `--clean-optimize-whitespace`    | Optimize whitespace          | true    |

### Cleaning Examples

```bash
# Basic cleaning with default options
filefusion --clean input.go -o clean.xml

# Preserve all comments
filefusion --clean --clean-remove-comments=false input.py -o clean.xml

# Remove everything except essential code
filefusion --clean \
  --clean-remove-comments \
  --clean-preserve-doc-comments=false \
  --clean-remove-logging \
  --clean-remove-getters-setters \
  input.java -o clean.xml

# Clean TypeScript while preserving docs
filefusion --clean \
  --clean-preserve-doc-comments \
  --clean-remove-logging \
  --pattern "*.ts" \
  src/ -o clean.xml

# Clean multiple languages in a project
filefusion --clean \
  --pattern "*.{go,js,py}" \
  --clean-preserve-doc-comments \
  --clean-remove-logging \
  project/ -o clean.xml
```

### Language-Specific Features

The cleaner automatically detects and handles language-specific patterns:

-   **Logging Statements**: Recognizes common logging patterns

    -   Go: `log.`, `logger.`
    -   JavaScript/TypeScript: `console.`, `logger.`
    -   Python: `logging.`, `logger.`, `print`
    -   Java: `Logger.`, `System.out.`, `System.err.`
    -   And more...

-   **Documentation**: Preserves language-specific doc formats

    -   Go: `//` and `/* */` doc comments
    -   Python: Docstrings
    -   JavaScript/TypeScript: JSDoc
    -   Java: Javadoc

-   **Code Structure**: Maintains language idioms while removing noise
    -   Preserves package/module structure
    -   Keeps essential imports
    -   Removes debug/test code

## 📚 Advanced Examples

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

### Code Cleaning and Size Optimization

```bash
# Clean and optimize a Go project
filefusion \
  --pattern "*.go" \
  --exclude "*_test.go" \
  --clean \
  --clean-remove-comments \
  --clean-remove-logging \
  --output optimized.xml \
  /path/to/go/project

# Clean TypeScript/JavaScript with preserved documentation
filefusion \
  --pattern "*.ts,*.js" \
  --clean \
  --clean-preserve-doc-comments \
  --clean-remove-logging \
  --clean-optimize-whitespace \
  --output web-optimized.xml \
  /path/to/web/project
```

## 📄 Output Format Examples

### XML Output

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
</documents>
```

### JSON Output

```json
{
    "documents": [
        {
            "index": 1,
            "source": "main.go",
            "document_content": "package main\n..."
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
          ...
```

## 💡 Tips and Best Practices

1. **Start Small**

    - Begin with specific patterns
    - Add more patterns as needed
    - Test with `--dry-run` first

2. **Use Exclusions**

    - Always exclude build directories
    - Exclude dependency folders
    - Use specific patterns for tests

3. **Monitor Size**

    - Check reported total size
    - Use `--clean` for size reduction
    - Set appropriate limits

4. **Format Choice**

    - XML: Best for most LLM interactions
    - JSON: For programmatic processing
    - YAML: For human readability

5. **Path Handling**
    - Use relative paths when possible
    - Be specific with patterns
    - Test patterns before processing

## ❗ Issues and Solutions

### "no files found matching pattern"

-   Check if patterns match your file extensions
-   Verify files exist in the specified directory
-   Make sure patterns don't conflict with exclusions

### "output size exceeds maximum"

-   Increase `--max-output-size`
-   Use more specific patterns
-   Split processing into multiple runs

### "error processing files"

-   Check file permissions
-   Verify file encodings (UTF-8 recommended)
-   Ensure sufficient disk space

## 📜 License

[Mozilla Public License Version 2.0](LICENSE)

---

<div align="center">
Made with ❤️ by the DrGos
</div>
