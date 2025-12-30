package cmd

import (
	"testing"
)

func TestAddCmd(t *testing.T) {
	t.Run("Command definition", func(t *testing.T) {
		expected := "add [alias]"
		if addCmd.Use != expected {
			t.Errorf("addCmd.Use = %q, want %q", addCmd.Use, expected)
		}
	})

	t.Run("Short description", func(t *testing.T) {
		if addCmd.Short == "" {
			t.Error("addCmd.Short should not be empty")
		}
	})

	t.Run("Long description", func(t *testing.T) {
		if addCmd.Long == "" {
			t.Error("addCmd.Long should not be empty")
		}
	})

	t.Run("RunE is set", func(t *testing.T) {
		if addCmd.RunE == nil {
			t.Error("addCmd.RunE should not be nil")
		}
	})

	t.Run("Flags are defined", func(t *testing.T) {
		flags := []struct {
			name     string
			shortcut string
		}{
			{"sk", ""},
			{"ak", ""},
			{"url", "u"},
			{"model", "m"},
		}

		for _, f := range flags {
			flag := addCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Errorf("Flag --%s should be defined", f.name)
				continue
			}
			if f.shortcut != "" && flag.Shorthand != f.shortcut {
				t.Errorf("Flag --%s shorthand = %q, want %q", f.name, flag.Shorthand, f.shortcut)
			}
		}
	})
}

func TestAPIConfigBuilder(t *testing.T) {
	t.Run("Build with valid config", func(t *testing.T) {
		builder := NewAPIConfigBuilder().
			SetAlias("test-alias").
			SetAPIKey("sk-test-key").
			SetBaseURL("https://api.example.com").
			SetModel("claude-3")

		cfg, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v, want nil", err)
		}

		if cfg.Alias != "test-alias" {
			t.Errorf("Alias = %q, want %q", cfg.Alias, "test-alias")
		}
		if cfg.APIKey != "sk-test-key" {
			t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test-key")
		}
		if cfg.BaseURL != "https://api.example.com" {
			t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.example.com")
		}
		if cfg.Model != "claude-3" {
			t.Errorf("Model = %q, want %q", cfg.Model, "claude-3")
		}
	})

	t.Run("Build with auth token", func(t *testing.T) {
		builder := NewAPIConfigBuilder().
			SetAlias("test-alias").
			SetAuthToken("bearer-token")

		cfg, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v, want nil", err)
		}

		if cfg.AuthToken != "bearer-token" {
			t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "bearer-token")
		}
	})

	t.Run("Build fails with empty alias", func(t *testing.T) {
		builder := NewAPIConfigBuilder().
			SetAPIKey("sk-test-key")

		_, err := builder.Build()
		if err == nil {
			t.Error("Build() should return error for empty alias")
		}
	})

	t.Run("Build fails with no auth", func(t *testing.T) {
		builder := NewAPIConfigBuilder().
			SetAlias("test-alias")

		_, err := builder.Build()
		if err == nil {
			t.Error("Build() should return error when both API key and auth token are empty")
		}
	})

	t.Run("Build fails with invalid URL", func(t *testing.T) {
		builder := NewAPIConfigBuilder().
			SetAlias("test-alias").
			SetAPIKey("sk-test-key").
			SetBaseURL("not-a-valid-url")

		_, err := builder.Build()
		if err == nil {
			t.Error("Build() should return error for invalid URL")
		}
	})
}
