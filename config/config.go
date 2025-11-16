package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"apimgr/internal/providers"
	"apimgr/internal/utils"
)

// APIConfig represents a single API configuration
type APIConfig struct {
	Alias     string `json:"alias"`
	Provider  string `json:"provider"` // API提供商类型
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
	mu         sync.Mutex // 互斥锁，保护并发访问
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

	// Write to new location with locking
	file, err := os.OpenFile(newPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("打开新配置文件失败: %v", err)
	}

	// Lock the new config file exclusively
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return fmt.Errorf("锁定新配置文件失败: %v", err)
	}

	// Write data while holding the lock
	_, err = file.Write(data)
	if err != nil {
		file.Close()
		return fmt.Errorf("写入新配置文件失败: %v", err)
	}

	// Ensure data is flushed
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("同步新配置文件失败: %v", err)
	}

	// Unlock and close
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		file.Close()
		return fmt.Errorf("解锁新配置文件失败: %v", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭新配置文件失败: %v", err)
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
	// Open the file with read lock
	file, err := os.OpenFile(cm.configPath, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return &ConfigFile{Configs: []APIConfig{}}, nil
		}
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	// Lock the file for shared read access (LOCK_SH)
	if err := cm.lockFileShared(file); err != nil {
		return nil, fmt.Errorf("锁定配置文件失败: %v", err)
	}
	defer cm.unlockFile(file)

	// Read from the locked file descriptor instead of using os.ReadFile
	data, err := io.ReadAll(file)
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

	// Open the file with write access (create if not exists)
	file, err := os.OpenFile(cm.configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	// Lock the file for exclusive write access
	if err := cm.lockFile(file); err != nil {
		return fmt.Errorf("锁定配置文件失败: %v", err)
	}
	defer cm.unlockFile(file)

	// Write the file while holding the lock
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	// Ensure data is flushed to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("同步配置文件失败: %v", err)
	}

	return nil
}

// lockFile locks the config file with exclusive lock (for write operations)
func (cm *ConfigManager) lockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

// lockFileShared locks the config file with shared lock (for read operations)
func (cm *ConfigManager) lockFileShared(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_SH)
}

// unlockFile unlocks the config file
func (cm *ConfigManager) unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}

// Load loads all configurations from the config file
func (cm *ConfigManager) Load() ([]APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}
	return configFile.Configs, nil
}

// Save saves configurations to the config file
func (cm *ConfigManager) Save(configs []APIConfig) error {
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
func (cm *ConfigManager) Add(config APIConfig) error {
	// 设置默认提供商
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
func (cm *ConfigManager) Remove(alias string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configs, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	for i, config := range configs.Configs {
		if config.Alias == alias {
			configs.Configs = append(configs.Configs[:i], configs.Configs[i+1:]...)
			// 如果移除的是活动配置，则清空活动配置
			if configs.Active == alias {
				configs.Active = ""
			}
			return cm.saveConfigFile(configs)
		}
	}

	return fmt.Errorf("配置 '%s' 不存在", alias)
}

// Get returns a configuration by alias
func (cm *ConfigManager) Get(alias string) (*APIConfig, error) {
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

	return nil, fmt.Errorf("配置 '%s' 不存在", alias)
}

// List returns all configurations
func (cm *ConfigManager) List() ([]APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configs, err := cm.loadConfigFile()
	if err != nil {
		return nil, err
	}
	return configs.Configs, nil
}

// SetActive sets the active configuration
func (cm *ConfigManager) SetActive(alias string) error {
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
		return fmt.Errorf("配置 '%s' 不存在", alias)
	}

	configFile.Active = alias
	return cm.saveConfigFile(configFile)
}

// GetActive returns the active configuration
func (cm *ConfigManager) GetActive() (*APIConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

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
	cm.mu.Lock()
	defer cm.mu.Unlock()

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

	// 默认提供商为anthropic
	providerName := config.Provider
	if providerName == "" {
		providerName = "anthropic"
	}

	// 至少需要一种认证方式
	if config.APIKey == "" && config.AuthToken == "" {
		return fmt.Errorf("API密钥和认证令牌不能同时为空")
	}

	// 验证提供商
	provider, err := providers.Get(providerName)
	if err != nil {
		return fmt.Errorf("未知的API提供商: %s", providerName)
	}

	// 提供商特定验证
	if err := provider.ValidateConfig(config.BaseURL, config.APIKey, config.AuthToken); err != nil {
		return err
	}

	// URL格式验证
	if config.BaseURL != "" {
		if !utils.ValidateURL(config.BaseURL) {
			return fmt.Errorf("无效的URL格式: %s", config.BaseURL)
		}
	}

	return nil
}


// UpdatePartial updates only the specified fields of a configuration
func (cm *ConfigManager) UpdatePartial(alias string, updates map[string]string) error {
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

	return fmt.Errorf("配置 '%s' 不存在", alias)
}

// RenameAlias renames a configuration alias
func (cm *ConfigManager) RenameAlias(oldAlias, newAlias string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		return err
	}

	// Check if new alias already exists
	for _, cfg := range configFile.Configs {
		if cfg.Alias == newAlias {
			return fmt.Errorf("配置 '%s' 已存在", newAlias)
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
		return fmt.Errorf("配置 '%s' 不存在", oldAlias)
	}

	// Update active config if needed
	if configFile.Active == oldAlias {
		configFile.Active = newAlias
	}

	return cm.saveConfigFile(configFile)
}

// GenerateActiveScript 生成活动配置的激活脚本
func (cm *ConfigManager) GenerateActiveScript() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	configFile, err := cm.loadConfigFile()
	if err != nil {
		// 没有活动配置，清理 active.env 文件
		activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
		os.Remove(activeEnvPath)
		return nil
	}

	var active *APIConfig
	if configFile.Active != "" {
		for _, config := range configFile.Configs {
			if config.Alias == configFile.Active {
				active = &config
				break
			}
		}
	}

	if active == nil {
		// 没有活动配置，清理 active.env 文件
		activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
		os.Remove(activeEnvPath)
		return nil
	}

	// 生成激活脚本内容
	envScript := generateEnvScript(active)

	// 写入文件
	activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
	if err := os.WriteFile(activeEnvPath, []byte(envScript), 0644); err != nil {
		return err
	}

	// 同步到全局 Claude Code 设置（可选功能，不影响主流程）
	if syncErr := cm.syncClaudeSettings(active); syncErr != nil {
		fmt.Printf("⚠️  同步全局 Claude Code 设置失败: %v\n", syncErr)
	}

	// 同步到项目级配置文件（如果存在 .claude 目录）
	if syncErr := cm.syncProjectClaudeConfig(active); syncErr != nil {
		// 项目级同步失败不报错，因为不是所有项目都有 .claude 目录
	}

	return nil
}

