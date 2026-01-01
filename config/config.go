package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"apimgr/internal/providers"
	"apimgr/internal/utils"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// APIConfig represents a single API configuration
type APIConfig struct {
	Alias     string   `json:"alias"`
	Provider  string   `json:"provider"` // API provider type
	APIKey    string   `json:"api_key"`
	AuthToken string   `json:"auth_token"`
	BaseURL   string   `json:"base_url"`
	Model     string   `json:"model"`            // Currently active model
	Models    []string `json:"models,omitempty"` // Supported models list
}

// Feature flags for experimental features
const FeatureSurgicalUpdates = "surgical_updates_v1"

// Backup constants
const (
	// DefaultBackupRetention is the default number of backups to keep
	DefaultBackupRetention = 3
)

// BackupManager manages backup files for configurations
type BackupManager struct {
	// MaxBackups is the maximum number of backups to retain
	MaxBackups int
}

// NewBackupManager creates a new BackupManager with default settings
func NewBackupManager(maxBackups int) *BackupManager {
	if maxBackups <= 0 {
		maxBackups = DefaultBackupRetention
	}
	return &BackupManager{
		MaxBackups: maxBackups,
	}
}

// CreateBackup creates a new backup file with timestamp-PID naming format
func (bm *BackupManager) CreateBackup(filePath string) (string, error) {
	// Get PID for the backup filename
	pid := syscall.Getpid()

	// Create backup filename with pattern: original.backup-YYYYMMDDHHMMSS-PID
	timestamp := time.Now().Format("20060102150405")
	backupPath := fmt.Sprintf("%s.backup-%s-%d", filePath, timestamp, pid)

	// Copy the file to create backup
	if err := copyFile(filePath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// Preserve file permissions
	srcInfo, err := os.Stat(filePath)
	if err != nil {
		return backupPath, nil // Non-fatal, backup was created
	}
	if err := os.Chmod(backupPath, srcInfo.Mode()); err != nil {
		return backupPath, nil // Non-fatal, backup was created
	}

	return backupPath, nil
}

// ListBackups returns a list of all backup files for the given filePath
func (bm *BackupManager) ListBackups(filePath string) ([]string, error) {
	// Pattern to match backup files
	pattern := fmt.Sprintf("%s.backup-*", filePath)

	// Find all matching backup files
	backupFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	// Sort backup files by modification time (oldest first)
	sort.Slice(backupFiles, func(i, j int) bool {
		iInfo, err1 := os.Stat(backupFiles[i])
		jInfo, err2 := os.Stat(backupFiles[j])
		if err1 != nil || err2 != nil {
			return false
		}
		return iInfo.ModTime().Before(jInfo.ModTime())
	})

	return backupFiles, nil
}

// CleanupOldBackups removes old backup files, retaining only the most recent MaxBackups
func (bm *BackupManager) CleanupOldBackups(filePath string) error {
	// Get all backup files sorted by modification time (oldest first)
	backupFiles, err := bm.ListBackups(filePath)
	if err != nil {
		return err
	}

	// Calculate how many backups to remove
	numToRemove := len(backupFiles) - bm.MaxBackups
	if numToRemove <= 0 {
		return nil // No old backups to remove
	}

	// Remove old backups (from the beginning of the sorted list)
	for _, oldBackup := range backupFiles[:numToRemove] {
		if err := os.Remove(oldBackup); err != nil {
			return fmt.Errorf("failed to remove old backup %s: %w", oldBackup, err)
		}
	}

	return nil
}

// RestoreFromBackup restores the file from a specific backup path
func (bm *BackupManager) RestoreFromBackup(filePath string, backupPath string) error {
	// Validate the backup file path
	pattern := fmt.Sprintf("%s.backup-*", filePath)
	match, err := filepath.Match(pattern, backupPath)
	if err != nil {
		return fmt.Errorf("invalid backup path: %w", err)
	}
	if !match {
		return fmt.Errorf("backup path %s is not a valid backup for %s", backupPath, filePath)
	}

	// Copy the backup to original file
	if err := copyFile(backupPath, filePath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Restore file permissions
	srcInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil // Non-fatal, restore was successful
	}
	return os.Chmod(filePath, srcInfo.Mode())
}

// RestoreFromLatestBackup restores the file from the most recent backup
func (bm *BackupManager) RestoreFromLatestBackup(filePath string) error {
	// Get all backup files sorted by modification time (oldest first)
	backupFiles, err := bm.ListBackups(filePath)
	if err != nil {
		return err
	}

	if len(backupFiles) == 0 {
		return fmt.Errorf("no backup files found for %s", filePath)
	}

	// Get the latest backup (last in sorted list)
	latestBackup := backupFiles[len(backupFiles)-1]

	// Restore from the latest backup
	return bm.RestoreFromBackup(filePath, latestBackup)
}

// SyncOptions provides options for synchronization
type SyncOptions struct {
	DryRun        bool  // 仅验证，不写入
	CreateBackup  bool  // 更新前创建备份
	PreserveOther bool  // 保留非 ANTHROPIC 环境变量
}

// File represents the structure of the config file
type File struct {
	Active  string      `json:"active"`
	Configs []APIConfig `json:"configs"`
}

// normalizeModels ensures backward compatibility for configs loaded without models field.
// If models field is empty but model field has a value, populate models from model.
// If model field is empty, models list remains empty.
func normalizeModels(config *APIConfig) {
	if len(config.Models) == 0 && config.Model != "" {
		config.Models = []string{config.Model}
	}
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
			// Normalize models for backward compatibility
			for i := range configs {
				normalizeModels(&configs[i])
			}
			return &File{Configs: configs}, nil
		}
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Normalize models for backward compatibility
	for i := range configFile.Configs {
		normalizeModels(&configFile.Configs[i])
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
// Uses surgical update mechanism to preserve JSON structure and non-ANTHROPIC fields
func (cm *Manager) syncClaudeSettings(cfg *APIConfig) error {
	claudeSettingsPath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")

	// Check if Claude Code config file exists
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		// File doesn't exist, skip sync
		return nil
	}

	// Read existing settings content (raw to preserve structure and comments)
	originalContent, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		return fmt.Errorf("Failed to read global Claude Code settings: %v", err)
	}

	// Create synchronization options
	opts := SyncOptions{
		DryRun:        false,
		CreateBackup:  true,  // Create backup before update to ensure data safety
		PreserveOther: true,  // Preserve non-ANTHROPIC environment variables
	}

	// Perform surgical update using sjson
	updatedContent, err := updateEnvField(string(originalContent), cfg, opts)
	if err != nil {
		return fmt.Errorf("Failed to update settings content: %v", err)
	}

	// Write back to file using atomic update to prevent data corruption
	if err := atomicFileUpdate(claudeSettingsPath, updatedContent, true); err != nil {
		// Attempt to restore from backup if update fails
		restoreErr := restoreFromBackup(claudeSettingsPath)
		if restoreErr != nil {
			return fmt.Errorf("Failed to write settings file and restore from backup: update error=%v, restore error=%v", err, restoreErr)
		}
		return fmt.Errorf("Failed to write settings file but restored from backup: %v", err)
	}

	return nil
}

