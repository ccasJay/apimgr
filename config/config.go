package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"apimgr/internal/providers"
	"apimgr/internal/utils"
)

// APIConfig represents a single API configuration
type APIConfig struct {
	Alias     string `json:"alias"`
	Provider  string `json:"provider"` // API provider type
	APIKey    string `json:"api_key"`
	AuthToken string `json:"auth_token"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
}

// File represents the structure of the config file
type File struct {
	Active  string      `json:"active"`
	Configs []APIConfig `json:"configs"`
}

// Manager manages API configurations
type Manager struct {
	configPath string
	mu         sync.Mutex // Mutex to protect concurrent access
}

// NewConfigManager creates a new Manager with unified config path
func NewConfigManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Check XDG_CONFIG_HOME environment variable for custom config location
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		// Use default XDG path (~/.config)
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	// Always use XDG config location (new standard)
	xdgConfigPath := filepath.Join(xdgConfigHome, "apimgr", "config.json")
	oldConfigPath := filepath.Join(homeDir, ".apimgr.json")

	configPath := xdgConfigPath

	// Ensure XDG directory exists
	configDir := filepath.Dir(xdgConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Migrate from old config if it exists and new config doesn't
	if shouldMigrateConfig(oldConfigPath, xdgConfigPath) {
		if err := migrateConfig(oldConfigPath, xdgConfigPath); err != nil {
			fmt.Printf("⚠️  Failed to migrate config: %v\n", err)
			// Continue with new config path anyway
		} else {
			fmt.Println("✅ Migrated config from old location successfully")
		}
	}

	return &Manager{
		configPath: configPath,
	}, nil
}

// shouldMigrateConfig checks if config migration should be performed
func shouldMigrateConfig(oldPath, newPath string) bool {
	// Migrate if old config exists and new config doesn't
	oldExists := fileExists(oldPath)
	newExists := fileExists(newPath)
	return oldExists && !newExists
}

// migrateConfig migrates configuration from old path to new path
func migrateConfig(oldPath, newPath string) error {
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old config file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("old config file is empty")
	}

	// Validate that it's a valid config
	var tempConfig File
	if err := json.Unmarshal(data, &tempConfig); err != nil {
		// Try old format (array of configs)
		var tempConfigs []APIConfig
		if err2 := json.Unmarshal(data, &tempConfigs); err2 != nil {
			return fmt.Errorf("old config file format is invalid: %w", err)
		}
	}

	// Write to new location with locking
	file, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open new config file: %w", err)
	}

	// Lock the new config file exclusively
	if err := lockFileExclusive(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to lock new config file: %w", err)
	}

	// Write data while holding the lock
	_, err = file.Write(data)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to write new config file: %w", err)
	}

	// Ensure data is flushed
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("failed to sync new config file to disk: %w", err)
	}

	// Unlock and close
	if err := unlockFile(file); err != nil {
		file.Close()
		return fmt.Errorf("failed to unlock new config file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close new config file: %w", err)
	}

	// Backup old config
	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		// Don't fail migration if backup fails
		fmt.Printf("⚠️  Failed to create backup of old config: %v\n", err)
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetConfigPath returns the path to the config file
func (cm *Manager) GetConfigPath() string {
	return cm.configPath
}

// loadConfigFile loads the config file with locking
func (cm *Manager) loadConfigFile() (*File, error) {
	// Open the file with read lock
	file, err := os.OpenFile(cm.configPath, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Configs: []APIConfig{}}, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Lock the file for shared read access (LOCK_SH)
	if err := cm.lockFileShared(file); err != nil {
		return nil, fmt.Errorf("failed to lock config file: %w", err)
	}
	defer func() {
		if err := cm.unlockFile(file); err != nil {
			fmt.Printf("⚠️  Failed to unlock file: %v\n", err)
		}
	}()

	// Read from the locked file descriptor instead of using os.ReadFile
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) == 0 {
		return &File{Configs: []APIConfig{}}, nil
	}

	var configFile File
	err = json.Unmarshal(data, &configFile)
	if err != nil {
		// Try to parse as old format (array of configs)
		var configs []APIConfig
		if err2 := json.Unmarshal(data, &configs); err2 == nil {
			return &File{Configs: configs}, nil
		}
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &configFile, nil
}

// saveConfigFile saves the config file with locking
func (cm *Manager) saveConfigFile(configFile *File) error {
	data, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Open the file with write access (create if not exists)
	file, err := os.OpenFile(cm.configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Lock the file for exclusive write access
	if err := cm.lockFile(file); err != nil {
		return fmt.Errorf("failed to lock config file: %w", err)
	}
	defer func() {
		if err := cm.unlockFile(file); err != nil {
			fmt.Printf("⚠️  Failed to unlock file: %v\n", err)
		}
	}()

	// Write the file while holding the lock
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Ensure data is flushed to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync config file: %w", err)
	}

	return nil
}

// lockFile locks the config file with exclusive lock (for write operations)
func (cm *Manager) lockFile(file *os.File) error {
	return lockFileExclusive(file)
}

// lockFileShared locks the config file with shared lock (for read operations)
func (cm *Manager) lockFileShared(file *os.File) error {
	return lockFileShared(file)
}

// unlockFile unlocks the config file
func (cm *Manager) unlockFile(file *os.File) error {
	return unlockFile(file)
}

// Load loads all configurations from the config file
func (cm *Manager) Load() ([]APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}
	return configFile.Configs, nil
}

// Save saves configurations to the config file
func (cm *Manager) Save(configs []APIConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}
	configFile.Configs = configs
	return cm.saveConfigFile(configFile)
}

// Add adds a new configuration
func (cm *Manager) Add(config APIConfig) error {
	// Set default provider
	if config.Provider == "" {
		config.Provider = "anthropic"
	}

	if err := cm.validateConfig(config); err != nil {
		return err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	configs, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	// Check if alias already exists
	for i, existingConfig := range configs.Configs {
		if existingConfig.Alias == config.Alias {
			configs.Configs[i] = config
			return cm.saveConfigFile(configs)
		}
	}

	configs.Configs = append(configs.Configs, config)
	return cm.saveConfigFile(configs)
}

// Remove removes a configuration by alias
func (cm *Manager) Remove(alias string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configs, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	for i, config := range configs.Configs {
		if config.Alias == alias {
			configs.Configs = append(configs.Configs[:i], configs.Configs[i+1:]...)
			// If removing the active config, clear the active config
			if configs.Active == alias {
				configs.Active = ""
			}
			return cm.saveConfigFile(configs)
		}
	}

	return fmt.Errorf("configuration '%s' does not exist", alias)
}

// Get returns a configuration by alias
func (cm *Manager) Get(alias string) (*APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configs, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}

	for _, config := range configs.Configs {
		if config.Alias == alias {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("configuration '%s' does not exist", alias)
}

// List returns all configurations
func (cm *Manager) List() ([]APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configs, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}
	return configs.Configs, nil
}

// SetActive sets the active configuration
func (cm *Manager) SetActive(alias string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	// Verify the alias exists
	found := false
	for _, config := range configFile.Configs {
		if config.Alias == alias {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("configuration '%s' does not exist", alias)
	}

	configFile.Active = alias
	return cm.saveConfigFile(configFile)
}

// GetActive returns the active configuration
func (cm *Manager) GetActive() (*APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}

	if configFile.Active == "" {
		return nil, fmt.Errorf("no active configuration set")
	}

	for _, config := range configFile.Configs {
		if config.Alias == configFile.Active {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("active configuration '%s' does not exist", configFile.Active)
}

// GetActiveName returns the active configuration name
func (cm *Manager) GetActiveName() (string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return "", err
	}
	return configFile.Active, nil
}

// validateConfig validates a configuration
func (cm *Manager) validateConfig(config APIConfig) error {
	if config.Alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}

	// Default provider is anthropic
	providerName := config.Provider
	if providerName == "" {
		providerName = "anthropic"
	}

	// 至少需要一种认证方式
	if config.APIKey == "" && config.AuthToken == "" {
		return fmt.Errorf("API key and auth token cannot both be empty")
	}

	// Validate provider
	provider, err := providers.Get(providerName)
	if err != nil {
		return fmt.Errorf("unknown API provider: %s", providerName)
	}

	// Provider-specific validation
	if err := provider.ValidateConfig(config.BaseURL, config.APIKey, config.AuthToken); err != nil {
		return err
	}

	// URL format validation
	if config.BaseURL != "" {
		if !utils.ValidateURL(config.BaseURL) {
			return fmt.Errorf("invalid URL format: %s", config.BaseURL)
		}
	}

	return nil
}

// UpdatePartial updates only the specified fields of a configuration
func (cm *Manager) UpdatePartial(alias string, updates map[string]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	for i, config := range configFile.Configs {
		if config.Alias == alias {
			// Update only the fields that are provided
			if apiKey, ok := updates["api_key"]; ok {
				configFile.Configs[i].APIKey = apiKey
			}
			if authToken, ok := updates["auth_token"]; ok {
				configFile.Configs[i].AuthToken = authToken
			}
			if baseURL, ok := updates["base_url"]; ok {
				configFile.Configs[i].BaseURL = baseURL
			}
			if model, ok := updates["model"]; ok {
				configFile.Configs[i].Model = model
			}

			// Validate the updated config
			if err := cm.validateConfig(configFile.Configs[i]); err != nil {
				return err
			}

			return cm.saveConfigFile(configFile)
		}
	}

	return fmt.Errorf("configuration '%s' does not exist", alias)
}

// RenameAlias renames a configuration alias
func (cm *Manager) RenameAlias(oldAlias, newAlias string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	// Check if new alias already exists
	for _, cfg := range configFile.Configs {
		if cfg.Alias == newAlias {
			return fmt.Errorf("configuration '%s' already exists", newAlias)
		}
	}

	// Find and rename
	found := false
	for i, cfg := range configFile.Configs {
		if cfg.Alias == oldAlias {
			configFile.Configs[i].Alias = newAlias
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("configuration '%s' does not exist", oldAlias)
	}

	// Update active config if needed
	if configFile.Active == oldAlias {
		configFile.Active = newAlias
	}

	return cm.saveConfigFile(configFile)
}

// GenerateActiveScript generates the activation script for active configuration
func (cm *Manager) GenerateActiveScript() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		// No active configuration, clean up active.env file
		activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
		os.Remove(activeEnvPath)
		return nil
	}

	var active *APIConfig
	if configFile.Active != "" {
		for _, config := range configFile.Configs {
			if config.Alias == configFile.Active {
				// Create a copy to avoid implicit memory aliasing
				activeCopy := config
				active = &activeCopy
				break
			}
		}
	}

	if active == nil {
		// No active configuration, clean up active.env file
		activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
		os.Remove(activeEnvPath)
		return nil
	}

	// Generate activation script content
	envScript := generateEnvScript(active)

	// Write to file
	activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
	if err := os.WriteFile(activeEnvPath, []byte(envScript), 0600); err != nil {
		return err
	}

	// Sync to global Claude Code settings (optional feature, doesn't affect main flow)
	if syncErr := cm.syncClaudeSettings(active); syncErr != nil {
		fmt.Printf("⚠️  Failed to sync to global Claude Code settings: %v\n", syncErr)
	}

	// Sync to project-level config file (if .claude directory exists)
	if syncErr := cm.syncProjectClaudeConfig(active); syncErr != nil {
		// Failed project-level sync won't error, since not all projects have a .claude directory
		fmt.Printf("⚠️  Failed to sync to project-level Claude Code settings: %v\n", syncErr)
	}

	return nil
}

// generateEnvScript generates environment variable script content
func generateEnvScript(cfg *APIConfig) string {
	var buf strings.Builder

	// Add comments
	buf.WriteString("# Auto-generated active configuration - updated on each config change\n")
	buf.WriteString("# Do not edit this file manually\n\n")

	// Clear old environment variables
	buf.WriteString("# Clear previously set environment variables\n")
	buf.WriteString("unset ANTHROPIC_API_KEY\n")
	buf.WriteString("unset ANTHROPIC_AUTH_TOKEN\n")
	buf.WriteString("unset ANTHROPIC_BASE_URL\n")
	buf.WriteString("unset ANTHROPIC_MODEL\n")
	buf.WriteString("unset APIMGR_ACTIVE\n\n")

	// Set new environment variables
	buf.WriteString("# Set new environment variables\n")
	if cfg.APIKey != "" {
		buf.WriteString(fmt.Sprintf("export ANTHROPIC_API_KEY=%q\n", cfg.APIKey))
	}
	if cfg.AuthToken != "" {
		buf.WriteString(fmt.Sprintf("export ANTHROPIC_AUTH_TOKEN=%q\n", cfg.AuthToken))
	}
	if cfg.BaseURL != "" {
		buf.WriteString(fmt.Sprintf("export ANTHROPIC_BASE_URL=%q\n", cfg.BaseURL))
	}
	if cfg.Model != "" {
		buf.WriteString(fmt.Sprintf("export ANTHROPIC_MODEL=%q\n", cfg.Model))
	}
	buf.WriteString(fmt.Sprintf("export APIMGR_ACTIVE=%q\n", cfg.Alias))

	return buf.String()
}

// syncClaudeSettings syncs configuration to global Claude Code settings file
func (cm *Manager) syncClaudeSettings(cfg *APIConfig) error {
	claudeSettingsPath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")

	// Check if Claude Code config file exists
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		// File doesn't exist, skip sync
		return nil
	}

	// Read existing settings
	data, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		return fmt.Errorf("Failed to read global Claude Code settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("Failed to parse global Claude Code settings: %v", err)
	}

	// Ensure env field exists
	if settings["env"] == nil {
		settings["env"] = make(map[string]interface{})
	}

	env := settings["env"].(map[string]interface{})

	// Update environment variables
	// Clear old ANTHROPIC related variables
	delete(env, "ANTHROPIC_API_KEY")
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, "ANTHROPIC_BASE_URL")
	delete(env, "ANTHROPIC_MODEL")

	// Set new environment variables
	if cfg.APIKey != "" {
		env["ANTHROPIC_API_KEY"] = cfg.APIKey
	}
	if cfg.AuthToken != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = cfg.AuthToken
	}
	if cfg.BaseURL != "" {
		env["ANTHROPIC_BASE_URL"] = cfg.BaseURL
	}
	if cfg.Model != "" {
		env["ANTHROPIC_MODEL"] = cfg.Model
	}

	// Write back to file
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize global Claude Code settings: %v", err)
	}

	if err := os.WriteFile(claudeSettingsPath, updatedData, 0600); err != nil {
		return fmt.Errorf("Failed to write global Claude Code settings: %v", err)
	}

	return nil
}

// syncProjectClaudeConfig syncs configuration to project-level .claude/settings.json
func (cm *Manager) syncProjectClaudeConfig(cfg *APIConfig) error {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Project-level config file path
	projectClaudePath := filepath.Join(workDir, ".claude", "settings.json")

	// Check if project has .claude directory
	if _, err := os.Stat(projectClaudePath); os.IsNotExist(err) {
		// No project-level config, skip
		return nil
	}

	// Read project-level config
	data, err := os.ReadFile(projectClaudePath)
	if err != nil {
		return fmt.Errorf("Failed to read project-level Claude Code settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("Failed to parse project-level Claude Code settings: %v", err)
	}

	// Update env field
	if settings["env"] == nil {
		settings["env"] = make(map[string]interface{})
	}

	env := settings["env"].(map[string]interface{})

	// Clear old ANTHROPIC related variables
	delete(env, "ANTHROPIC_API_KEY")
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, "ANTHROPIC_BASE_URL")
	delete(env, "ANTHROPIC_MODEL")

	// Set new environment variables
	if cfg.APIKey != "" {
		env["ANTHROPIC_API_KEY"] = cfg.APIKey
	}
	if cfg.AuthToken != "" {
		env["ANTHROPIC_AUTH_TOKEN"] = cfg.AuthToken
	}
	if cfg.BaseURL != "" {
		env["ANTHROPIC_BASE_URL"] = cfg.BaseURL
	}
	if cfg.Model != "" {
		env["ANTHROPIC_MODEL"] = cfg.Model
	}

	// Write back to file
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize project-level Claude Code settings: %v", err)
	}

	if err := os.WriteFile(projectClaudePath, updatedData, 0600); err != nil {
		return fmt.Errorf("Failed to write project-level Claude Code settings: %v", err)
	}

	fmt.Printf("✅ Project-level Claude Code configuration updated: %s\n", projectClaudePath)
	return nil
}


// SessionMarker represents a local session marker file
type SessionMarker struct {
	PID       string    `json:"pid"`
	Alias     string    `json:"alias"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateSessionMarker creates a session marker file for local mode
func (cm *Manager) CreateSessionMarker(pid string, alias string) error {
	marker := SessionMarker{
		PID:       pid,
		Alias:     alias,
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize session marker: %v", err)
	}

	markerPath := filepath.Join(filepath.Dir(cm.configPath), fmt.Sprintf("session-%s", pid))
	if err := os.WriteFile(markerPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session marker: %v", err)
	}

	return nil
}

// CleanupSession removes a session marker file
func (cm *Manager) CleanupSession(pid string) error {
	markerPath := filepath.Join(filepath.Dir(cm.configPath), fmt.Sprintf("session-%s", pid))
	err := os.Remove(markerPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session marker: %v", err)
	}
	return nil
}

// HasActiveLocalSessions checks if there are any active local sessions
// It also cleans up stale session files (PIDs that no longer exist)
func (cm *Manager) HasActiveLocalSessions() (bool, error) {
	configDir := filepath.Dir(cm.configPath)
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return false, fmt.Errorf("failed to read config directory: %v", err)
	}

	hasActive := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "session-") {
			continue
		}

		// Extract PID from filename
		pidStr := strings.TrimPrefix(name, "session-")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			// Invalid session file name, clean it up
			os.Remove(filepath.Join(configDir, name))
			continue
		}

		// Check if process is still running
		if isProcessRunning(pid) {
			hasActive = true
		} else {
			// Clean up stale session file
			os.Remove(filepath.Join(configDir, name))
		}
	}

	return hasActive, nil
}

