# API Manager (apimgr)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Cross-Platform](https://img.shields.io/badge/Platform-MacOS%20%7C%20Linux%20%7C%20Windows-blue)](https://github.com/your-username/apimgr)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org/)

[中文版](README.zh.md)

A modern, feature-rich command-line tool for managing API configurations and testing connectivity. apimgr simplifies working with multiple API providers by centralizing configuration management with secure storage and seamless shell integration.

## Features

### Core Features
- **Multi-Provider Support**: Manage configurations for Anthropic, OpenAI, Doubao, and custom API providers
- **Connectivity Testing**: Validate API endpoints with customizable requests and detailed error diagnostics
- **Easy Configuration Switching**: Seamlessly switch between different API configurations globally or locally
- **Shell Integration**: Automatically export configurations to environment variables for tools like Claude Code
- **JSON Output**: Machine-readable results for scripting and automation workflows
- **Secure Storage**: Encrypted storage for API keys (optional) with configurable security settings
- **Cross-Platform**: Native support for macOS, Linux, and Windows

### Advanced Features
- **Interactive Editing**: Intuitive interactive commands for adding and modifying configurations
- **Dual Configuration Modes**:
  - Global: Persistent configuration across all shells
  - Local: Temporary configuration for current shell session only (`-l/--local` flag)
- **Comprehensive Status Checking**: View both global and shell environment configurations in one command
- **XDG Compliance**: Follows XDG Base Directory Specification on Linux systems
- **Auto-Synchronization**: Sync configurations with supported tools (Claude Code, etc.)
- **Rich Diagnostics**: Detailed error messages for timeout, connection refused, DNS failures, and more

## Installation

### Prerequisites
- Go 1.21 or higher (for source compilation)

### Recommended Installation
#### Go Install
```bash
go install https://github.com/ccasJay/apimgr.git
```

#### From Source
```bash
git clone https://github.com/ccasJay/apimgr.git
cd apimgr
go build
sudo mv apimgr /usr/local/bin/  # Optional: install system-wide
```

#### Using Makefile
```bash
git clone https://github.com/ccasJay/apimgr.git

cd apimgr
make install  # Builds and installs locally
# sudo make install  # For system-wide installation
```

## Quick Start

1. **Add a new configuration**
   ```bash
   apimgr add my-config --sk sk-ant-api03-... --url https://api.anthropic.com
   ```
   or just use
   ```bash
   apimgr add
   ```

2. **List all configurations**
   ```bash
   apimgr list
   ```

3. **Switch to a configuration**
   ```bash
   apimgr switch my-config  # Global switch
   apimgr switch -l my-config  # Local (current shell only)
   ```

4. **Test connectivity**
   ```bash
   apimgr ping  # Test active configuration
   apimgr ping -u https://api.example.com  # Test custom URL
   apimgr ping -T -p /chat/completions  # Test real API endpoint with POST request
   ```

5. **Check current status**
   ```bash
   apimgr status
   ```

For detailed usage, see the [Quick Start Guide](QUICKSTART.md).

## Configuration

### Configuration Paths
- **Default**: `~/.config/apimgr/config.json` (XDG compliant on Linux)
- **Legacy**: `~/.apimgr.json` (automatically migrated to new path)
- **Custom**: Use `XDG_CONFIG_HOME` to specify a custom directory:
  ```bash
  XDG_CONFIG_HOME=~/.myconfig apimgr add my-config --sk sk-xxx...
  ```

### Configuration Format
```json
{
  "configs": [
    {
      "alias": "my-config",
      "api_key": "sk-ant-api03-...",
      "auth_token": "",
      "base_url": "https://api.anthropic.com",
      "model": "claude-3-opus-20240229",
      "provider": "anthropic"
    }
  ],
  "active": "my-config"
}
```

## Commands

