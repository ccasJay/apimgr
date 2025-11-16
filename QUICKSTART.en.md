# Quick Start Guide

[中文版](QUICKSTART.md)

## Installation

### Method 1: Binary Download
Download the appropriate binary for your system from [GitHub Releases](https://github.com/your-username/apimgr/releases).

### Method 2: Compile from Source
```bash
git clone https://github.com/your-username/apimgr.git
cd apimgr
go build
mv apimgr /usr/local/bin/
```

### Method 3: Go Install
```bash
go install github.com/your-username/apimgr@latest
```

## Initialization

Run the init command to set up the application:

```bash
apimgr init
```

This will guide you through:
1. Configuration directory creation
2. Default API provider selection
3. Shell integration setup

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
echo 'source ~/.config/apimgr/active.env' >> ~/.bashrc
```

### Zsh
```bash
echo 'source ~/.config/apimgr/active.env' >> ~/.zshrc
```

### Fish
```bash
echo 'source ~/.config/apimgr/active.env' >> ~/.config/fish/config.fish
```

Reload the configuration:
```bash
source ~/.bashrc
```

## Advanced Features

### Batch Operations
```bash
# Test all configurations in parallel
apimgr ping --all
```

### JSON Output
```bash
apimgr list --json
apimgr ping --json
```

## Help
```bash
apimgr --help
apimgr <command> --help
```
