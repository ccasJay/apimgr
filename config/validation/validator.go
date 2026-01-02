package validation

import (
	"fmt"
	"apimgr/config/models"
	"apimgr/internal/providers"
	"apimgr/internal/utils"
)

// Validator validates API configurations
type Validator struct {
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateConfig validates a configuration
func (v *Validator) ValidateConfig(config models.APIConfig) error {
	if config.Alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}

	// Default provider is anthropic
	providerName := config.Provider
	if providerName == "" {
		providerName = "anthropic"
	}

	// 至少需要一种认证方式
	if config.APIKey == "" && config.AuthToken == "" {
		return fmt.Errorf("API key and auth token cannot both be empty")
	}

	// Validate provider
	provider, err := providers.Get(providerName)
	if err != nil {
		return fmt.Errorf("unknown API provider: %s", providerName)
	}

	// Provider-specific validation
	if err := provider.ValidateConfig(config.BaseURL, config.APIKey, config.AuthToken); err != nil {
		return err
	}

	// URL format validation
	if config.BaseURL != "" {
		if !utils.ValidateURL(config.BaseURL) {
			return fmt.Errorf("invalid URL format: %s", config.BaseURL)
		}
	}

	return nil
}
