package config

import (
	"os"
	"testing"
)

func TestConfigManager(t *testing.T) {
	// 创建临时配置管理器用于测试
	cm := &Manager{configPath: "/tmp/test_apimgr.json"}

	// 清理测试文件
	defer os.Remove("/tmp/test_apimgr.json")

	// 测试添加配置
	config := APIConfig{
		Alias:   "test",
		APIKey:  "sk-test123",
		BaseURL: "https://api.example.com",
		Model:   "claude-3",
	}

	err := cm.Add(config)
	if err != nil {
		t.Fatalf("添加配置失败: %v", err)
	}

	// 测试获取配置
	retrievedConfig, err := cm.Get("test")
	if err != nil {
		t.Fatalf("获取配置失败: %v", err)
	}

	if retrievedConfig.Alias != "test" {
		t.Errorf("期望别名为 'test', 实际为 '%s'", retrievedConfig.Alias)
	}

	if retrievedConfig.APIKey != "sk-test123" {
		t.Errorf("期望API密钥为 'sk-test123', 实际为 '%s'", retrievedConfig.APIKey)
	}

	// 测试列出配置
	configs, err := cm.List()
	if err != nil {
		t.Fatalf("列出配置失败: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("期望1个配置，实际有 %d 个", len(configs))
	}

	// 测试删除配置
	err = cm.Remove("test")
	if err != nil {
		t.Fatalf("删除配置失败: %v", err)
	}

	// 验证配置已删除
	_, err = cm.Get("test")
	if err == nil {
		t.Error("配置应该已被删除，但还能获取到")
	}
}

func TestValidateConfig(t *testing.T) {
	cm := &Manager{configPath: "/tmp/test.json"}

	// Test empty alias
	err := cm.validateConfig(APIConfig{Alias: "", APIKey: "key"})
	if err == nil || err.Error() != "Alias cannot be empty" {
		t.Errorf("Expected 'Alias cannot be empty' error, got: %v", err)
	}

	// Test missing authentication
	err = cm.validateConfig(APIConfig{Alias: "test"})
	if err == nil || err.Error() != "API key and auth token cannot both be empty" {
		t.Errorf("Expected 'API key and auth token cannot both be empty' error, got: %v", err)
	}

	// Test auth token only (should pass)
	err = cm.validateConfig(APIConfig{Alias: "test", AuthToken: "token"})
	if err != nil {
		t.Errorf("Auth token only config should not error: %v", err)
	}

	// Test invalid URL
	err = cm.validateConfig(APIConfig{
		Alias:   "test",
		APIKey:  "sk-test",
		BaseURL: "invalid-url",
	})
	if err == nil || err.Error() != "Invalid URL format: invalid-url" {
		t.Errorf("Expected 'Invalid URL format' error, got: %v", err)
	}

	// Test valid config
	err = cm.validateConfig(APIConfig{
		Alias:   "test",
		APIKey:  "sk-test",
		BaseURL: "https://api.example.com",
	})
	if err != nil {
		t.Errorf("Valid config should not error: %v", err)
	}
}
