# Contributing to FileFusion 🚀

First off, thank you for considering contributing to FileFusion! It's people like you that make OSS Community great.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct. Please report unacceptable behavior to [project maintainers].

## How Can I Contribute?

### Reporting Bugs 🐛

Before creating bug reports, please check the issue list as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

-   Use a clear and descriptive title
-   Describe the exact steps which reproduce the problem
-   Provide specific examples to demonstrate the steps
-   Describe the behavior you observed after following the steps
-   Explain which behavior you expected to see instead and why
-   Include details about your configuration and environment

### Suggesting Enhancements ✨

If you have a suggestion for the project, we want to hear it! Before creating enhancement suggestions, please check the issue list as you might find out that you don't need to create one. When you are creating an enhancement suggestion, please include as many details as possible:

-   Use a clear and descriptive title
-   Provide a step-by-step description of the suggested enhancement
-   Provide specific examples to demonstrate the steps
-   Describe the current behavior and explain which behavior you expected to see instead and why
-   Explain why this enhancement would be useful to most FileFusion users

### Pull Requests 💪

Please follow these steps to have your contribution considered by the maintainers:

1. Follow all instructions in the template
2. Follow the [style guides](#style-guides)
3. After you submit your pull request, verify that all status checks are passing

#### Local Development Setup

1. Fork the repo
2. Clone your fork

```bash
git clone https://github.com/your-username/filefusion.git
```

3. Set up your development environment:

```bash
cd filefusion
go mod download
```

4. Create a new branch:

```bash
git checkout -b feature/your-feature-name
```

### Style Guides 📝

#### Git Commit Messages 

Each commit message should consist of a clear description of the change, with a reference to the GitHub issue number if applicable.
Format: [gh-{issue-number}:] {commit description}

#### Examples:

-   gh-123: Add support for YAML output format
-   gh-45: Fix file size calculation for large files
-   Add contributing guidelines (when no related issue exists)

#### Guidelines:

-   Start with the issue reference if one exists
-   Use clear and concise descriptions
-   Keep the first line under 72 characters
-   Use the present tense
-   Add additional details in the commit body if needed

### Go Code Style Guide

-   Follow standard Go conventions and format your code with `gofmt`
-   Write descriptive variable and function names
-   Add comments for complex logic
-   Follow the principles in [Effective Go](https://golang.org/doc/effective_go.html)
-   Use `golangci-lint` for linting

### Documentation 📚

-   Update the README.md with details of changes to the interface
-   Update the Wiki with any necessary changes
-   Add or update code comments as needed

### Testing 🧪

-   Write test cases for new features
-   Ensure existing tests pass
-   Follow table-driven test patterns when appropriate
-   Aim for high test coverage

### Continuous Integration ⚙️

We use GitHub Actions for our CI pipeline. Ensure your changes:

-   Pass all existing tests
-   Pass lint checks
-   Meet coverage requirements
-   Don't introduce new warnings

## Community 🌍

-   Follow the project's development
-   Share FileFusion in your network 
-   Help others in the issues section

## Questions? 💭

Don't hesitate to ask questions about contributing. You can:

-   Open an issue with your question
-   Reach out to the maintainers
-   Join community discussions

---

Again, thank you for your interest in contributing to FileFusion! ❤️
