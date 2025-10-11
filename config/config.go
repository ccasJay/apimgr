package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
)

// APIConfig represents a single API configuration
type APIConfig struct {
	Alias     string `json:"alias"`
	APIKey    string `json:"api_key"`
	BaseURL   string `json:"base_url"`
	Model     string `json:"model"`
}

// ConfigManager manages API configurations
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a new ConfigManager
func NewConfigManager() *ConfigManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("无法获取用户主目录: %v", err))
	}

	var configPath string
	if runtime.GOOS == "windows" {
		configPath = filepath.Join(homeDir, ".apimgr.json")
	} else {
		configPath = filepath.Join(homeDir, ".apimgr.json")
	}

	return &ConfigManager{
		configPath: configPath,
	}
}

// GetConfigPath returns the path to the config file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// Load loads all configurations from the config file
func (cm *ConfigManager) Load() ([]APIConfig, error) {
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return []APIConfig{}, nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var configs []APIConfig
	if len(data) == 0 {
		return configs, nil
	}

	err = json.Unmarshal(data, &configs)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return configs, nil
}

// Save saves configurations to the config file
func (cm *ConfigManager) Save(configs []APIConfig) error {
	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	err = os.WriteFile(cm.configPath, data, 0600)
	if err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
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

// validateConfig validates a configuration
func (cm *ConfigManager) validateConfig(config APIConfig) error {
	if config.Alias == "" {
		return fmt.Errorf("别名不能为空")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API密钥不能为空")
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