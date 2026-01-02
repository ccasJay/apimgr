package compatibility

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"apimgr/config/models"
	"apimgr/internal/providers"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: api-compatibility-test, Property 1: Provider-specific request format**
// **Validates: Requirements 1.1, 4.1, 4.2**
//
// *For any* API configuration with a known provider type, the request builder SHALL produce
// a request body and headers that conform to that provider's API specification.
// - Anthropic configs → Messages API format with `x-api-key` header
// - OpenAI configs → Chat Completions API format with `Authorization: Bearer` header
func TestProperty1_ProviderSpecificRequestFormat(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for non-empty strings (for API keys and models)
	// Use AlphaString with minimum length to avoid empty strings
	nonEmptyStringGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "default-value"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// Generator for valid base URLs
	baseURLGen := gen.OneConstOf(
		"https://api.anthropic.com",
		"https://api.openai.com/v1",
		"https://custom-api.example.com",
		"",
	)

	// Property: Anthropic configs produce Messages API format with x-api-key header
	properties.Property("anthropic config produces correct format", prop.ForAll(
		func(apiKey string, authToken string, baseURL string, model string) bool {
			cfg := &models.APIConfig{
				Provider:  "anthropic",
				APIKey:    apiKey,
				AuthToken: authToken,
				BaseURL:   baseURL,
				Model:     model,
			}

			provider, err := providers.Get("anthropic")
			if err != nil {
				return false
			}

			builder := NewRequestBuilder(cfg, provider)

			// Check endpoint
			if builder.GetEndpoint() != "/v1/messages" {
				return false
			}

			// Check headers
			headers := builder.GetHeaders()
			if headers["Content-Type"] != "application/json" {
				return false
			}
			if headers["anthropic-version"] != "2023-06-01" {
				return false
			}
			// Must have x-api-key if apiKey is provided
			if apiKey != "" && headers["x-api-key"] != apiKey {
				return false
			}
			// Must have Authorization if authToken is provided
			if authToken != "" && headers["Authorization"] != "Bearer "+authToken {
				return false
			}

			// Build request and verify body format
			req, err := builder.BuildChatRequest(model, false)
			if err != nil {
				return false
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return false
			}

			var reqBody AnthropicRequest
			if err := json.Unmarshal(body, &reqBody); err != nil {
				return false
			}

			// Verify Anthropic-specific fields
			if reqBody.Model != model {
				return false
			}
			if reqBody.MaxTokens != 100 {
				return false
			}
			if len(reqBody.Messages) != 1 {
				return false
			}
			if reqBody.Messages[0].Role != "user" {
				return false
			}

			return true
		},
		nonEmptyStringGen, // apiKey
		nonEmptyStringGen, // authToken
		baseURLGen,        // baseURL
		nonEmptyStringGen, // model
	))


	// Property: OpenAI configs produce Chat Completions API format with Authorization Bearer header
	properties.Property("openai config produces correct format", prop.ForAll(
		func(apiKey string, baseURL string, model string) bool {
			cfg := &models.APIConfig{
				Provider: "openai",
				APIKey:   apiKey,
				BaseURL:  baseURL,
				Model:    model,
			}

			provider, err := providers.Get("openai")
			if err != nil {
				return false
			}

			builder := NewRequestBuilder(cfg, provider)

			// Check endpoint
			if builder.GetEndpoint() != "/v1/chat/completions" {
				return false
			}

			// Check headers
			headers := builder.GetHeaders()
			if headers["Content-Type"] != "application/json" {
				return false
			}
			// Must have Authorization: Bearer header
			expectedAuth := "Bearer " + apiKey
			if headers["Authorization"] != expectedAuth {
				return false
			}

			// Build request and verify body format
			req, err := builder.BuildChatRequest(model, false)
			if err != nil {
				return false
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return false
			}

			var reqBody OpenAIRequest
			if err := json.Unmarshal(body, &reqBody); err != nil {
				return false
			}

			// Verify OpenAI-specific fields
			if reqBody.Model != model {
				return false
			}
			if reqBody.MaxTokens != 100 {
				return false
			}
			if len(reqBody.Messages) != 1 {
				return false
			}
			if reqBody.Messages[0].Role != "user" {
				return false
			}

			return true
		},
		nonEmptyStringGen, // apiKey
		baseURLGen,        // baseURL
		nonEmptyStringGen, // model
	))

	// Property: Request URL is correctly constructed from baseURL and endpoint
	properties.Property("request URL is correctly constructed", prop.ForAll(
		func(providerName string, apiKey string) bool {
			provider, err := providers.Get(providerName)
			if err != nil {
				return false
			}

			cfg := &models.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
				BaseURL:  provider.DefaultBaseURL(),
			}

			builder := NewRequestBuilder(cfg, provider)
			req, err := builder.BuildChatRequest("test-model", false)
			if err != nil {
				return false
			}

			expectedURL := strings.TrimSuffix(provider.DefaultBaseURL(), "/") + builder.GetEndpoint()
			return req.URL.String() == expectedURL
		},
		gen.OneConstOf("anthropic", "openai"),
		nonEmptyStringGen,
	))

	properties.TestingRun(t)
}

