package providers

import (
	"errors"
	"fmt"
)

// Provider defines the standard interface for API providers
type Provider interface {
	// Name returns the provider's name (e.g., "anthropic", "openai")
	Name() string
	// DefaultBaseURL returns the default base URL for the provider
	DefaultBaseURL() string
	// DefaultModel returns the default model for the provider
	DefaultModel() string
	// ValidateConfig validates the API configuration for this provider
	ValidateConfig(baseURL, apiKey, authToken string) error
	// NormalizeConfig normalizes the API configuration (e.g., add trailing slash to URL)
	NormalizeConfig(baseURL string) string
}

// registry stores all registered providers
var registry = make(map[string]Provider)

// Register registers a new provider
func Register(name string, provider Provider) {
	registry[name] = provider
}

// Get returns a provider by name
func Get(name string) (Provider, error) {
	provider, ok := registry[name]
	if !ok {
		return nil, errors.New("unknown provider: " + name)
	}
	return provider, nil
}

// List returns all registered providers
func List() []string {
	var list []string
	for name := range registry {
		list = append(list, name)
	}
	return list
}

// AnthropicProvider is the Anthropic API provider implementation
type AnthropicProvider struct{}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// DefaultBaseURL returns the default Anthropic API base URL
func (p *AnthropicProvider) DefaultBaseURL() string {
	return "https://api.anthropic.com"
}

// DefaultModel returns the default Anthropic model
func (p *AnthropicProvider) DefaultModel() string {
	return "claude-3-sonnet-20240229"
}

// ValidateConfig validates the Anthropic API configuration
func (p *AnthropicProvider) ValidateConfig(baseURL, apiKey, authToken string) error {
	if apiKey == "" && authToken == "" {
		return fmt.Errorf("anthropic: must provide either API key or auth token")
	}
	return nil
}

// NormalizeConfig normalizes the Anthropic API configuration
func (p *AnthropicProvider) NormalizeConfig(baseURL string) string {
	// Ensure URL ends with trailing slash
	if baseURL != "" && baseURL[len(baseURL)-1] != '/' {
		return baseURL + "/"
	}
	return baseURL
}

// OpenAIProvider is the OpenAI API provider implementation
type OpenAIProvider struct{}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// DefaultBaseURL returns the default OpenAI API base URL
func (p *OpenAIProvider) DefaultBaseURL() string {
	return "https://api.openai.com/v1"
}

// DefaultModel returns the default OpenAI model
func (p *OpenAIProvider) DefaultModel() string {
	return "gpt-4"
}

// ValidateConfig validates the OpenAI API configuration
func (p *OpenAIProvider) ValidateConfig(baseURL, apiKey, authToken string) error {
	if apiKey == "" {
		return fmt.Errorf("openai: must provide API key")
	}
	return nil
}

// NormalizeConfig normalizes the OpenAI API configuration
func (p *OpenAIProvider) NormalizeConfig(baseURL string) string {
	if baseURL != "" && baseURL[len(baseURL)-1] != '/' {
		return baseURL + "/"
	}
	return baseURL
}

// Initialize: register built-in providers
func init() {
	Register("anthropic", &AnthropicProvider{})
	Register("openai", &OpenAIProvider{})
}
