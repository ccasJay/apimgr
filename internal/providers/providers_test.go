package providers

import (
	"strings"
	"testing"
)

func TestAnthropicProvider(t *testing.T) {
	p := &AnthropicProvider{}

	t.Run("Name", func(t *testing.T) {
		if got := p.Name(); got != "anthropic" {
			t.Errorf("Name() = %v, want %v", got, "anthropic")
		}
	})

	t.Run("DefaultBaseURL", func(t *testing.T) {
		if got := p.DefaultBaseURL(); got != "https://api.anthropic.com" {
			t.Errorf("DefaultBaseURL() = %v, want %v", got, "https://api.anthropic.com")
		}
	})

	t.Run("DefaultModel", func(t *testing.T) {
		if got := p.DefaultModel(); got != "claude-3-sonnet-20240229" {
			t.Errorf("DefaultModel() = %v, want %v", got, "claude-3-sonnet-20240229")
		}
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		tests := []struct {
			name      string
			apiKey    string
			authToken string
			wantErr   bool
		}{
			{"valid with apiKey", "test-api-123", "", false},
			{"valid with authToken", "", "auth-token-123", false},
			{"valid with both", "test-api-123", "auth-token-123", false},
			{"invalid without both", "", "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := p.ValidateConfig("https://api.anthropic.com", tt.apiKey, tt.authToken)
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("NormalizeConfig", func(t *testing.T) {
		tests := []struct {
			name     string
			baseURL  string
			expected string
		}{
			{"URL without trailing slash", "https://api.anthropic.com", "https://api.anthropic.com/"},
			{"URL with trailing slash", "https://api.anthropic.com/", "https://api.anthropic.com/"},
			{"Custom URL without trailing slash", "https://custom.api.com", "https://custom.api.com/"},
			{"Empty URL", "", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := p.NormalizeConfig(tt.baseURL); got != tt.expected {
					t.Errorf("NormalizeConfig() = %v, want %v", got, tt.expected)
				}
			})
		}
	})
}

func TestOpenAIProvider(t *testing.T) {
	p := &OpenAIProvider{}

	t.Run("Name", func(t *testing.T) {
		if got := p.Name(); got != "openai" {
			t.Errorf("Name() = %v, want %v", got, "openai")
		}
	})

	t.Run("DefaultBaseURL", func(t *testing.T) {
		if got := p.DefaultBaseURL(); got != "https://api.openai.com/v1" {
			t.Errorf("DefaultBaseURL() = %v, want %v", got, "https://api.openai.com/v1")
		}
	})

	t.Run("DefaultModel", func(t *testing.T) {
		if got := p.DefaultModel(); got != "gpt-4" {
			t.Errorf("DefaultModel() = %v, want %v", got, "gpt-4")
		}
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		tests := []struct {
			name    string
			apiKey  string
			wantErr bool
		}{
			{"valid with apiKey", "test-123456", false},
			{"invalid without apiKey", "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := p.ValidateConfig("https://api.openai.com", tt.apiKey, "")
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("NormalizeConfig", func(t *testing.T) {
		tests := []struct {
			name     string
			baseURL  string
			expected string
		}{
			{"URL without trailing slash", "https://api.openai.com/v1", "https://api.openai.com/v1/"},
			{"URL with trailing slash", "https://api.openai.com/v1/", "https://api.openai.com/v1/"},
			{"Custom URL without trailing slash", "https://custom.api.com", "https://custom.api.com/"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := p.NormalizeConfig(tt.baseURL); got != tt.expected {
					t.Errorf("NormalizeConfig() = %v, want %v", got, tt.expected)
				}
			})
		}
	})
}

func TestRegistry(t *testing.T) {
	// Test that built-in providers are registered
	t.Run("Get registered providers", func(t *testing.T) {
		providers := []string{"anthropic", "openai"}
		for _, name := range providers {
			t.Run(name, func(t *testing.T) {
				p, err := Get(name)
				if err != nil {
					t.Errorf("Get(%q) error = %v, want nil", name, err)
				}
				if p == nil {
					t.Errorf("Get(%q) = nil, want provider", name)
				}
			})
		}
	})

	t.Run("Get unknown provider", func(t *testing.T) {
		_, err := Get("unknown")
		if err == nil {
			t.Error("Get(\"unknown\") error = nil, want error")
		}
		if !strings.Contains(err.Error(), "unknown provider") {
			t.Errorf("Get(\"unknown\") error = %v, want error containing 'unknown provider'", err)
		}
	})

	t.Run("List providers", func(t *testing.T) {
		list := List()
		if len(list) < 2 {
			t.Errorf("List() returned %d providers, want at least 2", len(list))
		}

		// Check that built-in providers are in the list
		hasAnthropic, hasOpenAI := false, false
		for _, name := range list {
			if name == "anthropic" {
				hasAnthropic = true
			}
			if name == "openai" {
				hasOpenAI = true
			}
		}

		if !hasAnthropic {
			t.Error("List() does not contain 'anthropic' provider")
		}
		if !hasOpenAI {
			t.Error("List() does not contain 'openai' provider")
		}
	})

	t.Run("Register custom provider", func(t *testing.T) {
		// Create a mock provider for testing
		mockProvider := &AnthropicProvider{} // Reuse for simplicity
		Register("custom", mockProvider)

		// Test that the custom provider is registered
		p, err := Get("custom")
		if err != nil {
			t.Errorf("Get(\"custom\") error = %v, want nil", err)
		}
		if p == nil {
			t.Errorf("Get(\"custom\") = nil, want provider")
		}

		// Clean up
		delete(registry, "custom")
	})
}

func TestProviderIntegration(t *testing.T) {
	t.Run("All providers implement interface", func(t *testing.T) {
		providers := List()
		for _, name := range providers {
			t.Run(name, func(t *testing.T) {
				p, err := Get(name)
				if err != nil {
					t.Fatalf("Get(%q) error = %v", name, err)
				}

				// Test all interface methods
				_ = p.Name()
				_ = p.DefaultBaseURL()
				_ = p.DefaultModel()
				_ = p.NormalizeConfig("https://example.com")

				// ValidateConfig should not panic with empty values
				_ = p.ValidateConfig("", "", "")
			})
		}
	})
}
