# Architecture Documentation

This document provides a technical overview of the apimgr architecture, design decisions, and implementation details.

## Table of Contents
- [Overview](#overview)
- [Project Structure](#project-structure)
- [Core Components](#core-components)
- [Data Flow](#data-flow)
- [Configuration Management](#configuration-management)
- [Provider System](#provider-system)
- [TUI Architecture](#tui-architecture)
- [Security Design](#security-design)
- [Performance Considerations](#performance-considerations)
- [Platform Support](#platform-support)

## Overview

apimgr is a command-line tool for managing API configurations with a focus on:
- **Simplicity**: Easy-to-use CLI and TUI interfaces
- **Security**: Secure storage and handling of API keys
- **Flexibility**: Support for multiple API providers
- **Reliability**: Robust error handling and file locking

### Technology Stack
- **Language**: Go 1.21+
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **TUI Framework**: [Bubbletea](https://github.com/charmbracelet/bubbletea)
- **TUI Components**: [Bubbles](https://github.com/charmbracelet/bubbles)
- **Styling**: [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **JSON Processing**: [gjson](https://github.com/tidwall/gjson) and [sjson](https://github.com/tidwall/sjson)

## Project Structure

```
apimgr/
├── main.go                     # Application entry point
├── cmd/                        # CLI commands
│   ├── root.go                # Root command setup
│   ├── add.go                 # Add configuration
│   ├── list.go                # List configurations
│   ├── switch.go              # Switch configuration
│   ├── ping.go                # Connectivity testing
│   ├── status.go              # Status display
│   ├── edit.go                # Edit configuration
│   ├── remove.go              # Remove configuration
│   ├── enable.go              # Shell integration setup
│   ├── model_selection.go     # Model selection UI
│   └── ...                    # Other commands and tests
├── config/                     # Configuration management
│   ├── config.go              # Core configuration manager
│   ├── model_validator.go     # Model validation
│   ├── lock_unix.go           # File locking (Unix)
│   ├── lock_windows.go        # File locking (Windows)
│   └── ...                    # Other config utilities
├── internal/                   # Internal packages
│   ├── providers/             # API provider implementations
│   │   ├── provider.go        # Provider interface
│   │   ├── anthropic.go       # Anthropic provider
│   │   ├── openai.go          # OpenAI provider
│   │   └── ...
│   ├── compatibility/         # API compatibility testing
│   │   ├── tester.go          # Compatibility test logic
│   │   └── ...
│   ├── tui/                   # Terminal UI components
│   │   ├── model.go           # TUI model
│   │   ├── view.go            # TUI views
│   │   └── ...
│   └── utils/                 # Utility functions
│       ├── mask.go            # API key masking
│       └── ...
├── go.mod                      # Go module definition
└── go.sum                      # Dependency checksums
```

## Core Components

### 1. Configuration Manager (`config/config.go`)

The configuration manager is the heart of apimgr, responsible for:
- Loading and saving configuration files
- Managing multiple API configurations
- Handling active configuration state
- File locking for concurrent access
- XDG Base Directory compliance
- Claude Code synchronization

**Key Structures**:
```go
type Manager struct {
    configPath string
    mu         sync.Mutex
}

type File struct {
    Active  string      `json:"active"`
    Configs []APIConfig `json:"configs"`
}

type APIConfig struct {
    Alias      string `json:"alias"`
    APIKey     string `json:"api_key"`
    AuthToken  string `json:"auth_token"`
    BaseURL    string `json:"base_url"`
    Model      string `json:"model"`
    Provider   string `json:"provider"`
}
```

**Design Decisions**:
- **JSON Format**: Human-readable and easily editable
- **Mutex Locking**: Prevents concurrent modification conflicts
- **XDG Compliance**: Follows Linux desktop standards
- **Atomic Writes**: Uses temporary files and renames for data integrity

### 2. Command Layer (`cmd/`)

Each command is implemented as a separate file using Cobra:
- **Independence**: Commands are self-contained
- **Testability**: Each command has corresponding tests
- **Consistency**: Shared flags and error handling patterns

**Command Pattern**:
```go
var cmdName = &cobra.Command{
    Use:   "name",
    Short: "Brief description",
    Long:  "Detailed description",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Command implementation
        return nil
    },
}
```

### 3. Provider System (`internal/providers/`)

Abstraction layer for different API providers:

**Interface**:
```go
type Provider interface {
    CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error)
    CreateChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (*http.Response, error)
    ValidateConnection(ctx context.Context) error
}
```

**Implementations**:
- **AnthropicProvider**: Anthropic API (Claude models)
- **OpenAIProvider**: OpenAI API (GPT models)
- **Custom Providers**: Extensible for future providers

**Benefits**:
- **Polymorphism**: Uniform interface for different providers
- **Extensibility**: Easy to add new providers
- **Testability**: Can mock providers for testing

### 4. TUI System (`internal/tui/`)

Built with Bubbletea following the Elm Architecture:

**Model-View-Update Pattern**:
```go
type Model struct {
    configs     []config.APIConfig
    cursor      int
    state       ViewState
    // ... other fields
}

func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

**View States**:
- List View: Browse configurations
- Detail View: Show configuration details
- Help View: Display keyboard shortcuts
- Input View: Interactive input for add/edit

## Data Flow

### Configuration Loading
```
User Command
    ↓
cmd/command.go
    ↓
config.Manager.Load()
    ↓
File System (JSON)
    ↓
Parse & Validate
    ↓
Return Config
```

### Configuration Switching
```
User: apimgr switch my-config
    ↓
cmd/switch.go
    ↓
config.Manager.SetActive()
    ↓
Update config.json
    ↓
Generate active.env
    ↓
Sync to Claude Code (if applicable)
    ↓
Success
```

### Connectivity Testing
```
User: apimgr ping -T
    ↓
cmd/ping.go
    ↓
providers.NewProvider()
    ↓
compatibility.TestAPI()
    ↓
HTTP Request
    ↓
Response Validation
    ↓
Display Results
```

## Configuration Management

### Storage Locations

**XDG Compliant** (Linux/macOS):
- Primary: `~/.config/apimgr/config.json`
- Active env: `~/.config/apimgr/active.env`
- Legacy: `~/.apimgr.json` (auto-migrated)

**Windows**:
- Primary: `%APPDATA%\apimgr\config.json`
- Active env: `%APPDATA%\apimgr\active.env`

### File Locking

Platform-specific implementations prevent concurrent modifications:

**Unix** (`lock_unix.go`):
```go
func (m *Manager) acquireLock() error {
    // Uses flock() system call
    syscall.Flock(fd, syscall.LOCK_EX)
}
```

**Windows** (`lock_windows.go`):
```go
func (m *Manager) acquireLock() error {
    // Uses Windows file locking APIs
    LockFileEx(...)
}
```

### Active Environment

Generated `active.env` file exports:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export ANTHROPIC_BASE_URL="https://api.anthropic.com"
export ANTHROPIC_MODEL="claude-3-opus-20240229"
export APIMGR_ACTIVE="my-config"
```

Shell integration sources this file automatically.

## Provider System

### Auto-Detection

Provider is auto-detected based on base URL:
```go
func DetectProvider(baseURL string) string {
    if strings.Contains(baseURL, "api.anthropic.com") {
        return "anthropic"
    }
    if strings.Contains(baseURL, "api.openai.com") {
        return "openai"
    }
    return "anthropic" // default
}
```

### Request Formatting

Each provider implements its own request format:

**Anthropic**:
```json
{
  "model": "claude-3-opus-20240229",
  "messages": [{"role": "user", "content": "test"}],
  "max_tokens": 1024
}
```

**OpenAI**:
```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "test"}],
  "max_tokens": 1024
}
```

## TUI Architecture

### Keyboard Navigation

```
j/k or ↑/↓  → Move cursor
g/G         → Jump to top/bottom
Enter       → Select/View details
s           → Switch locally
S           → Switch globally
a           → Add config
e           → Edit config
d           → Delete config
p           → Ping test
t           → Compatibility test
m           → Switch model
?           → Help
q           → Quit
```

### State Management

TUI maintains state through the model:
- Current view (list/detail/help/input)
- Selected configuration
- Cursor position
- Input buffers
- Error messages

### Rendering

Lipgloss provides styled components:
```go
var titleStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("205"))

var listStyle = lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder())
```

## Security Design

### API Key Protection

1. **Masking**: Keys are masked in output
   ```go
   func MaskAPIKey(key string) string {
       if len(key) <= 8 {
           return strings.Repeat("*", len(key))
       }
       return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
   }
   ```

2. **File Permissions**: Config files set to 0600
   ```go
   os.Chmod(configPath, 0600) // Owner read/write only
   ```

3. **No Logging**: API keys never written to logs

### Secure Defaults

- HTTPS required for API endpoints
- TLS certificate verification enabled
- Timeout protections on all HTTP requests
- Input validation and sanitization

## Performance Considerations

### Optimization Strategies

1. **Lazy Loading**: TUI loads data only when needed
2. **Caching**: Configuration cached in memory during operations
3. **Connection Pooling**: HTTP client reuses connections
4. **Efficient JSON**: Using gjson/sjson for fast JSON operations

### Benchmarks

The project includes benchmarks for critical paths:
```go
func BenchmarkConfigLoad(b *testing.B)
func BenchmarkConfigSave(b *testing.B)
func BenchmarkProviderDetection(b *testing.B)
```

## Platform Support

### Build Configuration

Cross-platform builds using CGO_ENABLED=0:
```makefile
build:
	CGO_ENABLED=0 go build -o apimgr .
```

### Platform-Specific Code

- File locking: `lock_unix.go`, `lock_windows.go`
- Path handling: Uses `filepath.Join()` for cross-platform paths
- Environment variables: Platform-aware detection

### Release Targets

GoReleaser builds for:
- darwin/amd64 (macOS Intel)
- darwin/arm64 (macOS Apple Silicon)
- linux/amd64
- linux/arm64
- windows/amd64

## Testing Strategy

### Unit Tests
- Individual function testing
- Mocked dependencies
- Table-driven tests

### Integration Tests
- End-to-end command testing
- File system operations
- Provider interactions

### Property-Based Tests
Using gopter for randomized testing:
```go
properties.Property("Config round-trip", ...)
```

## Error Handling

### Patterns

1. **Wrapped Errors**: Using `fmt.Errorf("context: %w", err)`
2. **User-Friendly Messages**: Clear error messages for users
3. **Detailed Logging**: Technical details for debugging

### Error Recovery

- Automatic backup before destructive operations
- Lock cleanup on error
- Graceful degradation when possible

## Future Architecture Improvements

### Planned Enhancements

1. **Configuration Encryption**: Optional encrypted storage
2. **Plugin System**: Support for custom providers via plugins
3. **Remote Sync**: Synchronize configs across devices
4. **Audit Logging**: Track all configuration changes
5. **API Caching**: Cache API responses for development

### Refactoring Opportunities

See [CODE_AUDIT_REPORT.md](CODE_AUDIT_REPORT.md) for detailed analysis.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to the architecture.

---

Last Updated: 2025-01-25
