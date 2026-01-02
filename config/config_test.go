package config

import (
	"os"
	"path/filepath"
	"testing"

	"apimgr/config/models"
	"apimgr/config/validation"
)

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, content string) (string, func()) {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "config_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpfile.Name(), func() {
		os.Remove(tmpfile.Name())
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  models.APIConfig
		wantErr bool
	}{
		{
			name: "Valid config with API Key",
			config: models.APIConfig{
				Alias:    "test",
				Provider: "anthropic",
				APIKey:   "sk-test",
			},
			wantErr: false,
		},
		{
			name: "Valid config with Auth Token",
			config: models.APIConfig{
				Alias:     "test",
				Provider:  "anthropic",
				AuthToken: "token-test",
			},
			wantErr: false,
		},
		{
			name: "Missing Alias",
			config: models.APIConfig{
				Provider: "anthropic",
				APIKey:   "sk-test",
			},
			wantErr: true,
		},
		{
			name: "Missing Auth",
			config: models.APIConfig{
				Alias:    "test",
				Provider: "anthropic",
			},
			wantErr: true,
		},
		{
			name: "Invalid Provider",
			config: models.APIConfig{
				Alias:    "test",
				Provider: "invalid",
				APIKey:   "sk-test",
			},
			wantErr: true,
		},
	}

	validator := validation.NewValidator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &Manager{}
			_ = cm // Suppress unused variable warning
			if err := validator.ValidateConfig(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}


// setupTestConfig creates a test config manager with a temporary directory
func setupTestConfig(t *testing.T) *Manager {
	t.Helper()
	t.Setenv("APIMGR_ACTIVE", "") // Ensure clean environment
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	return &Manager{configPath: configPath}
}

// TestGetActiveEnvOverride tests that APIMGR_ACTIVE environment variable overrides the active configuration
func TestGetActiveEnvOverride(t *testing.T) {
	cm := setupTestConfig(t)
	// Add two configs
	cm.Add(models.APIConfig{Alias: "global", APIKey: "sk-global"})
	cm.Add(models.APIConfig{Alias: "local", APIKey: "sk-local"})

	// Set global active to "global"
	cm.SetActive("global")

	// Verify global is active initially
	active, err := cm.GetActiveName()
	if err != nil {
		t.Fatalf("GetActiveName() unexpected error: %v", err)
	}
	if active != "global" {
		t.Errorf("Initial GetActiveName() = %q, want %q", active, "global")
	}

	// Set environment variable override
	t.Setenv("APIMGR_ACTIVE", "local")

	// Verify local is now active via GetActiveName
	active, err = cm.GetActiveName()
	if err != nil {
		t.Fatalf("GetActiveName() unexpected error: %v", err)
	}
	if active != "local" {
		t.Errorf("Overridden GetActiveName() = %q, want %q", active, "local")
	}

	// Verify local is now active via GetActive
	cfg, err := cm.GetActive()
	if err != nil {
		t.Fatalf("GetActive() unexpected error: %v", err)
	}
	if cfg.Alias != "local" {
		t.Errorf("Overridden GetActive().Alias = %q, want %q", cfg.Alias, "local")
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
				cm.Add(models.APIConfig{Alias: "test1", APIKey: "sk-test1"})
				cm.Add(models.APIConfig{Alias: "test2", APIKey: "sk-test2"})
			},
			alias:   "test1",
			wantErr: false,
		},
		{
			name: "set non-existent alias returns error",
			setup: func(cm *Manager) {
				cm.Add(models.APIConfig{Alias: "test1", APIKey: "sk-test1"})
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
				cm.Add(models.APIConfig{Alias: "test1", APIKey: "sk-test1"})
				cm.SetActive("test1")
			},
			wantAlias: "test1",
			wantErr:   false,
		},
		{
			name: "no active configuration returns error",
			setup: func(cm *Manager) {
				cm.Add(models.APIConfig{Alias: "test1", APIKey: "sk-test1"})
			},
			wantErr:   true,
			errSubstr: "no active configuration set",
		},
		{
			name: "active configuration deleted returns error",
			setup: func(cm *Manager) {
				cm.Add(models.APIConfig{Alias: "test1", APIKey: "sk-test1"})
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
					return
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
		cm.Add(models.APIConfig{
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
				cm.Add(models.APIConfig{Alias: "test", APIKey: "sk-old"})
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
				cm.Add(models.APIConfig{Alias: "test", APIKey: "sk-test"})
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
				cm.Add(models.APIConfig{Alias: "test", APIKey: "sk-test"})
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
				cm.Add(models.APIConfig{Alias: "old", APIKey: "sk-test"})
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
				cm.Add(models.APIConfig{Alias: "config1", APIKey: "sk-test1"})
				cm.Add(models.APIConfig{Alias: "config2", APIKey: "sk-test2"})
			},
			oldAlias:  "config1",
			newAlias:  "config2",
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "rename non-existent alias returns error",
			setup: func(cm *Manager) {
				cm.Add(models.APIConfig{Alias: "test", APIKey: "sk-test"})
			},
			oldAlias:  "nonexistent",
			newAlias:  "new",
			wantErr:   true,
			errSubstr: "does not exist",
		},
		{
			name: "rename active config updates active field",
			setup: func(cm *Manager) {
				cm.Add(models.APIConfig{Alias: "active", APIKey: "sk-test"})
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