### Basic Commands
```bash
apimgr add        # Add a new API configuration (interactive or non-interactive)
apimgr list       # List all saved configurations with active indicator
apimgr switch     # Switch to a configuration (global or local)
apimgr ping       # Test API connectivity with detailed diagnostics
apimgr status     # Show combined global and shell configuration status
apimgr edit       # Edit an existing configuration (interactive or non-interactive)
apimgr remove     # Remove a configuration
```

### Command Details

#### `apimgr ping`
Test API connectivity with customizable options:
```bash
apimgr ping [alias]          # Test specific or active configuration
apimgr ping -u URL           # Test custom URL
apimgr ping -X GET           # Use specific HTTP method
apimgr ping -t 30s           # Custom timeout
apimgr ping -j               # JSON output
apimgr ping -T -p /chat/completions  # Test real API with POST request
```

#### `apimgr status`
Shows configuration source priority (shell environment overrides global):
```
Current configuration status:
=========================================
1. Global active configuration (config file):
   Alias: my-config
   API Key: sk-ant-api03-**************
   Base URL: https://api.anthropic.com
   Model: claude-3-opus-20240229

2. Current Shell environment:
   No environment variables set

=========================================
💡 Currently using global configuration (Shell has no environment variables set)
```

#### `apimgr list`
Lists configurations with active marker:
```
Available configurations:
* my-config: API Key: sk-ant-api03-************** (URL: https://api.anthropic.com, Model: claude-3-opus-20240229)
  openai-dev: API Key: sk-************** (URL: https://api.openai.com, Model: gpt-4o)
```

## Environment Variables

apimgr automatically respects and displays these environment variables:
- `ANTHROPIC_API_KEY`
- `ANTHROPIC_AUTH_TOKEN`
- `ANTHROPIC_BASE_URL`
- `ANTHROPIC_MODEL`
- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`
- `OPENAI_MODEL`
- `APIMGR_ACTIVE`

## Usage Examples

### Interactive Configuration
```bash
$ apimgr add
Enter config alias: my-anthropic
Enter API key: sk-ant-api03-...
Enter Authentication Token (press Enter to skip):
Enter Base URL (default: https://api.anthropic.com):
Enter Model name (press Enter to skip): claude-3-opus-20240229
✅ Configuration 'my-anthropic' added successfully
```

### Non-Interactive Configuration
```bash
apimgr add openai-prod \
  --sk sk-... \
  --url https://api.openai.com \
  --model gpt-4o
```

### Edit Configuration
```bash
# Interactive edit
apimgr edit my-config

# Non-interactive edit
apimgr edit my-config --url https://api.new-domain.com --model claude-3-sonnet-20240229
```

### Local Configuration
```bash
apimgr switch -l temporary-config  # Use configuration only for current shell
apimgr status  # Shows both global and local configuration
```

## Shell Integration

Run `apimgr install` to enable shell integration for automatic configuration loading. Supported shells:
- Bash
- Zsh
- Fish

## Troubleshooting

### Common Errors
- **Timeout Error**: Increase timeout with `-t` flag (e.g., `apimgr ping -t 30s`)
- **Connection Refused**: Check if API server is running and accessible
- **DNS Resolution Failed**: Verify domain name and network connectivity
- **Invalid URL**: Ensure URL includes protocol (http:// or https://)

### Detailed Diagnostics
Use `apimgr ping -j` for JSON output with full error details:

```json
{
  "url": "https://api.example.com",
  "statusCode": 0,
  "statusText": "",
  "requestMethod": "HEAD",
  "durationMs": 10001,
  "timeoutMs": 10000,
  "success": false
}
```

## Documentation

- [Quick Start Guide](QUICKSTART.md)
- [Command Reference](COMMANDS.md) (TODO)
- [Contribution Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md) (TODO)

## Contributing

We welcome contributions! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and development process.

## License

MIT License - see [LICENSE](LICENSE) for details

## Support

For issues, feature requests, or questions, please open an [issue](https://github.com/your-username/apimgr/issues) on GitHub.
