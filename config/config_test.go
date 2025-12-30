package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestConfig creates a test config manager with a temporary directory
func setupTestConfig(t *testing.T) *Manager {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	return &Manager{configPath: configPath}
}

func TestConfigManager(t *testing.T) {
	// Use helper function for automatic cleanup
	cm := setupTestConfig(t)

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
	// Use helper function for automatic cleanup
	cm := setupTestConfig(t)

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


// TestSetActive tests setting the active configuration
func TestSetActive(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Manager) // setup function to prepare test data
		alias     string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "set valid alias as active",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test1", APIKey: "sk-test1"})
				cm.Add(APIConfig{Alias: "test2", APIKey: "sk-test2"})
			},
			alias:   "test1",
			wantErr: false,
		},
		{
			name: "set non-existent alias returns error",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test1", APIKey: "sk-test1"})
			},
			alias:     "nonexistent",
			wantErr:   true,
			errSubstr: "does not exist",
		},
		{
			name:      "set alias on empty config returns error",
			setup:     func(cm *Manager) {},
			alias:     "test",
			wantErr:   true,
			errSubstr: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := setupTestConfig(t)
			tt.setup(cm)

			err := cm.SetActive(tt.alias)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SetActive(%q) expected error, got nil", tt.alias)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("SetActive(%q) error = %v, want error containing %q", tt.alias, err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("SetActive(%q) unexpected error: %v", tt.alias, err)
				}
				// Verify active was set correctly
				activeName, _ := cm.GetActiveName()
				if activeName != tt.alias {
					t.Errorf("GetActiveName() = %q, want %q", activeName, tt.alias)
				}
			}
		})
	}
}

// TestGetActive tests getting the active configuration
func TestGetActive(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Manager)
		wantAlias string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "get active configuration",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test1", APIKey: "sk-test1"})
				cm.SetActive("test1")
			},
			wantAlias: "test1",
			wantErr:   false,
		},
		{
			name: "no active configuration returns error",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test1", APIKey: "sk-test1"})
			},
			wantErr:   true,
			errSubstr: "no active configuration set",
		},
		{
			name: "active configuration deleted returns error",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test1", APIKey: "sk-test1"})
				cm.SetActive("test1")
				cm.Remove("test1")
			},
			wantErr:   true,
			errSubstr: "no active configuration set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := setupTestConfig(t)
			tt.setup(cm)

			cfg, err := cm.GetActive()
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetActive() expected error, got nil")
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("GetActive() error = %v, want error containing %q", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("GetActive() unexpected error: %v", err)
				}
				if cfg.Alias != tt.wantAlias {
					t.Errorf("GetActive().Alias = %q, want %q", cfg.Alias, tt.wantAlias)
				}
			}
		})
	}
}

// TestGenerateActiveScript tests generating the activation script
func TestGenerateActiveScript(t *testing.T) {
	t.Run("generates script with correct environment variables", func(t *testing.T) {
		cm := setupTestConfig(t)
		cm.Add(APIConfig{
			Alias:   "test",
			APIKey:  "sk-test123",
			BaseURL: "https://api.example.com",
			Model:   "claude-3",
		})
		cm.SetActive("test")

		err := cm.GenerateActiveScript()
		if err != nil {
			t.Fatalf("GenerateActiveScript() error: %v", err)
		}

		// Check active.env file was created
		activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
		data, err := os.ReadFile(activeEnvPath)
		if err != nil {
			t.Fatalf("Failed to read active.env: %v", err)
		}

		content := string(data)
		// Verify environment variables are present
		if !contains(content, "ANTHROPIC_API_KEY") {
			t.Error("active.env should contain ANTHROPIC_API_KEY")
		}
		if !contains(content, "sk-test123") {
			t.Error("active.env should contain the API key value")
		}
		if !contains(content, "ANTHROPIC_BASE_URL") {
			t.Error("active.env should contain ANTHROPIC_BASE_URL")
		}
		if !contains(content, "APIMGR_ACTIVE") {
			t.Error("active.env should contain APIMGR_ACTIVE")
		}
	})

	t.Run("cleans up active.env when no active config", func(t *testing.T) {
		cm := setupTestConfig(t)

		// Create a dummy active.env file
		activeEnvPath := filepath.Join(filepath.Dir(cm.configPath), "active.env")
		os.WriteFile(activeEnvPath, []byte("dummy"), 0600)

		err := cm.GenerateActiveScript()
		if err != nil {
			t.Fatalf("GenerateActiveScript() error: %v", err)
		}

		// Check active.env file was removed
		if _, err := os.Stat(activeEnvPath); !os.IsNotExist(err) {
			t.Error("active.env should be removed when no active config")
		}
	})
}

