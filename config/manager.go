package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ccasJay/apimgr/config/models"
	"github.com/ccasJay/apimgr/config/storage"
	syncpkg "github.com/ccasJay/apimgr/config/sync"
	"github.com/ccasJay/apimgr/config/validation"
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

func (cm *Manager) openConfigLock(exclusive bool) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(cm.configPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	lockPath := cm.configPath + ".lock"
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open config lock file: %w", err)
	}

	var lockErr error
	if exclusive {
		lockErr = cm.lockFile(file)
	} else {
		lockErr = cm.lockFileShared(file)
	}
	if lockErr != nil {
		_ = file.Close()
		return nil, fmt.Errorf("failed to lock config file: %w", lockErr)
	}

	return file, nil
}

func (cm *Manager) closeConfigLock(file *os.File) error {
	unlockErr := cm.unlockFile(file)
	closeErr := file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

// loadConfigFile loads the config file with shared locking.
func (cm *Manager) loadConfigFile() (*models.File, error) {
	lockFile, err := cm.openConfigLock(false)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cm.closeConfigLock(lockFile)
	}()

	return cm.loadConfigFileLocked()
}

func (cm *Manager) loadConfigFileLocked() (*models.File, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.File{Configs: []models.APIConfig{}}, nil
		}
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

func (cm *Manager) saveConfigFileLocked(configFile *models.File) error {
	data, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	configDir := filepath.Dir(cm.configPath)
	tmpFile, err := os.CreateTemp(configDir, filepath.Base(cm.configPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to sync temporary config file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary config file: %w", err)
	}
	if err := os.Chmod(tmpName, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}
	if err := os.Rename(tmpName, cm.configPath); err != nil {
		return fmt.Errorf("failed to replace config file: %w", err)
	}
	if err := storage.SyncDirectory(configDir); err != nil {
		return fmt.Errorf("failed to sync config directory: %w", err)
	}

	return nil
}

func (cm *Manager) updateConfigFile(update func(*models.File) error) error {
	lockFile, err := cm.openConfigLock(true)
	if err != nil {
		return err
	}
	defer func() {
		_ = cm.closeConfigLock(lockFile)
	}()

	configFile, err := cm.loadConfigFileLocked()
	if err != nil {
		return err
	}
	if err := update(configFile); err != nil {
		return err
	}
	return cm.saveConfigFileLocked(configFile)
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

	return cm.updateConfigFile(func(configFile *models.File) error {
		configFile.Configs = configs
		return nil
	})
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

	return cm.updateConfigFile(func(configs *models.File) error {
		// Check if alias already exists
		for i, existingConfig := range configs.Configs {
			if existingConfig.Alias == config.Alias {
				configs.Configs[i] = config
				return nil
			}
		}

		configs.Configs = append(configs.Configs, config)
		return nil
	})
}

// Remove removes a configuration by alias
func (cm *Manager) Remove(alias string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.updateConfigFile(func(configs *models.File) error {
		for i, config := range configs.Configs {
			if config.Alias == alias {
				configs.Configs = append(configs.Configs[:i], configs.Configs[i+1:]...)
				// If removing the active config, clear the active config
				if configs.Active == alias {
					configs.Active = ""
				}
				return nil
			}
		}

		return fmt.Errorf("configuration '%s' does not exist", alias)
	})
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

	if err := cm.updateConfigFile(func(configFile *models.File) error {
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
		return nil
	}); err != nil {
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

	activeAlias := configFile.Active
	// Check environment variable override
	if envActive := os.Getenv("APIMGR_ACTIVE"); envActive != "" {
		activeAlias = envActive
	}

	if activeAlias == "" {
		return nil, fmt.Errorf("no active configuration set")
	}

	for _, config := range configFile.Configs {
		if config.Alias == activeAlias {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("active configuration '%s' does not exist", activeAlias)
}

// GetActiveName returns the active configuration name
func (cm *Manager) GetActiveName() (string, error) {
	// Check environment variable override first
	if envActive := os.Getenv("APIMGR_ACTIVE"); envActive != "" {
		return envActive, nil
	}

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

	return cm.updateConfigFile(func(configFile *models.File) error {
		for i, config := range configFile.Configs {
			if config.Alias == alias {
				// Update only the fields that are provided
				if apiKey, ok := updates["api_key"]; ok {
					configFile.Configs[i].APIKey = apiKey
					if apiKey != "" {
						configFile.Configs[i].AuthToken = "" // Clear auth token
					}
				}
				if authToken, ok := updates["auth_token"]; ok {
					configFile.Configs[i].AuthToken = authToken
					if authToken != "" {
						configFile.Configs[i].APIKey = "" // Clear API key
					}
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

				return nil
			}
		}

		return fmt.Errorf("configuration '%s' does not exist", alias)
	})
}

// RenameAlias renames a configuration alias
func (cm *Manager) RenameAlias(oldAlias, newAlias string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return cm.updateConfigFile(func(configFile *models.File) error {
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

		return nil
	})
}

// SwitchModel switches the active model for a configuration.
// It validates that the model is in the supported models list before switching.
func (cm *Manager) SwitchModel(alias string, model string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	updateActiveScript := false

	if err := cm.updateConfigFile(func(configFile *models.File) error {
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
				updateActiveScript = configFile.Active == alias

				return nil
			}
		}

		return fmt.Errorf("configuration '%s' does not exist", alias)
	}); err != nil {
		return err
	}

	// If this is the active configuration, update the active.env
	if updateActiveScript {
		return cm.generateActiveScript()
	}

	return nil
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
func (cm *Manager) SetModels(alias string, modelNames []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Validate and normalize the models list
	validator := validation.NewModelValidator()
	normalizedModels := validator.NormalizeModels(modelNames)
	if err := validator.ValidateModelsList(normalizedModels); err != nil {
		return err
	}

	updateActiveScript := false

	if err := cm.updateConfigFile(func(configFile *models.File) error {
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

				updateActiveScript = configFile.Active == alias
				return nil
			}
		}

		return fmt.Errorf("configuration '%s' does not exist", alias)
	}); err != nil {
		return err
	}

	// If this is the active configuration, update the active.env
	if updateActiveScript {
		return cm.generateActiveScript()
	}

	return nil
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

	// Sync to Claude Code settings as a best-effort side effect.
	_ = cm.SyncClaudeSettingsOnly(active)

	return nil
}

// SyncClaudeSettingsOnly syncs configuration to Claude Code settings files
// without updating global active field or generating active.env file.
// This is used for local mode to update Claude Code immediately.
func (cm *Manager) SyncClaudeSettingsOnly(cfg *models.APIConfig) error {
	if err := cm.syncClaudeSettings(cfg); err != nil {
		return fmt.Errorf("failed to sync to Claude Code settings: %v", err)
	}

	return nil
}

func (cm *Manager) claudeSettingsPaths() []string {
	var paths []string
	seen := make(map[string]bool)
	addPath := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		paths = append(paths, path)
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		addPath(filepath.Join(homeDir, ".claude", "settings.json"))
	}
	if workDir, err := os.Getwd(); err == nil {
		addPath(filepath.Join(workDir, ".claude", "settings.json"))
	}

	return paths
}

// syncClaudeSettings syncs configuration to existing Claude Code settings files.
// Uses surgical update mechanism to preserve JSON structure and non-ANTHROPIC fields
func (cm *Manager) syncClaudeSettings(cfg *models.APIConfig) error {
	var errs []error
	for _, claudeSettingsPath := range cm.claudeSettingsPaths() {
		if _, err := os.Stat(claudeSettingsPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, fmt.Errorf("%s: %w", claudeSettingsPath, err))
			continue
		}

		if err := cm.syncClaudeSettingsFile(claudeSettingsPath, cfg); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", claudeSettingsPath, err))
		}
	}

	return errors.Join(errs...)
}

