package cmd

import (
	"testing"

	"github.com/ccasJay/apimgr/config"
	"github.com/ccasJay/apimgr/config/models"
	"github.com/ccasJay/apimgr/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			width:    10,
			expected: "hello",
		},
		{
			name:     "exact width",
			input:    "0123456789",
			width:    10,
			expected: "0123456789",
		},
		{
			name:     "long string",
			input:    "012345678901234567890",
			width:    10,
			expected: "0123456789\n0123456789\n0",
		},
		{
			name:     "empty string",
			input:    "",
			width:    10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapString(tt.input, tt.width)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExportSingleConfig(t *testing.T) {
	// Create a test config
	testCfg := models.APIConfig{
		Alias:    "test-export",
		Provider: "anthropic",
		APIKey:   "sk-test-key",
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-3-opus",
		Models:   []string{"claude-3-opus", "claude-3-sonnet"},
	}

	// Test encryption logic
	password := "testpass"
	pm := crypto.NewPasswordManager(password)

	exportConfigs := []crypto.ExportConfig{
		crypto.NewExportConfig(
			testCfg.Alias,
			testCfg.Provider,
			testCfg.APIKey,
			testCfg.AuthToken,
			testCfg.BaseURL,
			testCfg.Model,
			testCfg.Models,
		),
	}
	export := crypto.NewExportFormat(exportConfigs)

	encrypted, err := pm.Encrypt(export)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	// Decrypt to verify
	decrypted, err := pm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, testCfg.Alias, decrypted.Configs[0].Alias)
	assert.Equal(t, testCfg.APIKey, decrypted.Configs[0].APIKey)
	assert.Equal(t, testCfg.Model, decrypted.Configs[0].Model)
}

func TestExportMultipleConfigs(t *testing.T) {
	password := "testpass"
	pm := crypto.NewPasswordManager(password)

	configs := []crypto.ExportConfig{
		crypto.NewExportConfig(
			"config-1",
			"anthropic",
			"sk-test-1",
			"",
			"https://api.anthropic.com",
			"claude-3-opus",
			[]string{"claude-3-opus"},
		),
		crypto.NewExportConfig(
			"config-2",
			"openai",
			"sk-openai-test-key",
			"",
			"https://api.openai.com",
			"gpt-4",
			[]string{"gpt-4", "gpt-3.5"},
		),
		crypto.NewExportConfig(
			"config-3",
			"anthropic",
			"sk-ant-custom",
			"",
			"https://custom.com",
			"custom-model",
			nil,
		),
	}

	export := crypto.NewExportFormat(configs)

	encrypted, err := pm.Encrypt(export)
	require.NoError(t, err)

	decrypted, err := pm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Len(t, decrypted.Configs, 3)

	// Verify all configs
	aliases := []string{}
	for _, cfg := range decrypted.Configs {
		aliases = append(aliases, cfg.Alias)
	}
	assert.Contains(t, aliases, "config-1")
	assert.Contains(t, aliases, "config-2")
	assert.Contains(t, aliases, "config-3")
}

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		expectError bool
	}{
		{"valid password", "password123", false},
		{"exactly 8 chars", "12345678", false},
		{"too short", "short", true},
		{"empty", "", true},
		{"7 chars", "1234567", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := crypto.ValidatePassword(tt.password)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExportWithEncryptedAPIKey(t *testing.T) {
	// Test exporting configs that have encrypted API keys (from local storage)
	// This simulates real-world scenario where configs are encrypted locally

	password := "testpass"
	pm := crypto.NewPasswordManager(password)

	// Create an export with an encrypted-style API key
	exportConfigs := []crypto.ExportConfig{
		crypto.NewExportConfig(
			"encrypted-config",
			"anthropic",
			"ENC:base64Key==", // Simulates an encrypted key
			"",
			"https://api.anthropic.com",
			"claude-3-opus",
			[]string{"claude-3-opus"},
		),
	}
	export := crypto.NewExportFormat(exportConfigs)

	encrypted, err := pm.Encrypt(export)
	require.NoError(t, err)

	decrypted, err := pm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "encrypted-config", decrypted.Configs[0].Alias)
	assert.Equal(t, "ENC:base64Key==", decrypted.Configs[0].APIKey)
}

func TestExportEmptyConfigs(t *testing.T) {
	password := "testpass"
	pm := crypto.NewPasswordManager(password)

	// Export empty configs list
	export := crypto.NewExportFormat([]crypto.ExportConfig{})

	encrypted, err := pm.Encrypt(export)
	require.NoError(t, err)

	decrypted, err := pm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Empty(t, decrypted.Configs)
}

func TestExportWithAuthToken(t *testing.T) {
	// Test exporting configs with auth token instead of API key
	password := "testpass"
	pm := crypto.NewPasswordManager(password)

	exportConfigs := []crypto.ExportConfig{
		crypto.NewExportConfig(
			"auth-token-config",
			"anthropic",
			"",              // No API key
			"my-auth-token", // Using auth token instead
			"https://api.anthropic.com",
			"claude-3-opus",
			[]string{"claude-3-opus"},
		),
	}
	export := crypto.NewExportFormat(exportConfigs)

	encrypted, err := pm.Encrypt(export)
	require.NoError(t, err)

	decrypted, err := pm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "auth-token-config", decrypted.Configs[0].Alias)
	assert.Equal(t, "", decrypted.Configs[0].APIKey)
	assert.Equal(t, "my-auth-token", decrypted.Configs[0].AuthToken)
}

// mockExportConfigManager creates a config manager with test data
func mockExportConfigManager(t *testing.T) (*config.Manager, func()) {
	t.Helper()

	// Create a temp directory for config
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("APIMGR_CONFIG_DIR", tempDir)
	t.Setenv("USERPROFILE", tempDir)

	// Create config manager
	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	return cm, func() {}
}

func TestExportIntegration(t *testing.T) {
	// Integration test using actual config manager
	cm, cleanup := mockExportConfigManager(t)
	defer cleanup()

	// Add test config
	testCfg := models.APIConfig{
		Alias:    "integration-test",
		Provider: "anthropic",
		APIKey:   "sk-integration-test",
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-3-opus",
		Models:   []string{"claude-3-opus"},
	}
	err := cm.Add(testCfg)
	require.NoError(t, err)

	// List configs
	configs, err := cm.List()
	require.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.Equal(t, "integration-test", configs[0].Alias)

	// Convert to export format
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

	// Encrypt and decrypt
	password := "testpass"
	pm := crypto.NewPasswordManager(password)
	export := crypto.NewExportFormat(exportConfigs)

	encrypted, err := pm.Encrypt(export)
	require.NoError(t, err)

	decrypted, err := pm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Len(t, decrypted.Configs, 1)
	assert.Equal(t, "integration-test", decrypted.Configs[0].Alias)
}
