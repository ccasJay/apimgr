# Quick Start Guide

[中文版](QUICKSTART.zh.md)

## Installation

### Recommended: Binary Download
Download the appropriate binary for your system from [GitHub Releases](https://github.com/ccasJay/apimgr/releases).

### Alternative: Go Install
```bash
go install github.com/ccasJay/apimgr@latest
```

### Alternative: Compile from Source
```bash
git clone https://github.com/ccasJay/apimgr.git
cd apimgr
go build
sudo mv apimgr /usr/local/bin/  # Optional: install system-wide
```

## Initialization

**Note**: The `init` command has been replaced with `enable`. Run the enable command to set up the application:

```bash
apimgr enable
```

This will guide you through:
1. Configuration directory creation
2. Shell integration setup
3. Environment variable configuration

## Basic Usage

### 1. Add a Configuration

#### Interactive Mode
```bash
apimgr add
```

#### Command Line Mode
```bash
# Add Anthropic configuration
apimgr add my-anthropic --sk sk-ant-api-key --url https://api.anthropic.com --model claude-3-sonnet

# Add OpenAI configuration
apimgr add my-openai --sk sk-oo-api-key --url https://api.openai.com/v1 --model gpt-4 --provider openai
```

### 2. List Configurations
```bash
apimgr list
```

### 3. Switch Configurations
```bash
apimgr switch my-openai
```

### 4. Test Connectivity
```bash
# Test active configuration
apimgr ping

# Test specific configuration
apimgr ping my-anthropic

# Test custom URL
apimgr ping -u https://api.example.com
```

### 5. Check Status
```bash
apimgr status
```

### 6. Edit Configuration
```bash
apimgr edit my-openai
```

### 7. Remove Configuration
```bash
apimgr remove my-anthropic
```

## Shell Integration

### Bash
```bash
echo '[[ -f ~/.config/apimgr/active.env ]] && source ~/.config/apimgr/active.env' >> ~/.bashrc
source ~/.bashrc
```

### Zsh
```bash
echo '[[ -f ~/.config/apimgr/active.env ]] && source ~/.config/apimgr/active.env' >> ~/.zshrc
source ~/.zshrc
```

### Fish
```bash
echo 'test -f ~/.config/apimgr/active.env; and source ~/.config/apimgr/active.env' >> ~/.config/fish/config.fish
source ~/.config/fish/config.fish
```

## Advanced Features

### TUI Mode
Launch the interactive Terminal UI for a visual interface:
```bash
apimgr  # Run without arguments
```

### Compatibility Testing
```bash
# Test API compatibility with real requests
apimgr ping -T

# Test streaming API support
apimgr ping -T --stream

# Verbose output
apimgr ping -T -v
```

### JSON Output
```bash
apimgr list --json
apimgr ping --json
```

### Local vs Global Configuration
```bash
# Global switch (affects all shells)
apimgr switch my-config

# Local switch (current shell only)
apimgr switch -l my-config
```

## Help
```bash
apimgr --help
apimgr <command> --help
```
