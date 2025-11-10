package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"syscall"
)

// APIConfig represents a single API configuration
type APIConfig struct {
	Alias     string `json:"alias"`
	APIKey    string `json:"api_key"`
	AuthToken string `json:"auth_token"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
}

// ConfigFile represents the structure of the config file
type ConfigFile struct {
	Active  string      `json:"active"`
	Configs []APIConfig `json:"configs"`
}

// ConfigManager manages API configurations
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a new ConfigManager with unified config path
func NewConfigManager() *ConfigManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("无法获取用户主目录: %v", err))
	}

	// Always use XDG config location (new standard)
	xdgConfigPath := filepath.Join(homeDir, ".config", "apimgr", "config.json")
	oldConfigPath := filepath.Join(homeDir, ".apimgr.json")

	configPath := xdgConfigPath

	// Ensure XDG directory exists
	configDir := filepath.Dir(xdgConfigPath)
	os.MkdirAll(configDir, 0755)

	// Migrate from old config if it exists and new config doesn't
	if shouldMigrateConfig(oldConfigPath, xdgConfigPath) {
		if err := migrateConfig(oldConfigPath, xdgConfigPath); err != nil {
			fmt.Printf("⚠️  配置迁移失败: %v\n", err)
			// Continue with new config path anyway
		} else {
			fmt.Println("✅ 从旧配置位置迁移配置成功")
		}
	}

	return &ConfigManager{
		configPath: configPath,
	}
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
		return fmt.Errorf("读取旧配置文件失败: %v", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("旧配置文件为空")
	}

	// Validate that it's a valid config
	var tempConfig ConfigFile
	if err := json.Unmarshal(data, &tempConfig); err != nil {
		// Try old format (array of configs)
		var tempConfigs []APIConfig
		if err2 := json.Unmarshal(data, &tempConfigs); err2 != nil {
			return fmt.Errorf("旧配置文件格式无效: %v", err)
		}
	}

	// Write to new location
	if err := os.WriteFile(newPath, data, 0600); err != nil {
		return fmt.Errorf("写入新配置文件失败: %v", err)
	}

	// Backup old config
	backupPath := oldPath + ".backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		// Don't fail migration if backup fails
		fmt.Printf("⚠️  无法创建旧配置备份: %v\n", err)
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetConfigPath returns the path to the config file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// loadConfigFile loads the config file with locking
func (cm *ConfigManager) loadConfigFile() (*ConfigFile, error) {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return &ConfigFile{Configs: []APIConfig{}}, nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	if len(data) == 0 {
		return &ConfigFile{Configs: []APIConfig{}}, nil
	}

	var configFile ConfigFile
	err = json.Unmarshal(data, &configFile)
	if err != nil {
		// Try to parse as old format (array of configs)
		var configs []APIConfig
		if err2 := json.Unmarshal(data, &configs); err2 == nil {
			return &ConfigFile{Configs: configs}, nil
		}
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &configFile, nil
}

// saveConfigFile saves the config file with locking
func (cm *ConfigManager) saveConfigFile(configFile *ConfigFile) error {
	data, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	err = os.WriteFile(cm.configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
}

// lockFile locks the config file
func (cm *ConfigManager) lockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

// unlockFile unlocks the config file
func (cm *ConfigManager) unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

// Load loads all configurations from the config file
func (cm *ConfigManager) Load() ([]APIConfig, error) {
	configFile, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}
	return configFile.Configs, nil
}

// Save saves configurations to the config file
func (cm *ConfigManager) Save(configs []APIConfig) error {
	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}
	configFile.Configs = configs
	return cm.saveConfigFile(configFile)
}

// Add adds a new configuration
func (cm *ConfigManager) Add(config APIConfig) error {
	if err := cm.validateConfig(config); err != nil {
		return err
	}

	configs, err := cm.Load()
	if err != nil {
		return err
	}

	// Check if alias already exists
	for i, existingConfig := range configs {
		if existingConfig.Alias == config.Alias {
			configs[i] = config
			return cm.Save(configs)
		}
	}

	configs = append(configs, config)
	return cm.Save(configs)
}

// Remove removes a configuration by alias
func (cm *ConfigManager) Remove(alias string) error {
	configs, err := cm.Load()
	if err != nil {
		return err
	}

	for i, config := range configs {
		if config.Alias == alias {
			configs = append(configs[:i], configs[i+1:]...)
			return cm.Save(configs)
		}
	}

	return fmt.Errorf("配置 '%s' 不存在", alias)
}

// Get returns a configuration by alias
func (cm *ConfigManager) Get(alias string) (*APIConfig, error) {
	configs, err := cm.Load()
	if err != nil {
		return nil, err
	}

	for _, config := range configs {
		if config.Alias == alias {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("配置 '%s' 不存在", alias)
}

// List returns all configurations
func (cm *ConfigManager) List() ([]APIConfig, error) {
	return cm.Load()
}

// SetActive sets the active configuration
func (cm *ConfigManager) SetActive(alias string) error {
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
		return fmt.Errorf("配置 '%s' 不存在", alias)
	}

	configFile.Active = alias
	return cm.saveConfigFile(configFile)
}

// GetActive returns the active configuration
func (cm *ConfigManager) GetActive() (*APIConfig, error) {
	configFile, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}

	if configFile.Active == "" {
		return nil, fmt.Errorf("未设置活动配置")
	}

	for _, config := range configFile.Configs {
		if config.Alias == configFile.Active {
			return &config, nil
		}
	}

	return nil, fmt.Errorf("活动配置 '%s' 不存在", configFile.Active)
}

// GetActiveName returns the active configuration name
func (cm *ConfigManager) GetActiveName() (string, error) {
	configFile, err := cm.loadConfigFile()
	if err != nil {
		return "", err
	}
	return configFile.Active, nil
}

// validateConfig validates a configuration
func (cm *ConfigManager) validateConfig(config APIConfig) error {
	if config.Alias == "" {
		return fmt.Errorf("别名不能为空")
	}

	// 至少需要一种认证方式
	if config.APIKey == "" && config.AuthToken == "" {
		return fmt.Errorf("API密钥和认证令牌不能同时为空")
	}

	if config.BaseURL != "" {
		if !isValidURL(config.BaseURL) {
			return fmt.Errorf("无效的URL格式: %s", config.BaseURL)
		}
	}

	return nil
}

// isValidURL checks if a string is a valid URL
func isValidURL(u string) bool {
	_, err := url.ParseRequestURI(u)
	return err == nil
}

// UpdatePartial updates only the specified fields of a configuration
func (cm *ConfigManager) UpdatePartial(alias string, updates map[string]string) error {
	configs, err := cm.Load()
	if err != nil {
		return err
	}

	for i, config := range configs {
		if config.Alias == alias {
			// Update only the fields that are provided
			if apiKey, ok := updates["api_key"]; ok {
				configs[i].APIKey = apiKey
			}
			if authToken, ok := updates["auth_token"]; ok {
				configs[i].AuthToken = authToken
			}
			if baseURL, ok := updates["base_url"]; ok {
				configs[i].BaseURL = baseURL
			}
			if model, ok := updates["model"]; ok {
				configs[i].Model = model
			}

			// Validate the updated config
			if err := cm.validateConfig(configs[i]); err != nil {
				return err
			}

			return cm.Save(configs)
		}
	}

	return fmt.Errorf("配置 '%s' 不存在", alias)
}

// RenameAlias renames a configuration alias
func (cm *ConfigManager) RenameAlias(oldAlias, newAlias string) error {
	configs, err := cm.Load()
	if err != nil {
		return err
	}

	// Check if new alias already exists
	for _, cfg := range configs {
		if cfg.Alias == newAlias {
			return fmt.Errorf("配置 '%s' 已存在", newAlias)
		}
	}

	// Find and rename
	for i, cfg := range configs {
		if cfg.Alias == oldAlias {
			configs[i].Alias = newAlias

			// Update active config if needed
			configFile, err := cm.loadConfigFile()
			if err == nil && configFile.Active == oldAlias {
				configFile.Active = newAlias
				cm.saveConfigFile(configFile)
			}

			return cm.Save(configs)
		}
	}

	return fmt.Errorf("配置 '%s' 不存在", oldAlias)
}