// TestUpdatePartial tests partial configuration updates
func TestUpdatePartial(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Manager)
		alias     string
		updates   map[string]string
		wantErr   bool
		errSubstr string
		verify    func(*testing.T, *Manager)
	}{
		{
			name: "update api_key field",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test", APIKey: "sk-old"})
			},
			alias:   "test",
			updates: map[string]string{"api_key": "sk-new"},
			wantErr: false,
			verify: func(t *testing.T, cm *Manager) {
				cfg, _ := cm.Get("test")
				if cfg.APIKey != "sk-new" {
					t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-new")
				}
			},
		},
		{
			name: "update multiple fields",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test", APIKey: "sk-test"})
			},
			alias: "test",
			updates: map[string]string{
				"base_url": "https://new.api.com",
				"model":    "claude-4",
			},
			wantErr: false,
			verify: func(t *testing.T, cm *Manager) {
				cfg, _ := cm.Get("test")
				if cfg.BaseURL != "https://new.api.com" {
					t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://new.api.com")
				}
				if cfg.Model != "claude-4" {
					t.Errorf("Model = %q, want %q", cfg.Model, "claude-4")
				}
			},
		},
		{
			name: "update non-existent config returns error",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test", APIKey: "sk-test"})
			},
			alias:     "nonexistent",
			updates:   map[string]string{"api_key": "sk-new"},
			wantErr:   true,
			errSubstr: "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := setupTestConfig(t)
			tt.setup(cm)

			err := cm.UpdatePartial(tt.alias, tt.updates)
			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdatePartial(%q) expected error, got nil", tt.alias)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("UpdatePartial(%q) error = %v, want error containing %q", tt.alias, err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("UpdatePartial(%q) unexpected error: %v", tt.alias, err)
				}
				if tt.verify != nil {
					tt.verify(t, cm)
				}
			}
		})
	}
}

// TestRenameAlias tests renaming configuration aliases
func TestRenameAlias(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Manager)
		oldAlias  string
		newAlias  string
		wantErr   bool
		errSubstr string
		verify    func(*testing.T, *Manager)
	}{
		{
			name: "rename alias successfully",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "old", APIKey: "sk-test"})
			},
			oldAlias: "old",
			newAlias: "new",
			wantErr:  false,
			verify: func(t *testing.T, cm *Manager) {
				// Old alias should not exist
				_, err := cm.Get("old")
				if err == nil {
					t.Error("old alias should not exist after rename")
				}
				// New alias should exist
				cfg, err := cm.Get("new")
				if err != nil {
					t.Errorf("new alias should exist: %v", err)
				}
				if cfg.APIKey != "sk-test" {
					t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test")
				}
			},
		},
		{
			name: "rename to existing alias returns error",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "config1", APIKey: "sk-test1"})
				cm.Add(APIConfig{Alias: "config2", APIKey: "sk-test2"})
			},
			oldAlias:  "config1",
			newAlias:  "config2",
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "rename non-existent alias returns error",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "test", APIKey: "sk-test"})
			},
			oldAlias:  "nonexistent",
			newAlias:  "new",
			wantErr:   true,
			errSubstr: "does not exist",
		},
		{
			name: "rename active config updates active field",
			setup: func(cm *Manager) {
				cm.Add(APIConfig{Alias: "active", APIKey: "sk-test"})
				cm.SetActive("active")
			},
			oldAlias: "active",
			newAlias: "renamed",
			wantErr:  false,
			verify: func(t *testing.T, cm *Manager) {
				activeName, _ := cm.GetActiveName()
				if activeName != "renamed" {
					t.Errorf("active = %q, want %q", activeName, "renamed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := setupTestConfig(t)
			tt.setup(cm)

			err := cm.RenameAlias(tt.oldAlias, tt.newAlias)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RenameAlias(%q, %q) expected error, got nil", tt.oldAlias, tt.newAlias)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("RenameAlias(%q, %q) error = %v, want error containing %q", tt.oldAlias, tt.newAlias, err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("RenameAlias(%q, %q) unexpected error: %v", tt.oldAlias, tt.newAlias, err)
				}
				if tt.verify != nil {
					tt.verify(t, cm)
				}
			}
		})
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