// isProcessRunning checks if a process with the given PID is still running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// SyncClaudeSettingsOnly syncs configuration to Claude Code settings files
// without updating global active field or generating active.env file.
// This is used for local mode to update Claude Code immediately.
func (cm *Manager) SyncClaudeSettingsOnly(cfg *APIConfig) error {
	// Sync to global Claude Code settings
	if err := cm.syncClaudeSettings(cfg); err != nil {
		return fmt.Errorf("failed to sync to global Claude Code settings: %v", err)
	}

	// Sync to project-level Claude Code settings (if exists)
	if err := cm.syncProjectClaudeConfig(cfg); err != nil {
		// Project-level sync failure is not critical, just log it
		fmt.Printf("⚠️  Failed to sync to project-level Claude Code settings: %v\n", err)
	}

	return nil
}

// RestoreClaudeToGlobal restores Claude Code settings to match the global active configuration.
// If no global active configuration exists, it clears the ANTHROPIC_* env vars from Claude Code settings.
func (cm *Manager) RestoreClaudeToGlobal() error {
	// Get global active configuration
	activeConfig, err := cm.GetActive()
	if err != nil {
		// No global active configuration, clear Claude Code settings
		return cm.clearClaudeSettings()
	}

	// Sync global active configuration to Claude Code
	return cm.SyncClaudeSettingsOnly(activeConfig)
}

