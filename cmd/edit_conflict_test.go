package cmd

import (
	"path/filepath"
	"testing"

	"github.com/ccasJay/apimgr/config"
	"github.com/ccasJay/apimgr/config/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdatePartial_ApiKeyClearsAuthToken 测试设置 API Key 时自动清空 Auth Token
func TestUpdatePartial_ApiKeyClearsAuthToken(t *testing.T) {
	// 初始使用 AuthToken 认证
	tmpDir := t.TempDir()
	cfg := &models.File{
		Configs: []models.APIConfig{
			{
				Alias:     "test",
				AuthToken: "old-token",
				BaseURL:   "https://api.example.com",
				Model:     "model-v1",
			},
		},
		Active: "test",
	}
	configManager := setupTestConfigManager(t, tmpDir, cfg)

	// 设置 API Key — 应自动清空 AuthToken
	err := configManager.UpdatePartial("test", map[string]string{"api_key": "sk-new-key"})
	require.NoError(t, err)

	updated, err := configManager.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "sk-new-key", updated.APIKey)
	assert.Empty(t, updated.AuthToken, "Auth token should be cleared when API key is set")
}

// TestUpdatePartial_AuthTokenClearsApiKey 测试设置 Auth Token 时自动清空 API Key
func TestUpdatePartial_AuthTokenClearsApiKey(t *testing.T) {
	// 初始使用 APIKey 认证
	tmpDir := t.TempDir()
	cfg := &models.File{
		Configs: []models.APIConfig{
			{
				Alias:   "test",
				APIKey:  "sk-old-key",
				BaseURL: "https://api.example.com",
				Model:   "model-v1",
			},
		},
		Active: "test",
	}
	configManager := setupTestConfigManager(t, tmpDir, cfg)

	// 设置 Auth Token — 应自动清空 API Key
	err := configManager.UpdatePartial("test", map[string]string{"auth_token": "new-token"})
	require.NoError(t, err)

	updated, err := configManager.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "new-token", updated.AuthToken)
	assert.Empty(t, updated.APIKey, "API key should be cleared when auth token is set")
}

// TestUpdatePartial_BothFieldsProvided 测试同时提供 api_key 和 auth_token（后者应该覆盖）
func TestUpdatePartial_BothFieldsProvided(t *testing.T) {
	// 初始使用 APIKey 认证
	tmpDir := t.TempDir()
	cfg := &models.File{
		Configs: []models.APIConfig{
			{
				Alias:  "test",
				APIKey: "sk-old",
			},
		},
		Active: "test",
	}
	configManager := setupTestConfigManager(t, tmpDir, cfg)

	// 同时提供两个字段（模拟命令行同时指定 --sk 和 --ak）
	// 按照代码执行顺序：先处理 api_key，将 auth_token 清空；再处理 auth_token，将 api_key 清空
	// 最终 auth_token 会保留，api_key 被清空
	err := configManager.UpdatePartial("test", map[string]string{
		"api_key":    "sk-both",
		"auth_token": "token-both",
	})
	require.NoError(t, err)

	updated, err := configManager.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "token-both", updated.AuthToken)
	assert.Empty(t, updated.APIKey, "When both provided, auth_token takes precedence")
}

// TestUpdatePartial_ClearOnlyAuthFails 测试清空唯一的认证方式应返回错误
func TestUpdatePartial_ClearOnlyAuthFails(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &models.File{
		Configs: []models.APIConfig{
			{
				Alias:  "test",
				APIKey: "sk-keep",
			},
		},
		Active: "test",
	}
	configManager := setupTestConfigManager(t, tmpDir, cfg)

	// 清空唯一的 API Key — 验证器应拒绝（两种认证方式都为空）
	err := configManager.UpdatePartial("test", map[string]string{"api_key": ""})
	assert.Error(t, err, "Should fail when clearing the only auth method")
	assert.Contains(t, err.Error(), "cannot both be empty")
}

// TestUpdatePartial_NoChange 测试未提供字段时不影响现有值
func TestUpdatePartial_NoChange(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &models.File{
		Configs: []models.APIConfig{
			{
				Alias:   "test",
				APIKey:  "sk-keep",
				BaseURL: "https://api.example.com",
			},
		},
		Active: "test",
	}
	configManager := setupTestConfigManager(t, tmpDir, cfg)

	err := configManager.UpdatePartial("test", map[string]string{"base_url": "https://new.example.com"})
	require.NoError(t, err)

	updated, err := configManager.Get("test")
	require.NoError(t, err)
	assert.Equal(t, "sk-keep", updated.APIKey)
	assert.Empty(t, updated.AuthToken)
	assert.Equal(t, "https://new.example.com", updated.BaseURL)
}

// setupTestConfigManager 创建测试用的 ConfigManager，使用临时目录隔离测试
func setupTestConfigManager(t *testing.T, tmpDir string, initialConfig *models.File) *config.Manager {
	t.Helper()
	t.Setenv("APIMGR_ACTIVE", "") // 确保环境干净
	configPath := filepath.Join(tmpDir, "config.json")
	cm := config.NewConfigManagerWithPath(configPath)

	// 写入初始配置
	if initialConfig != nil {
		for _, cfg := range initialConfig.Configs {
			if err := cm.Add(cfg); err != nil {
				t.Fatalf("Failed to add initial config %q: %v", cfg.Alias, err)
			}
		}
		if initialConfig.Active != "" {
			if err := cm.SetActive(initialConfig.Active); err != nil {
				t.Fatalf("Failed to set active config %q: %v", initialConfig.Active, err)
			}
		}
	}
	return cm
}
