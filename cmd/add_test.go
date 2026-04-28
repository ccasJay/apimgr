package cmd

import (
	"os"
	"reflect"
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
			{"models", ""},
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

func TestCollectInteractivelyPreservesPresetDefaults(t *testing.T) {
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error: %v", err)
	}
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = reader.Close()
	})
	os.Stdin = reader

	if _, err := writer.WriteString("preset-alias\n"); err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	cfg, err := (&InputCollector{}).CollectInteractively(InteractiveDefaults{
		APIKey:  "sk-from-flag",
		BaseURL: "https://api.example.com",
		Model:   "claude-3-opus",
		Models:  []string{"claude-3-opus", "claude-3-sonnet"},
	})
	if err != nil {
		t.Fatalf("CollectInteractively() error: %v", err)
	}

	if cfg.Alias != "preset-alias" {
		t.Fatalf("Alias = %q, want preset-alias", cfg.Alias)
	}
	if cfg.APIKey != "sk-from-flag" {
		t.Fatalf("APIKey = %q, want sk-from-flag", cfg.APIKey)
	}
	if cfg.BaseURL != "https://api.example.com" {
		t.Fatalf("BaseURL = %q, want https://api.example.com", cfg.BaseURL)
	}
	if cfg.Model != "claude-3-opus" {
		t.Fatalf("Model = %q, want claude-3-opus", cfg.Model)
	}
	if !reflect.DeepEqual(cfg.Models, []string{"claude-3-opus", "claude-3-sonnet"}) {
		t.Fatalf("Models = %#v, want preset models", cfg.Models)
	}
}

func TestCollectInteractivelyPreservesAuthTokenPreset(t *testing.T) {
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() error: %v", err)
	}
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = reader.Close()
	})
	os.Stdin = reader

	if _, err := writer.WriteString("token-alias\nhttps://api.example.com\n\n"); err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	cfg, err := (&InputCollector{}).CollectInteractively(InteractiveDefaults{
		AuthToken: "token-from-flag",
	})
	if err != nil {
		t.Fatalf("CollectInteractively() error: %v", err)
	}

	if cfg.AuthToken != "token-from-flag" {
		t.Fatalf("AuthToken = %q, want token-from-flag", cfg.AuthToken)
	}
	if cfg.APIKey != "" {
		t.Fatalf("APIKey = %q, want empty", cfg.APIKey)
	}
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

	t.Run("Build with models list", func(t *testing.T) {
		models := []string{"claude-3-opus", "claude-3-sonnet", "gpt-4"}
		builder := NewAPIConfigBuilder().
			SetAlias("test-alias").
			SetAPIKey("sk-test-key").
			SetModel("claude-3-opus").
			SetModels(models)

		cfg, err := builder.Build()
		if err != nil {
			t.Fatalf("Build() error = %v, want nil", err)
		}

		if cfg.Model != "claude-3-opus" {
			t.Errorf("Model = %q, want %q", cfg.Model, "claude-3-opus")
		}
		if len(cfg.Models) != 3 {
			t.Errorf("len(Models) = %d, want 3", len(cfg.Models))
		}
		for i, expected := range models {
			if cfg.Models[i] != expected {
				t.Errorf("Models[%d] = %q, want %q", i, cfg.Models[i], expected)
			}
		}
	})
}

func TestParseModelsList(t *testing.T) {
	t.Run("Parse comma-separated models", func(t *testing.T) {
		result := parseModelsList("claude-3-opus,claude-3-sonnet,gpt-4")
		expected := []string{"claude-3-opus", "claude-3-sonnet", "gpt-4"}

		if len(result) != len(expected) {
			t.Fatalf("len(result) = %d, want %d", len(result), len(expected))
		}
		for i, v := range expected {
			if result[i] != v {
				t.Errorf("result[%d] = %q, want %q", i, result[i], v)
			}
		}
	})

	t.Run("Parse with whitespace", func(t *testing.T) {
		result := parseModelsList("  claude-3-opus , claude-3-sonnet , gpt-4  ")
		expected := []string{"claude-3-opus", "claude-3-sonnet", "gpt-4"}

		if len(result) != len(expected) {
			t.Fatalf("len(result) = %d, want %d", len(result), len(expected))
		}
		for i, v := range expected {
			if result[i] != v {
				t.Errorf("result[%d] = %q, want %q", i, result[i], v)
			}
		}
	})

	t.Run("Parse empty string", func(t *testing.T) {
		result := parseModelsList("")
		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
		}
	})

	t.Run("Parse single model", func(t *testing.T) {
		result := parseModelsList("claude-3-opus")
		if len(result) != 1 {
			t.Fatalf("len(result) = %d, want 1", len(result))
		}
		if result[0] != "claude-3-opus" {
			t.Errorf("result[0] = %q, want %q", result[0], "claude-3-opus")
		}
	})

	t.Run("Parse with empty entries", func(t *testing.T) {
		result := parseModelsList("claude-3-opus,,gpt-4,")
		expected := []string{"claude-3-opus", "gpt-4"}

		if len(result) != len(expected) {
			t.Fatalf("len(result) = %d, want %d", len(result), len(expected))
		}
		for i, v := range expected {
			if result[i] != v {
				t.Errorf("result[%d] = %q, want %q", i, result[i], v)
			}
		}
	})
}
