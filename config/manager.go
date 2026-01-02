package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"apimgr/config/models"
	"apimgr/config/storage"
	syncpkg "apimgr/config/sync"
	"apimgr/config/validation"
)

// normalizeModels ensures backward compatibility for configs loaded without models field.
// If models field is empty but model field has a value, populate models from model.
// If model field is empty, models list remains empty.
func normalizeModels(config *models.APIConfig) {
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
	if storage.ShouldMigrateConfig(oldConfigPath, xdgConfigPath) {
		if err := storage.MigrateConfig(oldConfigPath, xdgConfigPath); err != nil {
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

// GetConfigPath returns the path to the config file
func (cm *Manager) GetConfigPath() string {
	return cm.configPath
}

// loadConfigFile loads the config file with locking
func (cm *Manager) loadConfigFile() (*models.File, error) {
	// Open the file with read lock
	file, err := os.OpenFile(cm.configPath, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.File{Configs: []models.APIConfig{}}, nil
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
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if len(data) == 0 {
		return &models.File{Configs: []models.APIConfig{}}, nil
	}

	var configFile models.File
	err = json.Unmarshal(data, &configFile)
	if err != nil {
		// Try to parse as old format (array of configs)
		var configs []models.APIConfig
		if err2 := json.Unmarshal(data, &configs); err2 == nil {
			// Normalize models for backward compatibility
			for i := range configs {
				normalizeModels(&configs[i])
			}
			return &models.File{Configs: configs}, nil
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
func (cm *Manager) saveConfigFile(configFile *models.File) error {
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
func (cm *Manager) Load() ([]models.APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}
	return configFile.Configs, nil
}

// Save saves configurations to the config file
func (cm *Manager) Save(configs []models.APIConfig) error {
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
func (cm *Manager) Add(config models.APIConfig) error {
	// Set default provider
	if config.Provider == "" {
		config.Provider = "anthropic"
	}

	validator := validation.NewValidator()
	if err := validator.ValidateConfig(config); err != nil {
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
func (cm *Manager) Get(alias string) (*models.APIConfig, error) {
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
func (cm *Manager) List() ([]models.APIConfig, error) {
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
	if err := cm.saveConfigFile(configFile); err != nil {
		return err
	}

	return cm.generateActiveScript()
}

// GetActive returns the active configuration
func (cm *Manager) GetActive() (*models.APIConfig, error) {
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
			validator := validation.NewValidator()
			if err := validator.ValidateConfig(configFile.Configs[i]); err != nil {
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

// SwitchModel switches the active model for a configuration.
// It validates that the model is in the supported models list before switching.
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
			validator := validation.NewModelValidator()
			if err := validator.ValidateModelInList(model, config.Models); err != nil {
				return err
			}

			// Update active model
			configFile.Configs[i].Model = model

			// Save configuration
			if err := cm.saveConfigFile(configFile); err != nil {
				return err
			}

			// If this is the active configuration, update the active.env
			if configFile.Active == alias {
				return cm.generateActiveScript()
			}

			return nil
		}
	}

	return fmt.Errorf("configuration '%s' does not exist", alias)
}

// GetModels returns the supported models list for a configuration.
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
func (cm *Manager) SetModels(alias string, models []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate and normalize the models list
	validator := validation.NewModelValidator()
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
			if err := cm.saveConfigFile(configFile); err != nil {
				return err
			}

			// If this is the active configuration, update the active.env
			if configFile.Active == alias {
				return cm.generateActiveScript()
			}

			return nil
		}
	}

	return fmt.Errorf("configuration '%s' does not exist", alias)
}

// GenerateActiveScript generates the activation script for active configuration
func (cm *Manager) GenerateActiveScript() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.generateActiveScript()
}

// generateActiveScript is the internal implementation that generates the activation script.
// It assumes the caller already holds the lock.
func (cm *Manager) generateActiveScript() error {
	configFile, err := cm.loadConfigFile()
	if err != nil {
		// No active configuration, clean up active.env file
		activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
		os.Remove(activeEnvPath)
		return nil
	}

	var active *models.APIConfig
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
	envScript := syncpkg.GenerateEnvScript(active)

	// Write to file
	activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
	if err := os.WriteFile(activeEnvPath, []byte(envScript), 0600); err != nil {
		return err
	}

	// Sync to global Claude Code settings (optional feature, doesn't affect main flow)
	if syncErr := cm.SyncClaudeSettingsOnly(active); syncErr != nil {
		fmt.Printf("⚠️  Failed to sync to global Claude Code settings: %v\n", syncErr)
	}

	return nil
}

// SyncClaudeSettingsOnly syncs configuration to Claude Code settings files
// without updating global active field or generating active.env file.
// This is used for local mode to update Claude Code immediately.
func (cm *Manager) SyncClaudeSettingsOnly(cfg *models.APIConfig) error {
	// Sync to global Claude Code settings
	if err := cm.syncClaudeSettings(cfg); err != nil {
		return fmt.Errorf("failed to sync to global Claude Code settings: %v", err)
	}

	return nil
}

// syncClaudeSettings syncs configuration to global Claude Code settings file
// Uses surgical update mechanism to preserve JSON structure and non-ANTHROPIC fields
func (cm *Manager) syncClaudeSettings(cfg *models.APIConfig) error {
	claudeSettingsPath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")

	// Check if Claude Code config file exists
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		// models.File doesn't exist, skip sync
		return nil
	}

	// Read existing settings content (raw to preserve structure and comments)
	originalContent, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		return fmt.Errorf("Failed to read global Claude Code settings: %v", err)
	}

	// Create synchronization options
	opts := syncpkg.SyncOptions{
		DryRun:        false,
		CreateBackup:  true,  // Create backup before update to ensure data safety
		PreserveOther: true,  // Preserve non-ANTHROPIC environment variables
	}

	// Perform surgical update using sjson
	updatedContent, err := syncpkg.UpdateEnvField(string(originalContent), cfg, opts)
	if err != nil {
		return fmt.Errorf("Failed to update settings content: %v", err)
	}

	// Write back to file using atomic update to prevent data corruption
	if err := storage.AtomicFileUpdate(claudeSettingsPath, updatedContent, true); err != nil {
		// Attempt to restore from backup if update fails
		restoreErr := storage.NewBackupManager(storage.DefaultBackupRetention).RestoreFromLatestBackup(claudeSettingsPath)
		if restoreErr != nil {
			return fmt.Errorf("Failed to write settings file and restore from backup: update error=%v, restore error=%v", err, restoreErr)
		}
		return fmt.Errorf("Failed to write settings file but restored from backup: %v", err)
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

	return nil
}

// clearGlobalClaudeSettings removes ANTHROPIC_* env vars from global Claude Code settings
func (cm *Manager) clearGlobalClaudeSettings() error {
	claudeSettingsPath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")

	// Check if Claude Code config file exists
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		// models.File doesn't exist, nothing to clear
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
