package cmd

import (
	"os"
	"testing"

	"github.com/ccasJay/apimgr/config"
	"github.com/ccasJay/apimgr/config/models"
	"github.com/ccasJay/apimgr/internal/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskCredential(t *testing.T) {
	tests := []struct {
		name       string
		credential string
		expected   string
	}{
		{
			name:       "long credential",
			credential: "sk-ant-api03-very-long-key-here-123456789",
			expected:   "sk-a...6789",
		},
		{
			name:       "exact 8 characters",
			credential: "12345678",
			expected:   "********",
		},
		{
			name:       "short credential",
			credential: "short",
			expected:   "********",
		},
		{
			name:       "empty credential",
			credential: "",
			expected:   "********",
		},
		{
			name:       "just over 8 characters",
			credential: "123456789",
			expected:   "1234...6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskCredential(tt.credential)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindConflicts(t *testing.T) {
	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			{Alias: "config-1"},
			{Alias: "config-2"},
			{Alias: "config-3"},
		},
	}

	existingMap := map[string]bool{
		"config-1":      true,
		"config-3":      true,
		"existing-only": true,
	}

	conflicts := findConflicts(export, existingMap)

	assert.Len(t, conflicts, 2)
	assert.Contains(t, conflicts, "config-1")
	assert.Contains(t, conflicts, "config-3")
	assert.NotContains(t, conflicts, "config-2")
	assert.NotContains(t, conflicts, "existing-only")
}

func TestFindConflictsNoConflicts(t *testing.T) {
	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			{Alias: "config-1"},
			{Alias: "config-2"},
		},
	}

	existingMap := map[string]bool{
		"other-config": true,
	}

	conflicts := findConflicts(export, existingMap)

	assert.Empty(t, conflicts)
}

func TestFindConflictsEmptyExport(t *testing.T) {
	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{},
	}

	existingMap := map[string]bool{
		"config-1": true,
	}

	conflicts := findConflicts(export, existingMap)

	assert.Empty(t, conflicts)
}

func TestImportConfigsAllNew(t *testing.T) {
	cm, cleanup := mockConfigManager(t)
	defer cleanup()

	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			crypto.NewExportConfig(
				"new-config-1",
				"anthropic",
				"sk-test-1",
				"",
				"https://api.anthropic.com",
				"claude-3-opus",
				[]string{"claude-3-opus"},
			),
			crypto.NewExportConfig(
				"new-config-2",
				"anthropic",
				"sk-test-2",
				"",
				"https://api.anthropic.com",
				"claude-3-sonnet",
				[]string{"claude-3-sonnet"},
			),
		},
	}

	existingMap := map[string]bool{}
	skip := false
	overwrite := false

	imported, skipped, overwritten, err := importConfigs(cm, export, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 2, imported)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 0, overwritten)

	// Verify configs were actually imported
	configs, _ := cm.List()
	assert.Len(t, configs, 2)
}

func TestImportConfigsSkipMode(t *testing.T) {
	cm, cleanup := mockConfigManager(t)
	defer cleanup()

	// Add an existing config
	existingCfg := models.APIConfig{
		Alias:    "existing-config",
		Provider: "anthropic",
		APIKey:   "sk-old-key",
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-3-opus",
		Models:   []string{"claude-3-opus"},
	}
	err := cm.Add(existingCfg)
	require.NoError(t, err)

	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			crypto.NewExportConfig(
				"existing-config",
				"anthropic",
				"sk-new-key",
				"",
				"https://api.anthropic.com",
				"claude-3-opus",
				[]string{"claude-3-opus"},
			),
			crypto.NewExportConfig(
				"new-config",
				"anthropic",
				"sk-new-2",
				"",
				"https://api.anthropic.com",
				"claude-3-sonnet",
				[]string{"claude-3-sonnet"},
			),
		},
	}

	existingMap := map[string]bool{
		"existing-config": true,
	}
	skip := true
	overwrite := false

	imported, skipped, overwritten, err := importConfigs(cm, export, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 1, imported)
	assert.Equal(t, 1, skipped)
	assert.Equal(t, 0, overwritten)

	// Verify existing config was NOT overwritten
	configs, _ := cm.List()
	for _, cfg := range configs {
		if cfg.Alias == "existing-config" {
			assert.Equal(t, "sk-old-key", cfg.APIKey)
		}
	}
}