// showEnvChanges displays the changes between old and new env maps
func showEnvChanges(oldEnv, newEnv map[string]interface{}) {
	// Create a map of all variables
	allVars := make(map[string]bool)
	for k := range oldEnv {
		allVars[k] = true
	}
	for k := range newEnv {
		allVars[k] = true
	}

	// Create sorted list of variables
	var sortedVars []string
	for k := range allVars {
		sortedVars = append(sortedVars, k)
	}
	sort.Strings(sortedVars)

	// Print table header
	fmt.Printf("┌──────────────────────────────┬─────────────────┬─────────────────┐\n")
	fmt.Printf("│ Variable                     │ Old Value       │ New Value       │\n")
	fmt.Printf("├──────────────────────────────┼─────────────────┼─────────────────┤\n")

	// Calculate summary counts
	updatedCount, addedCount, deletedCount, preservedCount := 0, 0, 0, 0

	// Print table rows
	for _, varName := range sortedVars {
		oldVal, oldExists := oldEnv[varName]
		newVal, newExists := newEnv[varName]

		// Determine the type of change
		var status string
		if !oldExists {
			status = "added"
			addedCount++
		} else if !newExists {
			status = "deleted"
			deletedCount++
		} else if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			status = "updated"
			updatedCount++
		} else {
			status = "preserved"
			preservedCount++
		}

		// Format values for display
		oldValStr := fmt.Sprintf("%v", oldVal)
		if !oldExists {
			oldValStr = "(not set)"
		}
		newValStr := fmt.Sprintf("%v", newVal)
		if !newExists {
			newValStr = "(deleted)"
		}
		if status == "preserved" {
			newValStr = "(unchanged)"
		}

		// Truncate long strings for table display
		const maxLen = 20
		if len(oldValStr) > maxLen {
			oldValStr = oldValStr[:maxLen-3] + "..."
		}
		if len(newValStr) > maxLen {
			newValStr = newValStr[:maxLen-3] + "..."
		}

		// Print the table row
		fmt.Printf("│ %-38s │ %-20s │ %-20s │\n", varName, oldValStr, newValStr)
	}

	// Print table footer and summary
	fmt.Printf("└──────────────────────────────┴─────────────────┴─────────────────┘\n\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("• %d variable(s) will be updated\n", updatedCount)
	fmt.Printf("• %d variable(s) will be added\n", addedCount)
	fmt.Printf("• %d variable(s) will be deleted\n", deletedCount)
	fmt.Printf("• %d variable(s) will be preserved\n", preservedCount)
}

