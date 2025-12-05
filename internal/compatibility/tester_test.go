package compatibility

import (
	"testing"

	"apimgr/config"
	"apimgr/internal/providers"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: api-compatibility-test, Property 3: Default model fallback**
// **Validates: Requirements 1.4**
//
// *For any* API configuration where the model field is empty, the tester SHALL use
// the provider's default model when constructing the request.
func TestProperty3_DefaultModelFallback(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for non-empty strings (for API keys)
	nonEmptyStringGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "default-value"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// Generator for provider names
	providerNameGen := gen.OneConstOf("anthropic", "openai")

	// Property: When model is empty, tester uses provider's default model
	properties.Property("empty model config uses provider default model", prop.ForAll(
		func(providerName string, apiKey string) bool {
			cfg := &config.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
				Model:    "", // Empty model
			}

			tester, err := NewTester(cfg)
			if err != nil {
				return false
			}

			provider, err := providers.Get(providerName)
			if err != nil {
				return false
			}

			// The tester should return the provider's default model
			actualModel := tester.GetModel()
			expectedModel := provider.DefaultModel()

			return actualModel == expectedModel
		},
		providerNameGen,
		nonEmptyStringGen,
	))

	// Property: When model is specified, tester uses the specified model
	properties.Property("specified model config uses specified model", prop.ForAll(
		func(providerName string, apiKey string, model string) bool {
			cfg := &config.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
				Model:    model, // Specified model
			}

			tester, err := NewTester(cfg)
			if err != nil {
				return false
			}

			// The tester should return the specified model
			actualModel := tester.GetModel()

			return actualModel == model
		},
		providerNameGen,
		nonEmptyStringGen,
		nonEmptyStringGen, // model (non-empty)
	))

	// Property: Default model is never empty
	properties.Property("default model is never empty", prop.ForAll(
		func(providerName string, apiKey string) bool {
			cfg := &config.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
				Model:    "", // Empty model
			}

			tester, err := NewTester(cfg)
			if err != nil {
				return false
			}

			// The model should never be empty
			actualModel := tester.GetModel()
			return actualModel != ""
		},
		providerNameGen,
		nonEmptyStringGen,
	))

	// Property: Provider resolution works correctly
	properties.Property("provider is correctly resolved from config", prop.ForAll(
		func(providerName string, apiKey string) bool {
			cfg := &config.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
			}

			tester, err := NewTester(cfg)
			if err != nil {
				return false
			}

			// The resolved provider should match the config
			resolvedProvider := tester.GetProvider()
			return resolvedProvider.Name() == providerName
		},
		providerNameGen,
		nonEmptyStringGen,
	))

	// Property: Empty provider defaults to anthropic
	properties.Property("empty provider defaults to anthropic", prop.ForAll(
		func(apiKey string) bool {
			cfg := &config.APIConfig{
				Provider: "", // Empty provider
				APIKey:   apiKey,
			}

			tester, err := NewTester(cfg)
			if err != nil {
				return false
			}

			// Should default to anthropic
			resolvedProvider := tester.GetProvider()
			return resolvedProvider.Name() == "anthropic"
		},
		nonEmptyStringGen,
	))

	properties.TestingRun(t)
}

// TestNewTester_NilConfig tests that NewTester returns an error for nil config
func TestNewTester_NilConfig(t *testing.T) {
	_, err := NewTester(nil)
	if err == nil {
		t.Error("expected error for nil config, got nil")
	}
}

// TestNewTester_InvalidProvider tests that NewTester returns an error for invalid provider
func TestNewTester_InvalidProvider(t *testing.T) {
	cfg := &config.APIConfig{
		Provider: "invalid-provider",
		APIKey:   "test-key",
	}

	_, err := NewTester(cfg)
	if err == nil {
		t.Error("expected error for invalid provider, got nil")
	}
}