func TestImportConfigsOverwriteMode(t *testing.T) {
	cm, cleanup := mockConfigManager(t)
	defer cleanup()

	// Add an existing config
	existingCfg := models.APIConfig{
		Alias:    "existing-config",
		Provider: "anthropic",
		APIKey:   "sk-old-key",
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-3-opus",
		Models:   []string{"claude-3-opus"},
	}
	err := cm.Add(existingCfg)
	require.NoError(t, err)

	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			crypto.NewExportConfig(
				"existing-config",
				"anthropic",
				"sk-new-key",
				"",
				"https://api.anthropic.com",
				"claude-3-sonnet",
				[]string{"claude-3-sonnet"},
			),
		},
	}

	existingMap := map[string]bool{
		"existing-config": true,
	}
	skip := false
	overwrite := true

	imported, skipped, overwritten, err := importConfigs(cm, export, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 0, imported)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 1, overwritten)

	// Verify existing config WAS overwritten
	cfg, err := cm.Get("existing-config")
	require.NoError(t, err)
	assert.Equal(t, "sk-new-key", cfg.APIKey)
	assert.Equal(t, "claude-3-sonnet", cfg.Model)
}

func TestImportConfigsDefaultMode(t *testing.T) {
	cm, cleanup := mockConfigManager(t)
	defer cleanup()

	// Add an existing config
	existingCfg := models.APIConfig{
		Alias:    "existing-config",
		Provider: "anthropic",
		APIKey:   "sk-old-key",
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-3-opus",
		Models:   []string{"claude-3-opus"},
	}
	err := cm.Add(existingCfg)
	require.NoError(t, err)

	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			crypto.NewExportConfig(
				"existing-config",
				"anthropic",
				"sk-new-key",
				"",
				"https://api.anthropic.com",
				"claude-3-opus",
				[]string{"claude-3-opus"},
			),
			crypto.NewExportConfig(
				"new-config",
				"anthropic",
				"sk-new-2",
				"",
				"https://api.anthropic.com",
				"claude-3-sonnet",
				[]string{"claude-3-sonnet"},
			),
		},
	}

	existingMap := map[string]bool{
		"existing-config": true,
	}
	skip := false
	overwrite := false // Default mode

	imported, skipped, overwritten, err := importConfigs(cm, export, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 1, imported)
	assert.Equal(t, 1, skipped)
	assert.Equal(t, 0, overwritten)

	// Verify existing config was NOT changed in default mode
	cfg, err := cm.Get("existing-config")
	require.NoError(t, err)
	assert.Equal(t, "sk-old-key", cfg.APIKey)
}

func TestImportConfigsEmptyExport(t *testing.T) {
	cm, cleanup := mockConfigManager(t)
	defer cleanup()

	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{},
	}

	existingMap := map[string]bool{}
	skip := false
	overwrite := false

	imported, skipped, overwritten, err := importConfigs(cm, export, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 0, imported)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 0, overwritten)
}

func TestImportConfigsMixedProviders(t *testing.T) {
	cm, cleanup := mockConfigManager(t)
	defer cleanup()

	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			crypto.NewExportConfig(
				"anthropic-config",
				"anthropic",
				"sk-ant-test",
				"",
				"https://api.anthropic.com",
				"claude-3-opus",
				[]string{"claude-3-opus"},
			),
			crypto.NewExportConfig(
				"openai-config",
				"openai",
				"sk-openai-test-key",
				"",
				"https://api.openai.com",
				"gpt-4",
				[]string{"gpt-4"},
			),
			crypto.NewExportConfig(
				"anthropic-custom",
				"anthropic",
				"sk-ant-custom",
				"",
				"https://custom.anthropic.com",
				"custom-model",
				nil,
			),
		},
	}

	existingMap := map[string]bool{}
	skip := false
	overwrite := true

	imported, skipped, overwritten, err := importConfigs(cm, export, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 3, imported)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 0, overwritten)

	// Verify all configs with different providers
	configs, _ := cm.List()
	providers := make(map[string]bool)
	for _, cfg := range configs {
		providers[cfg.Provider] = true
	}

	assert.True(t, providers["anthropic"])
	assert.True(t, providers["openai"])
}

