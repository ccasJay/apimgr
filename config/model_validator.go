package config

import (
	"fmt"
	"strings"
)

// ModelValidator validates model operations
type ModelValidator struct{}

// NewModelValidator creates a new ModelValidator instance
func NewModelValidator() *ModelValidator {
	return &ModelValidator{}
}

// ValidateModelInList checks if a model is in the supported list.
// Returns an error if the model is not found in the list.
func (v *ModelValidator) ValidateModelInList(model string, models []string) error {
	if model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// Normalize the model name for comparison
	normalizedModel := strings.TrimSpace(model)

	for _, m := range models {
		if strings.TrimSpace(m) == normalizedModel {
			return nil
		}
	}

	return fmt.Errorf("model '%s' is not in supported models list: %v", model, models)
}

// ValidateModelsList checks if the models list is valid.
// A valid models list must be non-empty and contain at least one non-empty model name.
func (v *ModelValidator) ValidateModelsList(models []string) error {
	if len(models) == 0 {
		return fmt.Errorf("models list cannot be empty")
	}

	// Check if there's at least one non-empty model after trimming
	hasValidModel := false
	for _, m := range models {
		if strings.TrimSpace(m) != "" {
			hasValidModel = true
			break
		}
	}

	if !hasValidModel {
		return fmt.Errorf("models list cannot be empty")
	}

	return nil
}

// NormalizeModels normalizes and deduplicates the models list.
// It trims whitespace from each model name and removes duplicates while preserving order.
// Empty model names are removed.
func (v *ModelValidator) NormalizeModels(models []string) []string {
	if len(models) == 0 {
		return []string{}
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(models))

	for _, m := range models {
		trimmed := strings.TrimSpace(m)
		// Skip empty strings and duplicates
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}

	return result
}
