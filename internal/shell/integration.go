package shell

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const shellIntegrationTemplate = `#!/bin/bash
# apimgr shell integration script
# Generated automatically by apimgr enable

# Configuration
APIMGR_CONFIG_DIR="${HOME}/.config/apimgr"
APIMGR_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/tmp}/apimgr-$(id -u)"
APIMGR_SOCKET="${APIMGR_RUNTIME_DIR}/apimgr.sock"
APIMGR_PID_FILE="${APIMGR_RUNTIME_DIR}/daemon.pid"

# Cache variables
_APIMGR_CACHE=""
_APIMGR_VERSION=""
_APIMGR_CACHE_TIME=0
_APIMGR_CMD_COUNT=0

# Auto-start daemon if not running
_apimgr_ensure_daemon() {
    if [[ ! -S "${APIMGR_SOCKET}" ]]; then
        # Check if daemon process exists
        if [[ -f "${APIMGR_PID_FILE}" ]]; then
            local pid=$(cat "${APIMGR_PID_FILE}" 2>/dev/null)
            if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
                # Stale PID file, remove it
                rm -f "${APIMGR_PID_FILE}"
            fi
        fi
        
        # Start daemon in background
        if command -v apimgr >/dev/null 2>&1; then
            apimgr daemon start >/dev/null 2>&1 &
            
            # Wait for socket to be available (max 2 seconds)
            local wait_count=0
            while [[ ! -S "${APIMGR_SOCKET}" ]] && [[ $wait_count -lt 20 ]]; do
                sleep 0.1
                ((wait_count++))
            done
        fi
    fi
}

# Query daemon via socket
_apimgr_query() {
    local cmd="$1"
    local result=""
    
    # Try nc first
    if command -v nc >/dev/null 2>&1; then
        result=$(echo "$cmd" | nc -U "${APIMGR_SOCKET}" 2>/dev/null)
    # Try socat
    elif command -v socat >/dev/null 2>&1; then
        result=$(echo "$cmd" | socat - UNIX-CONNECT:"${APIMGR_SOCKET}" 2>/dev/null)
    # Fallback to direct file read
    elif [[ -r "${APIMGR_CONFIG_DIR}/config.json" ]]; then
        # Pure shell JSON parsing (basic)
        local config=$(cat "${APIMGR_CONFIG_DIR}/config.json" 2>/dev/null)
        local active=$(echo "$config" | grep -o '"active"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4)
        
        if [[ -n "$active" ]]; then
            local api_key=$(echo "$config" | grep -A10 "\"alias\"[[:space:]]*:[[:space:]]*\"$active\"" | grep -o '"api_key"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | cut -d'"' -f4)
            local base_url=$(echo "$config" | grep -A10 "\"alias\"[[:space:]]*:[[:space:]]*\"$active\"" | grep -o '"base_url"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | cut -d'"' -f4)
            
            if [[ "$cmd" == "GET" ]]; then
                result="{\"ANTHROPIC_API_KEY\":\"$api_key\""
                [[ -n "$base_url" ]] && result="${result},\"ANTHROPIC_BASE_URL\":\"$base_url\""
                result="${result}}"
            elif [[ "$cmd" == "VERSION" ]]; then
                result=$(stat -f %m "${APIMGR_CONFIG_DIR}/config.json" 2>/dev/null || stat -c %Y "${APIMGR_CONFIG_DIR}/config.json" 2>/dev/null || echo "0")
            fi
        fi
    fi
    
    echo "$result"
}

# Load environment variables from daemon
_apimgr_load_env() {
    local force="${1:-false}"
    local current_time=$(date +%s)
    
    # Cache invalidation conditions:
    # 1. Force refresh requested
    # 2. Cache is older than 10 seconds
    # 3. Every 10 commands
    # 4. Version changed
    if [[ "$force" == "true" ]] || \
       [[ $((current_time - _APIMGR_CACHE_TIME)) -gt 10 ]] || \
       [[ $((_APIMGR_CMD_COUNT % 10)) -eq 0 ]]; then
        
        # Check version first
        local new_version=$(_apimgr_query "VERSION")
        if [[ "$new_version" != "$_APIMGR_VERSION" ]] || [[ "$force" == "true" ]]; then
            _APIMGR_CACHE=$(_apimgr_query "GET")
            _APIMGR_VERSION="$new_version"
            _APIMGR_CACHE_TIME=$current_time
        fi
    fi
    
    ((_APIMGR_CMD_COUNT++))
    
    # Parse and export environment variables
    if [[ -n "$_APIMGR_CACHE" ]] && [[ "$_APIMGR_CACHE" != "{}" ]]; then
        # Parse JSON and export variables
        while IFS= read -r line; do
            if [[ "$line" =~ \"([^\"]+)\":\"([^\"]+)\" ]]; then
                export "${BASH_REMATCH[1]}=${BASH_REMATCH[2]}"
            fi
        done <<< "$_APIMGR_CACHE"
    fi
}

# Hook for bash/zsh
if [[ -n "$BASH_VERSION" ]]; then
    # Bash: use PROMPT_COMMAND
    _apimgr_prompt_command() {
        _apimgr_load_env
        # Preserve existing PROMPT_COMMAND
        if [[ -n "$_APIMGR_ORIG_PROMPT_COMMAND" ]]; then
            eval "$_APIMGR_ORIG_PROMPT_COMMAND"
        fi
    }
    
    if [[ "$PROMPT_COMMAND" != *"_apimgr_prompt_command"* ]]; then
        _APIMGR_ORIG_PROMPT_COMMAND="$PROMPT_COMMAND"
        PROMPT_COMMAND="_apimgr_prompt_command"
    fi
elif [[ -n "$ZSH_VERSION" ]]; then
    # Zsh: use precmd
    precmd_functions+=(_apimgr_load_env)
fi

# Initial load
_apimgr_ensure_daemon
_apimgr_load_env true

# Debug function
apimgr_debug() {
    echo "APIMGR Debug Information:"
    echo "  Config Dir: ${APIMGR_CONFIG_DIR}"
    echo "  Runtime Dir: ${APIMGR_RUNTIME_DIR}"
    echo "  Socket: ${APIMGR_SOCKET}"
    echo "  Socket exists: $([ -S "${APIMGR_SOCKET}" ] && echo "yes" || echo "no")"
    echo "  Daemon PID: $(cat "${APIMGR_PID_FILE}" 2>/dev/null || echo "not found")"
    echo "  Cache Version: ${_APIMGR_VERSION}"
    echo "  Cache Time: ${_APIMGR_CACHE_TIME}"
    echo "  Command Count: ${_APIMGR_CMD_COUNT}"
    echo "  Environment Variables:"
    env | grep ANTHROPIC || echo "    (none set)"
}
`

type Generator struct {
	ConfigDir  string
	RuntimeDir string
}

func NewGenerator() *Generator {
	homeDir, _ := os.UserHomeDir()
	uid := os.Getuid()
	
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	runtimeDir = filepath.Join(runtimeDir, fmt.Sprintf("apimgr-%d", uid))
	
	return &Generator{
		ConfigDir:  filepath.Join(homeDir, ".config", "apimgr"),
		RuntimeDir: runtimeDir,
	}
}

func (g *Generator) Generate() (string, error) {
	tmpl, err := template.New("shell").Parse(shellIntegrationTemplate)
	if err != nil {
		return "", err
	}
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, g); err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

func (g *Generator) WriteToFile(path string) error {
	content, err := g.Generate()
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Write file with executable permissions
	return os.WriteFile(path, []byte(content), 0755)
}