func TestImportConfigsWithModelsList(t *testing.T) {
	cm, cleanup := mockConfigManager(t)
	defer cleanup()

	export := &crypto.ExportFormat{
		Version: "1",
		Configs: []crypto.ExportConfig{
			crypto.NewExportConfig(
				"multi-model-config",
				"anthropic",
				"sk-test",
				"",
				"https://api.anthropic.com",
				"claude-3-opus",
				[]string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
			),
		},
	}

	existingMap := map[string]bool{}
	skip := false
	overwrite := false

	imported, skipped, overwritten, err := importConfigs(cm, export, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 1, imported)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 0, overwritten)

	// Verify models list was preserved
	cfg, err := cm.Get("multi-model-config")
	require.NoError(t, err)
	assert.Len(t, cfg.Models, 3)
	assert.Contains(t, cfg.Models, "claude-3-opus")
	assert.Contains(t, cfg.Models, "claude-3-sonnet")
	assert.Contains(t, cfg.Models, "claude-3-haiku")
}

// mockConfigManager creates a config manager with test data
func mockConfigManager(t *testing.T) (*config.Manager, func()) {
	t.Helper()

	// Create a temp directory for config
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	// Create config manager
	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	// Cleanup function
	cleanup := func() {
		os.Setenv("HOME", oldHome)
	}

	return cm, cleanup
}

func TestExportImportRoundTrip(t *testing.T) {
	// Create a config manager and add a test config
	cmExport, cleanupExport := mockConfigManager(t)
	defer cleanupExport()

	testCfg := models.APIConfig{
		Alias:    "roundtrip-test",
		Provider: "anthropic",
		APIKey:   "sk-roundtrip-test",
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-3-opus",
		Models:   []string{"claude-3-opus", "claude-3-sonnet"},
	}
	err := cmExport.Add(testCfg)
	require.NoError(t, err)

	// Export
	configs, err := cmExport.List()
	require.NoError(t, err)

	exportConfigs := make([]crypto.ExportConfig, len(configs))
	for i, cfg := range configs {
		exportConfigs[i] = crypto.NewExportConfig(
			cfg.Alias,
			cfg.Provider,
			cfg.APIKey,
			cfg.AuthToken,
			cfg.BaseURL,
			cfg.Model,
			cfg.Models,
		)
	}

	password := "test-password"
	pm := crypto.NewPasswordManager(password)
	export := crypto.NewExportFormat(exportConfigs)

	encrypted, err := pm.Encrypt(export)
	require.NoError(t, err)

	// Decrypt
	decrypted, err := pm.Decrypt(encrypted)
	require.NoError(t, err)

	// Import into a new config manager
	cmImport, cleanupImport := mockConfigManager(t)
	defer cleanupImport()

	existingMap := map[string]bool{}
	skip := false
	overwrite := false

	imported, skipped, overwritten, err := importConfigs(cmImport, decrypted, existingMap, skip, overwrite)
	require.NoError(t, err)

	assert.Equal(t, 1, imported)
	assert.Equal(t, 0, skipped)
	assert.Equal(t, 0, overwritten)

	// Verify imported config matches original
	importedCfgs, _ := cmImport.List()
	require.Len(t, importedCfgs, 1)

	importedCfg := importedCfgs[0]
	assert.Equal(t, testCfg.Alias, importedCfg.Alias)
	assert.Equal(t, testCfg.Provider, importedCfg.Provider)
	assert.Equal(t, testCfg.APIKey, importedCfg.APIKey)
	assert.Equal(t, testCfg.BaseURL, importedCfg.BaseURL)
	assert.Equal(t, testCfg.Model, importedCfg.Model)
	assert.ElementsMatch(t, testCfg.Models, importedCfg.Models)
}