// **Feature: api-compatibility-test, Property 4: Streaming request construction**
// **Validates: Requirements 2.1**
//
// *For any* streaming test request, the request body SHALL include `stream: true` parameter.
func TestProperty4_StreamingRequestConstruction(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for non-empty strings
	nonEmptyStringGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "default-value"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// Property: Streaming requests include stream: true for Anthropic
	properties.Property("anthropic streaming request includes stream true", prop.ForAll(
		func(apiKey string, model string) bool {
			cfg := &models.APIConfig{
				Provider: "anthropic",
				APIKey:   apiKey,
			}

			provider, _ := providers.Get("anthropic")
			builder := NewRequestBuilder(cfg, provider)

			req, err := builder.BuildChatRequest(model, true)
			if err != nil {
				return false
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return false
			}

			var reqBody map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				return false
			}

			stream, ok := reqBody["stream"].(bool)
			return ok && stream == true
		},
		nonEmptyStringGen,
		nonEmptyStringGen,
	))

	// Property: Streaming requests include stream: true for OpenAI
	properties.Property("openai streaming request includes stream true", prop.ForAll(
		func(apiKey string, model string) bool {
			cfg := &models.APIConfig{
				Provider: "openai",
				APIKey:   apiKey,
			}

			provider, _ := providers.Get("openai")
			builder := NewRequestBuilder(cfg, provider)

			req, err := builder.BuildChatRequest(model, true)
			if err != nil {
				return false
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return false
			}

			var reqBody map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				return false
			}

			stream, ok := reqBody["stream"].(bool)
			return ok && stream == true
		},
		nonEmptyStringGen,
		nonEmptyStringGen,
	))

	// Property: Non-streaming requests do not include stream: true
	properties.Property("non-streaming request does not include stream true", prop.ForAll(
		func(providerName string, apiKey string, model string) bool {
			cfg := &models.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
			}

			provider, err := providers.Get(providerName)
			if err != nil {
				return false
			}

			builder := NewRequestBuilder(cfg, provider)

			req, err := builder.BuildChatRequest(model, false)
			if err != nil {
				return false
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				return false
			}

			var reqBody map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				return false
			}

			// stream should either not exist or be false
			stream, exists := reqBody["stream"]
			if !exists {
				return true
			}
			streamBool, ok := stream.(bool)
			return ok && streamBool == false
		},
		gen.OneConstOf("anthropic", "openai"),
		nonEmptyStringGen,
		nonEmptyStringGen,
	))

	properties.TestingRun(t)
}


