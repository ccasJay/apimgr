// Package compatibility provides API compatibility testing functionality
// for validating that API configurations work correctly with Claude Code.
package compatibility

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"apimgr/config"
	"apimgr/internal/providers"
)

// RequestBuilder defines the interface for building provider-specific API requests
type RequestBuilder interface {
	// BuildChatRequest builds a chat completion request for the provider
	BuildChatRequest(model string, streaming bool) (*http.Request, error)
	// GetEndpoint returns the API endpoint path
	GetEndpoint() string
	// GetHeaders returns the headers required for the request
	GetHeaders() map[string]string
}

// ChatMessage represents a message in the chat request
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequestBuilder builds requests for the Anthropic Messages API
type AnthropicRequestBuilder struct {
	baseURL   string
	apiKey    string
	authToken string
}

// AnthropicRequest represents the request body for Anthropic Messages API
type AnthropicRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []ChatMessage `json:"messages"`
	Stream    bool          `json:"stream,omitempty"`
}

// GetEndpoint returns the Anthropic Messages API endpoint
func (b *AnthropicRequestBuilder) GetEndpoint() string {
	return "/v1/messages"
}


// GetHeaders returns the headers required for Anthropic API requests
func (b *AnthropicRequestBuilder) GetHeaders() map[string]string {
	headers := map[string]string{
		"Content-Type":      "application/json",
		"anthropic-version": "2023-06-01",
	}

	if b.apiKey != "" {
		headers["x-api-key"] = b.apiKey
	}
	if b.authToken != "" {
		headers["Authorization"] = "Bearer " + b.authToken
	}

	return headers
}

// BuildChatRequest builds a chat completion request for Anthropic Messages API
func (b *AnthropicRequestBuilder) BuildChatRequest(model string, streaming bool) (*http.Request, error) {
	reqBody := AnthropicRequest{
		Model:     model,
		MaxTokens: 100,
		Messages: []ChatMessage{
			{Role: "user", Content: "ping"},
		},
	}

	if streaming {
		reqBody.Stream = true
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := strings.TrimSuffix(b.baseURL, "/") + b.GetEndpoint()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range b.GetHeaders() {
		req.Header.Set(key, value)
	}

	return req, nil
}

// OpenAIRequestBuilder builds requests for the OpenAI Chat Completions API
type OpenAIRequestBuilder struct {
	baseURL string
	apiKey  string
}

// OpenAIRequest represents the request body for OpenAI Chat Completions API
type OpenAIRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []ChatMessage `json:"messages"`
	Stream    bool          `json:"stream,omitempty"`
}

// GetEndpoint returns the OpenAI Chat Completions API endpoint
func (b *OpenAIRequestBuilder) GetEndpoint() string {
	return "/v1/chat/completions"
}

// GetHeaders returns the headers required for OpenAI API requests
func (b *OpenAIRequestBuilder) GetHeaders() map[string]string {
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + b.apiKey,
	}
}

// BuildChatRequest builds a chat completion request for OpenAI Chat Completions API
func (b *OpenAIRequestBuilder) BuildChatRequest(model string, streaming bool) (*http.Request, error) {
	reqBody := OpenAIRequest{
		Model:     model,
		MaxTokens: 100,
		Messages: []ChatMessage{
			{Role: "user", Content: "ping"},
		},
	}

	if streaming {
		reqBody.Stream = true
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := strings.TrimSuffix(b.baseURL, "/") + b.GetEndpoint()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range b.GetHeaders() {
		req.Header.Set(key, value)
	}

	return req, nil
}

// NewRequestBuilder creates a new RequestBuilder based on the provider type
func NewRequestBuilder(cfg *config.APIConfig, provider providers.Provider) RequestBuilder {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = provider.DefaultBaseURL()
	}

	switch provider.Name() {
	case "anthropic":
		return &AnthropicRequestBuilder{
			baseURL:   baseURL,
			apiKey:    cfg.APIKey,
			authToken: cfg.AuthToken,
		}
	case "openai":
		return &OpenAIRequestBuilder{
			baseURL: baseURL,
			apiKey:  cfg.APIKey,
		}
	default:
		// Default to OpenAI-compatible format for unknown providers
		return &OpenAIRequestBuilder{
			baseURL: baseURL,
			apiKey:  cfg.APIKey,
		}
	}
}

// NewRequestBuilderWithCustomPath creates a RequestBuilder with a custom endpoint path
func NewRequestBuilderWithCustomPath(cfg *config.APIConfig, provider providers.Provider, customPath string) RequestBuilder {
	builder := NewRequestBuilder(cfg, provider)

	// Wrap the builder to use custom path
	if customPath != "" {
		return &customPathBuilder{
			RequestBuilder: builder,
			customPath:     customPath,
		}
	}

	return builder
}

// customPathBuilder wraps a RequestBuilder to use a custom endpoint path
type customPathBuilder struct {
	RequestBuilder
	customPath string
}

// GetEndpoint returns the custom endpoint path
func (b *customPathBuilder) GetEndpoint() string {
	return b.customPath
}

// BuildChatRequest builds a request using the custom path
func (b *customPathBuilder) BuildChatRequest(model string, streaming bool) (*http.Request, error) {
	req, err := b.RequestBuilder.BuildChatRequest(model, streaming)
	if err != nil {
		return nil, err
	}

	// Replace the URL path with custom path
	originalURL := req.URL.String()
	baseURL := strings.TrimSuffix(originalURL, b.RequestBuilder.GetEndpoint())
	newURL := strings.TrimSuffix(baseURL, "/") + b.customPath

	newReq, err := http.NewRequest(req.Method, newURL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request with custom path: %w", err)
	}

	// Copy headers
	for key, values := range req.Header {
		for _, value := range values {
			newReq.Header.Add(key, value)
		}
	}

	return newReq, nil
}