// generateEnvScript 生成环境变量脚本内容
func generateEnvScript(cfg *APIConfig) string {
	var buf strings.Builder

	// 添加注释
	buf.WriteString("# 自动生成的活动配置 - 每次配置变更时更新\n")
	buf.WriteString("# 请不要手动编辑此文件\n\n")

	// 清除旧环境变量
	buf.WriteString("# 清除之前设置的环境变量\n")
	buf.WriteString("unset ANTHROPIC_API_KEY\n")
	buf.WriteString("unset ANTHROPIC_AUTH_TOKEN\n")
	buf.WriteString("unset ANTHROPIC_BASE_URL\n")
	buf.WriteString("unset ANTHROPIC_MODEL\n")
	buf.WriteString("unset APIMGR_ACTIVE\n\n")

	// 设置新环境变量
	buf.WriteString("# 设置新的环境变量\n")
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

// syncClaudeSettings 同步配置到全局 Claude Code 设置文件
func (cm *ConfigManager) syncClaudeSettings(cfg *APIConfig) error {
	claudeSettingsPath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")

	// 检查 Claude Code 配置文件是否存在
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		// 文件不存在，跳过同步
		return nil
	}

	// 读取现有设置
	data, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		return fmt.Errorf("读取全局 Claude Code 设置失败: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("解析全局 Claude Code 设置失败: %v", err)
	}

	// 确保 env 字段存在
	if settings["env"] == nil {
		settings["env"] = make(map[string]interface{})
	}

	env := settings["env"].(map[string]interface{})

	// 更新环境变量
	// 清除旧的 ANTHROPIC 相关变量
	delete(env, "ANTHROPIC_API_KEY")
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, "ANTHROPIC_BASE_URL")

	// 设置新的环境变量
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

	// 写回文件
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化全局 Claude Code 设置失败: %v", err)
	}

	if err := os.WriteFile(claudeSettingsPath, updatedData, 0644); err != nil {
		return fmt.Errorf("写入全局 Claude Code 设置失败: %v", err)
	}

	return nil
}

// syncProjectClaudeConfig 同步配置到项目级 .claude/settings.json
func (cm *ConfigManager) syncProjectClaudeConfig(cfg *APIConfig) error {
	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// 项目级配置文件路径
	projectClaudePath := filepath.Join(workDir, ".claude", "settings.json")

	// 检查项目是否有 .claude 目录
	if _, err := os.Stat(projectClaudePath); os.IsNotExist(err) {
		// 没有项目级配置，跳过
		return nil
	}

	// 读取项目级配置
	data, err := os.ReadFile(projectClaudePath)
	if err != nil {
		return fmt.Errorf("读取项目级 Claude Code 设置失败: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("解析项目级 Claude Code 设置失败: %v", err)
	}

	// 更新 env 字段
	if settings["env"] == nil {
		settings["env"] = make(map[string]interface{})
	}

	env := settings["env"].(map[string]interface{})

	// 清除旧的 ANTHROPIC 相关变量
	delete(env, "ANTHROPIC_API_KEY")
	delete(env, "ANTHROPIC_AUTH_TOKEN")
	delete(env, "ANTHROPIC_BASE_URL")

	// 设置新的环境变量
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

	// 写回文件
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化项目级 Claude Code 设置失败: %v", err)
	}

	if err := os.WriteFile(projectClaudePath, updatedData, 0644); err != nil {
		return fmt.Errorf("写入项目级 Claude Code 设置失败: %v", err)
	}

	fmt.Printf("✅ 项目级 Claude Code 配置已更新: %s\n", projectClaudePath)
	return nil
}
