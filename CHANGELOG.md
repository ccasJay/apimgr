# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive SECURITY.md with security policy and best practices
- CHANGELOG.md for tracking version history
- ARCHITECTURE.md for technical documentation

### Changed
- Updated all documentation to use correct repository URLs
- Improved installation instructions in README.md
- Enhanced QUICKSTART.md with TUI mode and compatibility testing examples
- Updated Chinese documentation to match English version improvements

### Fixed
- Fixed incorrect `go install` command in README files
- Fixed broken GitHub URL references (replaced placeholder URLs)
- Fixed shell integration commands for better cross-platform compatibility
- Corrected `init` command references to `enable` command

## [1.0.0] - TBD

### Added
- Complete TUI (Terminal User Interface) with keyboard navigation
- Multi-provider support (Anthropic, OpenAI, custom providers)
- Connectivity testing with detailed diagnostics
- Global and local configuration switching
- Shell integration for environment variable export
- XDG Base Directory Specification compliance
- Cross-platform support (macOS, Linux, Windows)
- Interactive configuration editing
- Model selection and validation
- Compatibility testing mode for API validation
- Streaming API support testing
- Configuration backup and restore
- File locking for concurrent access protection
- Claude Code synchronization

### Features

#### Commands
- `apimgr` - Launch interactive TUI
- `apimgr add` - Add new API configuration
- `apimgr list` - List all configurations
- `apimgr switch` - Switch active configuration
- `apimgr ping` - Test API connectivity
- `apimgr status` - Show configuration status
- `apimgr edit` - Edit existing configuration
- `apimgr remove` - Remove configuration
- `apimgr enable` - Initialize and enable shell integration

#### TUI Keyboard Shortcuts
- Navigation: `j/k`, `↑/↓`, `g/G`
- Actions: `s` (local switch), `S` (global switch)
- Operations: `a` (add), `e` (edit), `d` (delete)
- Testing: `p` (ping), `t` (compatibility test)
- Other: `m` (switch model), `?` (help), `q` (quit)

#### Configuration Management
- JSON-based configuration storage
- API key masking for security
- Configuration validation
- Provider auto-detection based on URL
- Model compatibility validation

#### Testing Features
- Basic connectivity testing (HEAD/GET requests)
- Compatibility testing with real API requests
- Streaming API support verification
- Detailed error diagnostics
- JSON output for automation

### Technical Details
- **Language**: Go 1.21+
- **CLI Framework**: Cobra
- **TUI Framework**: Bubbletea
- **Configuration**: JSON with XDG compliance
- **Dependencies**: Minimal and well-maintained

### Documentation
- Comprehensive README with examples
- Quick Start Guide
- Contributing Guidelines
- Code of Conduct
- Code Audit Report
- Security Policy

### Supported Platforms
- macOS (ARM64, AMD64)
- Linux (ARM64, AMD64)
- Windows (AMD64)

---

## Development History

This section provides context for the project evolution.

### Early Development
- Initial project structure and CLI framework setup
- Basic configuration CRUD operations
- Simple configuration switching

### Feature Expansion
- Added TUI interface with Bubbletea
- Implemented multi-provider support
- Added connectivity testing
- Integrated Claude Code synchronization

### Quality Improvements
- Comprehensive test coverage
- Code linting and formatting
- Security enhancements
- Documentation improvements

### Future Roadmap
See [GitHub Issues](https://github.com/ccasJay/apimgr/issues) for planned features and improvements.

---

## Notes

### Versioning
This project uses [Semantic Versioning](https://semver.org/):
- **MAJOR** version for incompatible API changes
- **MINOR** version for new functionality in a backward compatible manner
- **PATCH** version for backward compatible bug fixes

### Contributing
See [CONTRIBUTING.md](CONTRIBUTING.md) for how to contribute to this changelog.

### Questions
For questions about releases or changes, please open an issue on GitHub.

---

Last Updated: 2025-01-25
