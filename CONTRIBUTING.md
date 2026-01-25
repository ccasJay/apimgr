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
- Include table-driven tests for multiple test cases
- Use meaningful test names that describe what is being tested

### Commit Messages
- Use conventional commit format: `type(scope): description`
- Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
- Example: `feat(ping): add streaming support for compatibility tests`
- Keep the first line under 72 characters
- Add detailed description in the commit body if needed

### Documentation
- Update README.md for new commands or features
- Write detailed usage documentation for complex features
- Keep documentation consistent with code

## Community Guidelines

Please follow the [Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/) code of conduct.

## Development Environment

### Dependencies
- Go 1.21 or higher
- golangci-lint (for code linting)
- goreleaser (optional, for release builds)

### Setup Instructions

1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-username/apimgr.git
   cd apimgr
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Build the Project**
   ```bash
   make build
   # or
   go build -o apimgr .
   ```

4. **Run Tests**
   ```bash
   go test ./...
   ```

5. **Run Linter**
   ```bash
   golangci-lint run
   # or use the project configuration
   golangci-lint run --config .golangci.yml
   ```

### Common Commands
```bash
# Build binary
make build
# or
go build -o apimgr .

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestConfigLoad ./config

# Run benchmarks
go test -bench=. ./...

# Run lint checks
golangci-lint run

# Clean build artifacts
make clean
# or
rm -f apimgr

# Install locally
make install
```

### Project Structure
```
apimgr/
├── cmd/           # CLI commands
├── config/        # Configuration management
├── internal/      # Internal packages
│   ├── providers/ # API provider implementations
│   ├── tui/       # Terminal UI
│   └── utils/     # Utility functions
├── main.go        # Entry point
└── ...
```

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md).

## License
apimgr is licensed under the MIT License, and your contributions will automatically be covered by the same license.
