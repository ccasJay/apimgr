# API Manager (apimgr)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Cross-Platform](https://img.shields.io/badge/Platform-MacOS%20%7C%20Linux%20%7C%20Windows-blue)](https://github.com/your-username/apimgr)
[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue)](https://golang.org/)

[中文版](README.zh.md)

A modern, feature-rich command-line tool for managing API configurations and testing connectivity. apimgr simplifies working with multiple API providers by centralizing configuration management with secure storage and seamless shell integration.(This project is only compatible with Claude Code for the time being, more tools like codex or gemini will be available in the future. )

## Table of Contents

- [Features](#features)
  - [Core Features](#core-features)
  - [Advanced Features](#advanced-features)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Supported Operating Systems](#supported-operating-systems)
  - [Installation Methods](#installation-methods)
- [Quick Start](#quick-start)
  - [TUI Mode (Recommended)](#tui-mode-recommended)
  - [CLI Mode](#cli-mode)
- [Configuration](#configuration)
  - [Configuration Paths](#configuration-paths)
  - [Configuration Format](#configuration-format)
  - [Provider Auto-Detection](#provider-auto-detection)
- [Commands](#commands)
  - [TUI Mode](#tui-mode)
  - [Basic Commands](#basic-commands)
  - [Command Details](#command-details)
- [Environment Variables](#environment-variables)
- [Usage Examples](#usage-examples)
- [Shell Integration](#shell-integration)
- [FAQ](#faq)
- [Troubleshooting](#troubleshooting)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)
- [Support](#support)











## !!!WARNING : this may override your ~/.cladue/setting.json file,please remember to BACKUP.


## Features

### Core Features

```
┌─────────────┐        ┌──────────────────┐        ┌─────────────────┐
│   User      │───────▶│  apimgr CLI/TUI  │───────▶│  Config Storage │
│  Commands   │        │                  │        │   (JSON file)   │
└─────────────┘        └──────────────────┘        └─────────────────┘
                              │      │
                              │      │
                              ▼      ▼
                       ┌──────────────────┐
                       │  Environment     │
                       │  Variables       │
                       │  (active.env)    │
                       └──────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │  Claude Code &   │
                       │  Other Tools     │
                       └──────────────────┘
```

- **Multi-Provider Support**: Manage configurations for Anthropic, OpenAI, Doubao, and custom API providers
- **Connectivity Testing**: Validate API endpoints with customizable requests and detailed error diagnostics
- **Easy Configuration Switching**: Seamlessly switch between different API configurations globally or locally
- **Shell Integration**: Automatically export configurations to environment variables for tools like Claude Code
- **JSON Output**: Machine-readable results for scripting and automation workflows
- **Secure Storage**: Encrypted storage for API keys (optional) with configurable security settings
- **Cross-Platform**: Native support for macOS, Linux, and Windows

### Advanced Features

```
┌──────────────────────────────────────────────────────────────┐
│                      Configuration Modes                      │
├──────────────────────────┬───────────────────────────────────┤
│   Global Mode            │      Local Mode (-l flag)         │
│   ┌───────────────┐      │      ┌───────────────┐           │
│   │ config.json   │──────┼─────▶│  Shell ENV    │           │
│   │ (persistent)  │      │      │  (temporary)  │           │
│   └───────────────┘      │      └───────────────┘           │
│          │               │             │                     │
│          ▼               │             ▼                     │
│   ┌───────────────┐      │      ┌───────────────┐           │
│   │  active.env   │      │      │ Current Shell │           │
│   │  (all shells) │      │      │    Only       │           │
│   └───────────────┘      │      └───────────────┘           │
└──────────────────────────┴───────────────────────────────────┘
```

- **Interactive TUI**: Full-featured terminal user interface with keyboard navigation (just run `apimgr` without arguments)
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
- **Go 1.21+**: Required for building from source
- **Operating System**: One of the following:
  - macOS (x86_64/ARM64)
  - Linux (x86_64/ARM64)
  - Windows (x86_64)

### Supported Operating Systems

apimgr is tested and supported on:
- **macOS**: 10.15 (Catalina) or later
- **Linux**: Ubuntu 20.04+, Debian 10+, Fedora 33+, Arch Linux, and other modern distributions
- **Windows**: Windows 10 or later (use PowerShell or WSL2 for best experience)

### Installation Methods
#### Go Install
```bash
go install github.com/ccasJay/apimgr@latest
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

### TUI Mode (Recommended)

Simply run `apimgr` without arguments to launch the interactive TUI:

```bash
apimgr
```

TUI Keyboard Shortcuts:
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Move up/down |
| `g/G` | Jump to top/bottom |
| `Enter` | View details |
| `s` | Switch config locally (Claude Code) |
| `S` | Switch config globally |
| `a` | Add config |
| `e` | Edit config |
| `d` | Delete config |
| `p` | Ping test |
| `t` | Compatibility test |
| `m` | Switch model |
| `?` | Help |
| `q` | Quit |

### CLI Mode

1. **Add a new configuration**
   ```bash
   apimgr add my-config --sk sk-ant-api03-... --url https://api.anthropic.com
   ```
   
   **Output:**
   ```
   ✅ Configuration 'my-config' added successfully
   ✅ Configuration updated - active.env regenerated
   ```
   
   or just use
   ```bash
   apimgr add
   ```
   
   **Interactive Output:**
   ```
   Enter config alias: my-config
   Enter API key: sk-ant-api03-...
   Enter Authentication Token (press Enter to skip):
   Enter Base URL (default: https://api.anthropic.com):
   Enter Model name (press Enter to skip): claude-3-opus-20240229
   ✅ Configuration 'my-config' added successfully
   ```

2. **List all configurations**
   ```bash
   apimgr list
   ```
   
   **Output:**
   ```
   Available configurations:
   * my-config: API Key: sk-ant-api03-************** (URL: https://api.anthropic.com, Model: claude-3-opus-20240229)
     openai-dev: API Key: sk-************** (URL: https://api.openai.com, Model: gpt-4o)
     backup-config: API Key: sk-************** (URL: https://custom-api.example.com, Model: claude-3-sonnet-20240229)

   (* indicates currently active configuration)
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
   
   **Success Output:**
   ```
   Testing connection to: https://api.anthropic.com
   ✅ Connection successful!
   Status: 200 OK
   Response Time: 245ms
   ```
   
   **Failure Output (Connection Timeout):**
   ```
   Testing connection to: https://slow-api.example.com
   ❌ Connection failed!
   Error: Request timeout after 10000ms
   
   💡 Tip: Try increasing timeout with -t flag (e.g., apimgr ping -t 30s)
   ```
   
   **Failure Output (Server Unreachable):**
   ```
   Testing connection to: https://api.down-server.com
   ❌ Connection failed!
   Error: Connection refused - server is not responding
   
   💡 Tip: Check if the server is running and accessible
   ```
   
   **Failure Output (DNS Resolution Failed):**
   ```
   Testing connection to: https://invalid-domain.example
   ❌ Connection failed!
   Error: DNS resolution failed - no such host
   
   💡 Tip: Check your network connection and verify the domain name spelling
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

### Provider Auto-Detection
When the `provider` field is not explicitly set, apimgr will automatically detect the provider based on the base URL:

| URL Pattern | Detected Provider |
|-------------|-------------------|
| `*api.anthropic.com*` | anthropic |
| `*api.openai.com*` | openai |
| Other URLs | anthropic (default) |

This means you can omit the `provider` field when adding configurations with standard API URLs:
```bash
# Provider will be auto-detected as "anthropic"
apimgr add my-anthropic --sk sk-ant-... --url https://api.anthropic.com

# Provider will be auto-detected as "openai"
apimgr add my-openai --sk sk-... --url https://api.openai.com
```
```

## Commands

### TUI Mode
```bash
apimgr            # Launch interactive TUI interface
```

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
apimgr ping -T               # Test real API compatibility (auto-detects provider from URL)
apimgr ping -T -p /chat/completions  # Test real API with custom endpoint path
apimgr ping -T --stream      # Test streaming API compatibility
apimgr ping -T -v            # Verbose output with request/response details
```

The `-T` flag enables compatibility testing mode, which:
- Sends a real chat completion request to validate API format
- Auto-detects the provider (Anthropic/OpenAI) from the base URL
- Validates response structure matches Claude Code expectations
- Supports streaming mode testing with `--stream` flag

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

## FAQ

### Q: How do I resolve "Connection Timeout" errors?

**A:** If you're experiencing connection timeout errors, the server might be slow to respond or under heavy load. Try increasing the timeout parameter:

```bash
apimgr ping -t 30s  # Set timeout to 30 seconds
```

You can adjust the timeout value based on your network conditions:
- `-t 15s` for moderately slow connections
- `-t 30s` for very slow connections or servers under load
- `-t 60s` for extremely slow or distant servers

### Q: What should I do when I see "Cannot connect to server" errors?

**A:** This error occurs when the API server is not responding. Follow these troubleshooting steps:

1. **Verify the server is running**: Check if the API service is operational
   ```bash
   curl -I https://api.anthropic.com  # Test if server responds
   ```

2. **Check your network connection**: Ensure you have internet connectivity
   ```bash
   ping google.com  # Test general internet connectivity
   ```

3. **Verify the URL is correct**: Make sure you're using the correct base URL
   ```bash
   apimgr status  # Review your current configuration
   ```

4. **Check firewall settings**: Ensure your firewall isn't blocking outbound HTTPS connections

5. **Test with a different network**: Try connecting from a different network to rule out local network issues

### Q: How do I fix "Domain name resolution failure" errors?

**A:** DNS resolution errors indicate that your system cannot translate the domain name to an IP address. Try these solutions:

1. **Verify the domain name spelling**: Double-check for typos in the URL
   ```bash
   apimgr list  # Review your saved configurations
   ```

2. **Check your DNS settings**: Ensure your system's DNS is configured correctly
   ```bash
   # Test DNS resolution
   nslookup api.anthropic.com
   # Or use dig
   dig api.anthropic.com
   ```

3. **Test your internet connection**: Make sure you have network access
   ```bash
   ping 8.8.8.8  # Test connection to Google's DNS
   ```

4. **Try using a different DNS server**: Temporarily use public DNS like Google (8.8.8.8) or Cloudflare (1.1.1.1)

5. **Check if the domain exists**: Verify the domain is valid and active
   ```bash
   whois api.anthropic.com
   ```

### Q: Can I use apimgr with multiple API providers simultaneously?

**A:** Yes! apimgr is designed for multi-provider management. You can:
- Add configurations for different providers (Anthropic, OpenAI, custom APIs)
- Switch between them globally or locally (current shell only)
- Use different configurations in different terminal sessions

Example:
```bash
apimgr add anthropic-prod --sk sk-ant-... --url https://api.anthropic.com
apimgr add openai-dev --sk sk-... --url https://api.openai.com
apimgr switch anthropic-prod  # Global switch
apimgr switch -l openai-dev   # Local switch (current shell only)
```

### Q: How do I keep my API keys secure?

**A:** apimgr implements several security measures:
- Configuration files are stored with 0600 permissions (readable only by owner)
- API keys are masked in list/status output (e.g., `sk-ant-api03-**************`)
- Keys are stored locally on your machine, never sent to external services
- Environment variables are only set in your current shell session

Best practices:
1. Never commit configuration files to version control
2. Use different API keys for development and production
3. Regularly rotate your API keys
4. Use the `-l/--local` flag for temporary testing to avoid changing global config

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