// clearClaudeSettings removes ANTHROPIC_* environment variables from Claude Code settings files
func (cm *Manager) clearClaudeSettings() error {
	// Clear global Claude Code settings
	if err := cm.clearGlobalClaudeSettings(); err != nil {
		return fmt.Errorf("failed to clear global Claude Code settings: %v", err)
	}

	// Clear project-level Claude Code settings (if exists)
	if err := cm.clearProjectClaudeSettings(); err != nil {
		// Project-level clear failure is not critical, just log it
		fmt.Printf("⚠️  Failed to clear project-level Claude Code settings: %v\n", err)
	}

	return nil
}

// clearGlobalClaudeSettings removes ANTHROPIC_* env vars from global Claude Code settings
func (cm *Manager) clearGlobalClaudeSettings() error {
	claudeSettingsPath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")

	// Check if Claude Code config file exists
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		// File doesn't exist, nothing to clear
		return nil
	}

	// Read existing settings
	data, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		return fmt.Errorf("failed to read global Claude Code settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse global Claude Code settings: %v", err)
	}

	// Check if env field exists
	if settings["env"] == nil {
		// No env field, nothing to clear
		return nil
	}

	env := settings["env"].(map[string]interface{})

	// Clear ANTHROPIC related variables
	delete(env, "ANTHROPIC_API_KEY")
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, "ANTHROPIC_BASE_URL")
	delete(env, "ANTHROPIC_MODEL")

	// Write back to file
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize global Claude Code settings: %v", err)
	}

	if err := os.WriteFile(claudeSettingsPath, updatedData, 0600); err != nil {
		return fmt.Errorf("failed to write global Claude Code settings: %v", err)
	}

	return nil
}