// **Feature: api-compatibility-test, Property 8: Custom path override**
// **Validates: Requirements 4.4**
//
// *For any* test with a custom path specified, the final request URL SHALL use
// the custom path instead of the provider's default endpoint path.
func TestProperty8_CustomPathOverride(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for non-empty strings
	nonEmptyStringGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "default-value"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// Generator for valid custom paths (must start with /)
	customPathGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "/custom/endpoint"
		}
		// Ensure path starts with /
		path := "/" + s
		if len(path) > 50 {
			return path[:50]
		}
		return path
	})

	// Property: Custom path overrides default endpoint for Anthropic
	properties.Property("custom path overrides anthropic default endpoint", prop.ForAll(
		func(apiKey string, customPath string, model string) bool {
			cfg := &models.APIConfig{
				Provider: "anthropic",
				APIKey:   apiKey,
				BaseURL:  "https://api.anthropic.com",
			}

			provider, err := providers.Get("anthropic")
			if err != nil {
				return false
			}

			// Create builder with custom path
			builder := NewRequestBuilderWithCustomPath(cfg, provider, customPath)

			// Verify GetEndpoint returns custom path
			if builder.GetEndpoint() != customPath {
				return false
			}

			// Build request and verify URL uses custom path
			req, err := builder.BuildChatRequest(model, false)
			if err != nil {
				return false
			}

			// The URL should contain the custom path, not the default /v1/messages
			expectedURL := "https://api.anthropic.com" + customPath
			if req.URL.String() != expectedURL {
				return false
			}

			// Verify the default endpoint is NOT in the URL
			defaultEndpoint := "/v1/messages"
			if strings.Contains(req.URL.String(), defaultEndpoint) && customPath != defaultEndpoint {
				return false
			}

			return true
		},
		nonEmptyStringGen, // apiKey
		customPathGen,     // customPath
		nonEmptyStringGen, // model
	))

	// Property: Custom path overrides default endpoint for OpenAI
	properties.Property("custom path overrides openai default endpoint", prop.ForAll(
		func(apiKey string, customPath string, model string) bool {
			cfg := &models.APIConfig{
				Provider: "openai",
				APIKey:   apiKey,
				BaseURL:  "https://api.openai.com",
			}

			provider, err := providers.Get("openai")
			if err != nil {
				return false
			}

			// Create builder with custom path
			builder := NewRequestBuilderWithCustomPath(cfg, provider, customPath)

			// Verify GetEndpoint returns custom path
			if builder.GetEndpoint() != customPath {
				return false
			}

			// Build request and verify URL uses custom path
			req, err := builder.BuildChatRequest(model, false)
			if err != nil {
				return false
			}

			// The URL should contain the custom path, not the default /v1/chat/completions
			expectedURL := "https://api.openai.com" + customPath
			if req.URL.String() != expectedURL {
				return false
			}

			// Verify the default endpoint is NOT in the URL
			defaultEndpoint := "/v1/chat/completions"
			if strings.Contains(req.URL.String(), defaultEndpoint) && customPath != defaultEndpoint {
				return false
			}

			return true
		},
		nonEmptyStringGen, // apiKey
		customPathGen,     // customPath
		nonEmptyStringGen, // model
	))

	// Property: Empty custom path falls back to default endpoint
	properties.Property("empty custom path uses default endpoint", prop.ForAll(
		func(providerName string, apiKey string, model string) bool {
			cfg := &models.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
			}

			provider, err := providers.Get(providerName)
			if err != nil {
				return false
			}

			// Create builder with empty custom path (should use default)
			builder := NewRequestBuilderWithCustomPath(cfg, provider, "")

			// Get expected default endpoint
			var expectedEndpoint string
			switch providerName {
			case "anthropic":
				expectedEndpoint = "/v1/messages"
			case "openai":
				expectedEndpoint = "/v1/chat/completions"
			default:
				return false
			}

			// Verify GetEndpoint returns default endpoint
			if builder.GetEndpoint() != expectedEndpoint {
				return false
			}

			return true
		},
		gen.OneConstOf("anthropic", "openai"),
		nonEmptyStringGen,
		nonEmptyStringGen,
	))

	// Property: Custom path works with streaming requests
	properties.Property("custom path works with streaming requests", prop.ForAll(
		func(providerName string, apiKey string, customPath string, model string) bool {
			cfg := &models.APIConfig{
				Provider: providerName,
				APIKey:   apiKey,
				BaseURL:  "https://api.example.com",
			}

			provider, err := providers.Get(providerName)
			if err != nil {
				return false
			}

			// Create builder with custom path
			builder := NewRequestBuilderWithCustomPath(cfg, provider, customPath)

			// Build streaming request
			req, err := builder.BuildChatRequest(model, true)
			if err != nil {
				return false
			}

			// Verify URL uses custom path
			expectedURL := "https://api.example.com" + customPath
			if req.URL.String() != expectedURL {
				return false
			}

			// Verify streaming is enabled in body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return false
			}

			var reqBody map[string]interface{}
			if err := json.Unmarshal(body, &reqBody); err != nil {
				return false
			}

			stream, ok := reqBody["stream"].(bool)
			return ok && stream == true
		},
		gen.OneConstOf("anthropic", "openai"),
		nonEmptyStringGen,
		customPathGen,
		nonEmptyStringGen,
	))

	properties.TestingRun(t)
}
