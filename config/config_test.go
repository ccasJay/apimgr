package config

import (
	"os"
	"testing"
)

func TestConfigManager(t *testing.T) {
	// Create temporary config manager for testing
	cm := &Manager{configPath: "/tmp/test_apimgr.json"}

	// Clean up test file
	defer os.Remove("/tmp/test_apimgr.json")

	// Test adding config
	config := APIConfig{
		Alias:   "test",
		APIKey:  "sk-test123",
		BaseURL: "https://api.example.com",
		Model:   "claude-3",
	}

	err := cm.Add(config)
	if err != nil {
		t.Fatalf("Failed to add config: %v", err)
	}

	// Test getting config
	retrievedConfig, err := cm.Get("test")
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	if retrievedConfig.Alias != "test" {
		t.Errorf("Expected alias 'test', got '%s'", retrievedConfig.Alias)
	}

	if retrievedConfig.APIKey != "sk-test123" {
		t.Errorf("Expected API key 'sk-test123', got '%s'", retrievedConfig.APIKey)
	}

	// Test listing configs
	configs, err := cm.List()
	if err != nil {
		t.Fatalf("Failed to list configs: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("Expected 1 config, got %d", len(configs))
	}

	// Test removing config
	err = cm.Remove("test")
	if err != nil {
		t.Fatalf("Failed to remove config: %v", err)
	}

	// Verify config was deleted
	_, err = cm.Get("test")
	if err == nil {
		t.Error("Config should have been deleted, but was still retrievable")
	}
}

func TestValidateConfig(t *testing.T) {
	cm := &Manager{configPath: "/tmp/test.json"}

	// Test empty alias
	err := cm.validateConfig(APIConfig{Alias: "", APIKey: "key"})
	if err == nil || err.Error() != "alias cannot be empty" {
		t.Errorf("Expected 'alias cannot be empty' error, got: %v", err)
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
	if err == nil || err.Error() != "invalid URL format: invalid-url" {
		t.Errorf("Expected 'invalid URL format' error, got: %v", err)
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