// generateDiffReport generates a human-readable report showing differences between old and new content
func generateDiffReport(originalContent, updatedContent string) (string, error) {
	// Extract env fields from both
	originalEnv, err := extractEnv(originalContent)
	if err != nil {
		return "", err
	}

	updatedEnv, err := extractEnv(updatedContent)
	if err != nil {
		return "", err
	}

	// Create a report buffer
	var report strings.Builder

	// Set up table headers
	report.WriteString("┌──────────────────────────────┬─────────────────┬─────────────────┐\n")
	report.WriteString("│ Variable                     │ Old Value       │ New Value       │\n")
	report.WriteString("├──────────────────────────────┼─────────────────┼─────────────────┤\n")

	// Create a map of all variables
	allVars := make(map[string]bool)
	for k := range originalEnv {
		allVars[k] = true
	}
	for k := range updatedEnv {
		allVars[k] = true
	}

	// Create sorted list of variables
	var sortedVars []string
	for k := range allVars {
		sortedVars = append(sortedVars, k)
	}
	sort.Strings(sortedVars)

	// Calculate summary counts
	updatedCount, addedCount, deletedCount, preservedCount := 0, 0, 0, 0

	// Generate table rows
	for _, varName := range sortedVars {
		oldVal, oldExists := originalEnv[varName]
		newVal, newExists := updatedEnv[varName]

		// Determine the type of change
		if !oldExists {
			addedCount++
		} else if !newExists {
			deletedCount++
		} else if fmt.Sprintf("%v", oldVal) != fmt.Sprintf("%v", newVal) {
			updatedCount++
		} else {
			preservedCount++
		}

		// Format values for display
		oldValStr := fmt.Sprintf("%v", oldVal)
		if !oldExists {
			oldValStr = "(not set)"
		}
		newValStr := fmt.Sprintf("%v", newVal)
		if !newExists {
			newValStr = "(deleted)"
		}
		if oldExists && newExists && fmt.Sprintf("%v", oldVal) == fmt.Sprintf("%v", newVal) {
			newValStr = "(unchanged)"
		}

		// Truncate long strings for table display
		const maxLen = 20
		if len(oldValStr) > maxLen {
			oldValStr = oldValStr[:maxLen-3] + "..."
		}
		if len(newValStr) > maxLen {
			newValStr = newValStr[:maxLen-3] + "..."
		}

		// Write the table row
		report.WriteString(fmt.Sprintf("│ %-38s │ %-20s │ %-20s │\n", varName, oldValStr, newValStr))
	}

	// Close table
	report.WriteString("└──────────────────────────────┴─────────────────┴─────────────────┘\n\n")

	// Write summary
	report.WriteString(fmt.Sprintf("Summary:\n"))
	report.WriteString(fmt.Sprintf("• %d variable(s) will be updated\n", updatedCount))
	report.WriteString(fmt.Sprintf("• %d variable(s) will be added\n", addedCount))
	report.WriteString(fmt.Sprintf("• %d variable(s) will be deleted\n", deletedCount))
	report.WriteString(fmt.Sprintf("• %d variable(s) will be preserved\n", preservedCount))

	return report.String(), nil
}