func (cm *Manager) syncClaudeSettingsFile(claudeSettingsPath string, cfg *models.APIConfig) error {
	// Read existing settings content (raw to preserve structure and comments)
	originalContent, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		return fmt.Errorf("failed to read Claude Code settings: %w", err)
	}

	// Create synchronization options
	opts := syncpkg.SyncOptions{
		DryRun:        false,
		CreateBackup:  true, // Create backup before update to ensure data safety
		PreserveOther: true, // Preserve non-ANTHROPIC environment variables
	}

	// Perform surgical update using sjson
	updatedContent, err := syncpkg.UpdateEnvField(string(originalContent), cfg, opts)
	if err != nil {
		return fmt.Errorf("failed to update settings content: %w", err)
	}

	// Write back to file using atomic update to prevent data corruption
	if err := storage.AtomicFileUpdate(claudeSettingsPath, updatedContent, true); err != nil {
		// Attempt to restore from backup if update fails
		restoreErr := storage.NewBackupManager(storage.DefaultBackupRetention).RestoreFromLatestBackup(claudeSettingsPath)
		if restoreErr != nil {
			return fmt.Errorf("failed to write settings file and restore from backup: update error=%v, restore error=%v", err, restoreErr)
		}
		return fmt.Errorf("failed to write settings file but restored from backup: %w", err)
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
	var errs []error
	for _, claudeSettingsPath := range cm.claudeSettingsPaths() {
		if err := cm.clearClaudeSettingsFile(claudeSettingsPath); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", claudeSettingsPath, err))
		}
	}

	return errors.Join(errs...)
}

func (cm *Manager) clearClaudeSettingsFile(claudeSettingsPath string) error {
	// Check if Claude Code config file exists
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		// models.File doesn't exist, nothing to clear
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to stat Claude Code settings: %w", err)
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

	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		// Preserve malformed or non-object env fields instead of panicking.
		return nil
	}

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

	if err := storage.AtomicFileUpdate(claudeSettingsPath, string(updatedData), true); err != nil {
		return fmt.Errorf("failed to write global Claude Code settings: %v", err)
	}

	return nil
}

// Disable clears the active configuration and syncs with Claude Code
func (cm *Manager) Disable() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 1. Clear active alias
	if err := cm.updateConfigFile(func(configFile *models.File) error {
		configFile.Active = ""
		return nil
	}); err != nil {
		return err
	}

	// 3. Generate active.env (which will remove the file since Active is empty)
	if err := cm.generateActiveScript(); err != nil {
		return err
	}

	// 4. Clear Claude Code settings
	return cm.clearClaudeSettings()
}