// TestTesterOptions tests the functional options for Tester
func TestTesterOptions(t *testing.T) {
	cfg := &config.APIConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
	}

	// Test WithVerbose
	tester, err := NewTester(cfg, WithVerbose(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tester.verbose {
		t.Error("expected verbose to be true")
	}

	// Test WithCustomPath
	tester, err = NewTester(cfg, WithCustomPath("/custom/path"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tester.customPath != "/custom/path" {
		t.Errorf("expected customPath to be '/custom/path', got '%s'", tester.customPath)
	}
}

// TestDetectProviderFromURL tests the provider auto-detection from URL
func TestDetectProviderFromURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedProvider string
		expectedOK     bool
	}{
		// Anthropic URLs
		{"anthropic api url", "https://api.anthropic.com", "anthropic", true},
		{"anthropic api url with path", "https://api.anthropic.com/v1/messages", "anthropic", true},
		{"anthropic api url with port", "https://api.anthropic.com:443", "anthropic", true},
		{"anthropic subdomain", "https://proxy.api.anthropic.com", "anthropic", true},
		
		// OpenAI URLs
		{"openai api url", "https://api.openai.com", "openai", true},
		{"openai api url with path", "https://api.openai.com/v1/chat/completions", "openai", true},
		{"openai api url with port", "https://api.openai.com:443", "openai", true},
		{"openai subdomain", "https://proxy.api.openai.com", "openai", true},
		
		// Unknown/ambiguous URLs
		{"localhost", "http://localhost:8080", "", false},
		{"custom domain", "https://my-llm-proxy.example.com", "", false},
		{"empty url", "", "", false},
		{"invalid url", "not-a-url", "", false},
		
		// Case insensitivity
		{"uppercase anthropic", "https://API.ANTHROPIC.COM", "anthropic", true},
		{"mixed case openai", "https://Api.OpenAI.Com", "openai", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, ok := DetectProviderFromURL(tt.url)
			if ok != tt.expectedOK {
				t.Errorf("DetectProviderFromURL(%q) ok = %v, want %v", tt.url, ok, tt.expectedOK)
			}
			if provider != tt.expectedProvider {
				t.Errorf("DetectProviderFromURL(%q) provider = %q, want %q", tt.url, provider, tt.expectedProvider)
			}
		})
	}
}

// TestNewTester_AutoDetectProvider tests that NewTester auto-detects provider from URL
func TestNewTester_AutoDetectProvider(t *testing.T) {
	tests := []struct {
		name             string
		baseURL          string
		expectedProvider string
	}{
		{"anthropic url auto-detect", "https://api.anthropic.com", "anthropic"},
		{"openai url auto-detect", "https://api.openai.com", "openai"},
		{"unknown url defaults to anthropic", "https://my-proxy.example.com", "anthropic"},
		{"empty url defaults to anthropic", "", "anthropic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.APIConfig{
				Provider: "", // Empty provider to trigger auto-detection
				APIKey:   "test-key",
				BaseURL:  tt.baseURL,
			}

			tester, err := NewTester(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tester.GetProvider().Name() != tt.expectedProvider {
				t.Errorf("expected provider %q, got %q", tt.expectedProvider, tester.GetProvider().Name())
			}
		})
	}
}

// TestNewTester_ExplicitProviderOverridesAutoDetect tests that explicit provider takes precedence
func TestNewTester_ExplicitProviderOverridesAutoDetect(t *testing.T) {
	// Even with an OpenAI URL, explicit anthropic provider should be used
	cfg := &config.APIConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com", // OpenAI URL
	}

	tester, err := NewTester(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use explicit provider, not auto-detected
	if tester.GetProvider().Name() != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", tester.GetProvider().Name())
	}
}

// TestWasProviderAutoDetected tests the WasProviderAutoDetected method
func TestWasProviderAutoDetected(t *testing.T) {
	// Test with explicit provider
	cfg1 := &config.APIConfig{
		Provider: "anthropic",
		APIKey:   "test-key",
	}
	tester1, _ := NewTester(cfg1)
	if tester1.WasProviderAutoDetected() {
		t.Error("expected WasProviderAutoDetected to be false for explicit provider")
	}

	// Test with auto-detected provider
	cfg2 := &config.APIConfig{
		Provider: "", // Empty to trigger auto-detection
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com",
	}
	tester2, _ := NewTester(cfg2)
	if !tester2.WasProviderAutoDetected() {
		t.Error("expected WasProviderAutoDetected to be true for auto-detected provider")
	}
}
