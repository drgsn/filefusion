# FileFusion Pattern Guide 🎯

<div align="center">

**A comprehensive guide to FileFusion's file pattern matching system**

[Input Patterns](#input-patterns) • [Exclusion Patterns](#exclusion-patterns) • [Combined Examples](#combining-patterns-and-exclusions)

</div>

## 📋 Input Patterns

### General Patterns

#### Match all `.go` files:

-   **Pattern**: `*.go`
-   **Matches**:
    -   `main.go`
    -   `handlers/utils.go`
    -   `core/main.go`

#### Match `.json` and `.yaml` files:

-   **Pattern**: `*.json,*.yaml`
-   **Matches**:
    -   `config.json`
    -   `settings.yaml`
    -   `nested/data.json`

#### Recursive match for all `.txt` files in nested directories:

-   **Pattern**: `*.txt`
-   **Matches**:
    -   `logs/errors.txt`
    -   `nested/directory/notes.txt`

## 🚫 Exclusion Patterns

### Examples

#### Exclude a specific directory:

-   **Exclusion**: `build/**`
-   **Matches files in**:
    -   `src/main.go`
-   **Does not match**:
    -   `build/output.json`
    -   `build/logs/errors.log`

#### Exclude a file type globally:

-   **Exclusion**: `*.log`
-   **Matches**:
    -   `main.go`
    -   `config.json`
-   **Does not match**:
    -   `app.log`
    -   `nested/errors.log`

#### Exclude files in a specific subdirectory:

-   **Exclusion**: `logs/*.txt`
-   **Matches**:
    -   `logs/errors.log`
    -   `main.go`
-   **Does not match**:
    -   `logs/errors.txt`

#### Exclude hidden directories:

-   **Exclusion**: `**/.hidden/**`
-   **Matches**:
    -   `src/file.go`
-   **Does not match**:
    -   `.hidden/file.txt`
    -   `nested/.hidden/file.txt`

#### Exclude multiple directories:

-   **Exclusion**: `build/**,node_modules/**`
-   **Matches**:
    -   `src/main.go`
-   **Does not match**:
    -   `build/main.o`
    -   `node_modules/lib/index.js`

#### Exclude specific files:

-   **Exclusion**: `README.md`
-   **Matches**:
    -   `main.go`
-   **Does not match**:
    -   `README.md`

#### Exclude files with special characters:

-   **Exclusion**: `**/#*`
-   **Matches**:
    -   `main.go`
-   **Does not match**:
    -   `#tempfile.txt`

## 🔄 Combining Patterns and Exclusions

### Example 1

-   **Input Pattern**: `*.go,*.json`
    -   **Matches**:
        -   `main.go`
        -   `config.json`
-   **Exclusion**: `build/**,test/**`
    -   **Does not match**:
        -   `build/main.go`
        -   `test/utils_test.go`
-   **Resulting Files**:
    -   `src/main.go`
    -   `src/config.json`

### Example 2

-   **Input Pattern**: `**/*.txt`
-   **Exclusion**: `logs/**`
    -   **Matches**:
        -   `docs/readme.txt`
    -   **Does not match**:
        -   `logs/errors.txt`

## 💻 Command Examples

### Basic Go Project

```bash
# Include all Go-related files
filefusion -p "*.go,*.mod,*.sum" .

# Include Go files but exclude tests
filefusion -p "*.go" -e "*_test.go" .
```

### Web Project

```bash
# Process frontend source files
filefusion -p "*.{js,ts,jsx,tsx,css,scss}" /src

# Include configuration but exclude build artifacts
filefusion -p "*.{js,json,yaml,env}" -e "dist/**,build/**" 

# Process only React components
filefusion -p "*.{jsx,tsx}" src/components/
```

### Complex Patterns

```bash
# Multiple file types and exclusions
filefusion \
  -p "*.go,*.yaml,*.json" \
  -e "**/*_test.go,vendor/**,**/testdata/**" \
  internal cmd
# will generate 2 output files internal.xml and cmd.xml

# Specific directory patterns with multiple exclusions
filefusion \
  -p "*.{js,ts}, *.{json,yaml},*.sh" \
  -e "**/*.test.{js,ts},**/__tests__/**,**/node_modules/**" 

# Documentation and configuration files
filefusion \
  -p "*.md,*.{yaml,yml,json}" \
  -e "**/draft/**,**/.git/**,private/**" \
  .
```

### Language-Specific Examples

```bash
# Python project
filefusion \
  -p "*.{py,ipynb},requirements.txt,setup.py" \
  -e "**/__pycache__/**,**/*.pyc,venv/**" \
  .

# Java/Kotlin project
filefusion \
  -p "*.{java,kt},build.gradle,pom.xml" \
  -e "**/build/**,**/target/**,**/*Test.{java,kt}" \
  . src

# Full-stack project
filefusion \
  -p "*.go,*.{ts,tsx},*.yaml" \
  -e "**/node_modules/**,**/dist/**,**/*_test.go" \
  . backend frontend/src
```

## ⚠️ Notes on Validation

### Invalid Input Patterns

-   **Pattern**: `***`
    -   **Error**: "invalid pattern: contains invalid glob pattern '\*\*\*'"

### Glob Confusion 

-   **Valid For Exclusion**: `**/file.*`, `**/*.txt`
-   **Invalid**: `***/*.txt` (returns an error)

---

<div align="center">
Made with ❤️ by the DrGos
</div>
