# API Manager (apimgr)

[中文版](README.md)

A command-line tool for managing API configurations (keys, base URLs, models) and testing connectivity.

## Features

### Core Features

- **Multi-provider support**: Anthropic, OpenAI, and custom providers
- **Connectivity testing**: Validate API endpoints with configurable settings
- **Easy configuration switching**: Switch between different API configurations
- **Shell integration**: Automatically apply configurations to environment variables
- **JSON output**: Machine-readable results for scripting
- **Secure storage**: Optional encryption for API keys
- **Cross-platform**: Works on macOS, Linux, and Windows

### Advanced Features

- **Persistent configuration**: Configurations are saved automatically
- **Interactive editing**: Interactive add and edit commands
- **Local switching**: Use `-l/--local` to switch only in current shell
- **Status checking**: View global and current shell configuration status
- **XDG support**: Follows XDG Base Directory Specification on Linux
- **Auto-synchronization**: Sync configurations to tools like Claude Code

## Installation

### Binary Download

Download the latest release from [GitHub Releases](https://github.com/your-username/apimgr/releases).

### Source Compilation

```bash
git clone https://github.com/your-username/apimgr.git
cd apimgr
go build
sudo mv apimgr /usr/local/bin/
```

### Go Install

```bash
go install github.com/your-username/apimgr@latest
```

## Quick Start

See the full [Quick Start Guide](QUICKSTART.en.md)

## Configuration File

### Configuration Paths

- **Default**: `~/.config/apimgr/config.json` (follows XDG specification on Linux)
- **Legacy**: `~/.apimgr.json` (automatically migrated to new path)
- **Custom**: Use `XDG_CONFIG_HOME` to specify custom configuration directory:

  ```bash
  XDG_CONFIG_HOME=~/.myconfig apimgr add my-config --sk sk-xxx...
  ```

## Commands

```bash
apimgr add        # Add a new API configuration
apimgr list       # List all configurations
apimgr switch     # Switch to a configuration
apimgr ping       # Test connectivity
apimgr status     # Show active configuration
apimgr remove     # Remove a configuration
```

## Documentation

- [Quick Start Guide](QUICKSTART.en.md)
- [Contribution Guide](CONTRIBUTING.en.md)
- [Code of Conduct](CODE_OF_CONDUCT.en.md)

## Contributing

Please read [CONTRIBUTING.en.md](CONTRIBUTING.en.md) for details on our code of conduct and development process.

## License

MIT