// clearProjectClaudeSettings removes ANTHROPIC_* env vars from project-level Claude Code settings
func (cm *Manager) clearProjectClaudeSettings() error {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Project-level config file path
	projectClaudePath := filepath.Join(workDir, ".claude", "settings.json")

	// Check if project has .claude directory
	if _, err := os.Stat(projectClaudePath); os.IsNotExist(err) {
		// No project-level config, nothing to clear
		return nil
	}

	// Read project-level config
	data, err := os.ReadFile(projectClaudePath)
	if err != nil {
		return fmt.Errorf("failed to read project-level Claude Code settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse project-level Claude Code settings: %v", err)
	}

	// Check if env field exists
	if settings["env"] == nil {
		// No env field, nothing to clear
		return nil
	}

	env := settings["env"].(map[string]interface{})

	// Clear ANTHROPIC related variables
	delete(env, "ANTHROPIC_API_KEY")
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, "ANTHROPIC_BASE_URL")
	delete(env, "ANTHROPIC_MODEL")

	// Write back to file
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize project-level Claude Code settings: %v", err)
	}

	if err := os.WriteFile(projectClaudePath, updatedData, 0600); err != nil {
		return fmt.Errorf("failed to write project-level Claude Code settings: %v", err)
	}

	return nil
}
