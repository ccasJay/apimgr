# Contributing Guide

[中文版](CONTRIBUTING.zh.md)

Thank you for your interest in contributing to apimgr! We welcome all forms of contribution.

## Ways to Contribute

### 1. Report Issues
If you find a bug or have a feature request, please submit it through GitHub Issues. When submitting, please include:
- Clear issue description
- Reproduction steps (for bugs)
- Expected behavior
- Actual behavior
- Environment information (OS, Go version, etc.)

### 2. Submit Pull Requests
1. **Fork the repository**
2. **Clone your fork**: `git clone https://github.com/your-username/apimgr.git`
3. **Create a branch**: `git checkout -b feature/your-feature-name`
4. **Implement changes**: Write code following the project's coding style
5. **Run tests**: `go test ./...`
6. **Run lint**: `golangci-lint run`
7. **Commit changes**: `git commit -m "feat: add your feature"`
8. **Push branch**: `git push origin feature/your-feature-name`
9. **Create PR**: Create a Pull Request on GitHub

## Development Workflow

### Code Style
- Follow Go's official coding style (`go fmt`)
- Use `golint` and `staticcheck` for code checking
- Write clear comments, especially for public functions and structs
- Use camelCase naming, avoid abbreviations

### Testing
- Write unit tests for new features
- Ensure test coverage is at least 80%
- Run `go test ./...` to ensure all tests pass

### Documentation
- Update README.md for new commands or features
- Write detailed usage documentation for complex features
- Keep documentation consistent with code

## Community Guidelines

Please follow the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) code of conduct.

## Development Environment

### Dependencies
- Go 1.21+
- golangci-lint
- goreleaser (for release)

### Common Commands
```bash
# Run all tests
go test ./...

# Run lint checks
golangci-lint run

# Build binary
go build

# Clean build files
go clean
```

## License
apimgr is licensed under the MIT License, and your contributions will automatically be covered by the same license.