// atomicFileUpdate ensures atomic file update to prevent data corruption
func atomicFileUpdate(filePath string, newContent string, createBackup bool) error {
	// Create backup if requested
	if createBackup {
		bm := NewBackupManager(DefaultBackupRetention)
		if _, err := bm.CreateBackup(filePath); err != nil {
			return fmt.Errorf("failed to create backup file: %w", err)
		}
	}

	// Create temporary file in the same directory
	tmpFile, err := os.CreateTemp(filepath.Dir(filePath), "settings.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up on failure

	// Write new content to temporary file
	if _, err := tmpFile.WriteString(newContent); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close()

	// Change file permissions to match existing file (0600)
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		return fmt.Errorf("failed to set permissions on temporary file: %w", err)
	}

	// Atomic rename - this is guaranteed to be atomic on all POSIX systems
	if err := os.Rename(tmpFile.Name(), filePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	// Cleanup old backups after successful update
	if createBackup {
		bm := NewBackupManager(DefaultBackupRetention)
		if err := bm.CleanupOldBackups(filePath); err != nil {
			// Non-fatal error, update was successful
			fmt.Printf("⚠️  Failed to cleanup old backups: %v\n", err)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve permissions from source file
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// restoreFromBackup restores a file from its most recent backup (legacy wrapper for backward compatibility)
func restoreFromBackup(filePath string) error {
	bm := NewBackupManager(DefaultBackupRetention)
	return bm.RestoreFromLatestBackup(filePath)
}

// parseToMaps parses two JSON strings to maps for deep comparison
func parseToMaps(originalStr, updatedStr string) (map[string]interface{}, map[string]interface{}, error) {
	var original map[string]interface{}
	if err := json.Unmarshal([]byte(originalStr), &original); err != nil {
		return nil, nil, fmt.Errorf("failed to parse original JSON: %w", err)
	}

	var updated map[string]interface{}
	if err := json.Unmarshal([]byte(updatedStr), &updated); err != nil {
		return nil, nil, fmt.Errorf("failed to parse updated JSON: %w", err)
	}

	return original, updated, nil
}

// deepCompare compares two maps and returns a list of differing fields
func deepCompare(original, updated map[string]interface{}) []string {
	var differences []string

	// Check all keys in original
	for key, originalVal := range original {
		if key == "env" {
			// Skip env field for now, it will be checked separately
			continue
		}

		if updatedVal, exists := updated[key]; exists {
			// Check if values are maps for deep comparison
			originalMap, originalIsMap := originalVal.(map[string]interface{})
			updatedMap, updatedIsMap := updatedVal.(map[string]interface{})

			if originalIsMap && updatedIsMap {
				// Recursively compare nested maps
				nestedDiffs := deepCompare(originalMap, updatedMap)
				for _, diff := range nestedDiffs {
					differences = append(differences, key+"."+diff)
				}
			} else {
				// Compare values directly
				if fmt.Sprintf("%v", originalVal) != fmt.Sprintf("%v", updatedVal) {
					differences = append(differences, key)
				}
			}
		} else {
			// Key missing in updated
			differences = append(differences, key+" (missing)")
		}
	}

	// Check for new keys in updated
	for key := range updated {
		if key == "env" {
			continue // Skip env field, checked separately
		}
		if _, exists := original[key]; !exists {
			differences = append(differences, key+" (new)")
		}
	}

	return differences
}

// extractEnv extracts the env field from JSON content
func extractEnv(jsonContent string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return nil, err
	}

	env, exists := data["env"]
	if !exists {
		return make(map[string]interface{}), nil // Return empty map if env doesn't exist
	}

	envMap, ok := env.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("env field is not a map")
	}

	return envMap, nil
}

// validateJSONUpdate validates that only the env field has changed in the JSON
// Requirements: 5.1
func validateJSONUpdate(originalContent string, updatedContent string) error {
	// 1. Validate JSON validity
	if !json.Valid([]byte(originalContent)) {
		return fmt.Errorf("original JSON is invalid")
	}

	if !json.Valid([]byte(updatedContent)) {
		return fmt.Errorf("updated JSON is invalid")
	}

	// 2. Ensure only env field has changed (excluding ANTHROPIC_ fields)
	original, updated, err := parseToMaps(originalContent, updatedContent)
	if err != nil {
		return err
	}

	// Compare all fields except env
	differences := deepCompare(original, updated)
	if len(differences) > 0 {
		return fmt.Errorf("unexpected changes to non-env fields: %s", strings.Join(differences, ", "))
	}

	// 3. Check if non-ANTHROPIC fields were preserved in env
	originalEnv, err := extractEnv(originalContent)
	if err != nil {
		return err
	}

	updatedEnv, err := extractEnv(updatedContent)
	if err != nil {
		return err
	}

	// Check that all non-ANTHROPIC fields are preserved
	for key, originalVal := range originalEnv {
		if !strings.HasPrefix(strings.ToUpper(key), "ANTHROPIC_") {
			if updatedVal, exists := updatedEnv[key]; exists {
				if fmt.Sprintf("%v", originalVal) != fmt.Sprintf("%v", updatedVal) {
					return fmt.Errorf("non-ANTHROPIC field '%s' was modified", key)
				}
			} else {
				return fmt.Errorf("non-ANTHROPIC field '%s' was deleted", key)
			}
		}
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

	// Read project-level config content (raw to preserve structure and comments)
	originalContent, err := os.ReadFile(projectClaudePath)
	if err != nil {
		return fmt.Errorf("Failed to read project-level Claude Code settings: %v", err)
	}

	// Create synchronization options
	opts := SyncOptions{
		DryRun:        false,
		CreateBackup:  true,  // Create backup before update to ensure data safety
		PreserveOther: true,  // Preserve non-ANTHROPIC environment variables
	}

	// Perform surgical update using sjson
	updatedContent, err := updateEnvField(string(originalContent), cfg, opts)
	if err != nil {
		return fmt.Errorf("Failed to update project-level settings content: %v", err)
	}

	// Write back to file using atomic update to prevent data corruption
	if err := atomicFileUpdate(projectClaudePath, updatedContent, true); err != nil {
		// Attempt to restore from backup if update fails
		restoreErr := restoreFromBackup(projectClaudePath)
		if restoreErr != nil {
			return fmt.Errorf("Failed to write project-level settings file and restore from backup: update error=%v, restore error=%v", err, restoreErr)
		}
		return fmt.Errorf("Failed to write project-level settings file but restored from backup: %v", err)
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

// SwitchModel switches the active model for a configuration.
// It validates that the model is in the supported models list before switching.
// Requirements: 2.1
func (cm *Manager) SwitchModel(alias string, model string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	// Find the configuration by alias
	for i, config := range configFile.Configs {
		if config.Alias == alias {
			// Validate model is in supported list
			validator := NewModelValidator()
			if err := validator.ValidateModelInList(model, config.Models); err != nil {
				return err
			}

			// Update active model
			configFile.Configs[i].Model = model

			// Save configuration
			return cm.saveConfigFile(configFile)
		}
	}

	return fmt.Errorf("configuration '%s' does not exist", alias)
}

// GetModels returns the supported models list for a configuration.
// Requirements: 4.1
func (cm *Manager) GetModels(alias string) ([]string, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}

	for _, config := range configFile.Configs {
		if config.Alias == alias {
			// Return a copy to prevent external modification
			result := make([]string, len(config.Models))
			copy(result, config.Models)
			return result, nil
		}
	}

	return nil, fmt.Errorf("configuration '%s' does not exist", alias)
}

// SetModels updates the supported models list for a configuration.
// It validates the models list and handles active model fallback when the current active model is removed.
// Requirements: 4.1, 4.2, 4.3
func (cm *Manager) SetModels(alias string, models []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate and normalize the models list
	validator := NewModelValidator()
	normalizedModels := validator.NormalizeModels(models)
	if err := validator.ValidateModelsList(normalizedModels); err != nil {
		return err
	}

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	// Find the configuration by alias
	for i, config := range configFile.Configs {
		if config.Alias == alias {
			// Update models list
			configFile.Configs[i].Models = normalizedModels

			// Handle active model fallback when removed
			// Check if current active model is still in the new list
			activeModelInList := false
			for _, m := range normalizedModels {
				if m == config.Model {
					activeModelInList = true
					break
				}
			}

			// If active model is not in the new list, fallback to first model
			if !activeModelInList && len(normalizedModels) > 0 {
				configFile.Configs[i].Model = normalizedModels[0]
			}

			// Save configuration
			return cm.saveConfigFile(configFile)
		}
	}

	return fmt.Errorf("configuration '%s' does not exist", alias)
}

// updateEnvField updates the env field in Claude Code configuration JSON
// It only updates ANTHROPIC_ fields and preserves non-ANTHROPIC fields when PreserveOther is true
func updateEnvField(originalContent string, cfg *APIConfig, opts SyncOptions) (string, error) {
	// Parse the JSON content to verify it's valid
	result := gjson.Parse(originalContent)
	if !result.Exists() {
		return "", fmt.Errorf("invalid JSON content")
	}

	// Extract existing env field if it exists
	existingEnv := make(map[string]string)
	if envResult := result.Get("env"); envResult.Exists() {
		envResult.ForEach(func(key, value gjson.Result) bool {
			existingEnv[key.Str] = value.Str
			return true
		})
	}

	// Create updated env map
	updatedEnv := make(map[string]string)

	// Preserve non-ANTHROPIC environment variables if requested
	if opts.PreserveOther {
		for key, value := range existingEnv {
			if !strings.HasPrefix(strings.ToUpper(key), "ANTHROPIC_") {
				updatedEnv[key] = value
			}
		}
	}

	// Set new ANTHROPIC values (only non-empty values)
	if cfg.APIKey != "" {
		updatedEnv["ANTHROPIC_API_KEY"] = cfg.APIKey
	}
	if cfg.Model != "" {
		updatedEnv["ANTHROPIC_MODEL"] = cfg.Model
	}
	if cfg.AuthToken != "" {
		updatedEnv["ANTHROPIC_AUTH_TOKEN"] = cfg.AuthToken
	}
	if cfg.BaseURL != "" {
		updatedEnv["ANTHROPIC_BASE_URL"] = cfg.BaseURL
	}

	// Convert updatedEnv to JSON string
	envJSON, err := json.Marshal(updatedEnv)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated env: %w", err)
	}

	// Use sjson.SetRaw to update the env field precisely
	updatedContent, err := sjson.SetRaw(originalContent, "env", string(envJSON))
	if err != nil {
		return "", fmt.Errorf("failed to update env field: %w", err)
	}

	// Validate the update to ensure only env field has changed and non-ANTHROPIC fields are preserved
	if err := validateJSONUpdate(originalContent, updatedContent); err != nil {
		return "", fmt.Errorf("update validation failed: %w", err)
	}

	return updatedContent, nil
}
