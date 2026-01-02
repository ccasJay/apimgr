package validation

import (
	"fmt"
	"strings"

	"apimgr/internal/utils"
)

// InputValidator validates user input
type InputValidator struct {
}

// NewInputValidator creates a new InputValidator
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// ValidateAlias checks if an alias is valid
func (iv *InputValidator) ValidateAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if strings.ContainsAny(alias, "<>\"'&/\\") {
		return fmt.Errorf("alias contains invalid characters")
	}
	if len(alias) > 50 {
		return fmt.Errorf("alias is too long (max 50 characters)")
	}
	return nil
}

// ValidateURL checks if a URL is valid
func (iv *InputValidator) ValidateURL(url string) error {
	if url != "" && !utils.ValidateURL(url) {
		return fmt.Errorf("invalid URL format")
	}
	return nil
}

// ValidateModelName checks if a model name is valid
func (iv *InputValidator) ValidateModelName(model string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	if strings.ContainsAny(model, "<>\"'&/\\") {
		return fmt.Errorf("model name contains invalid characters")
	}
	return nil
}
