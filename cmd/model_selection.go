package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"apimgr/config/models"
	"apimgr/config/validation"
)

// ModelSelector handles interactive model selection functionality
type ModelSelector struct{}

// NewModelSelector creates a new ModelSelector instance
func NewModelSelector() *ModelSelector {
	return &ModelSelector{}
}

// ShouldPrompt determines whether to prompt the user for model selection
// It returns true if all conditions are met:
// 1. noPrompt is false
// 2. modelFlag is empty (user didn't specify a model)
// 3. config.Models has more than one model
// 4. Stdin is an interactive terminal
func (ms *ModelSelector) ShouldPrompt(cfg *models.APIConfig, modelFlag string, noPrompt bool) bool {
	if noPrompt {
		return false
	}
	if modelFlag != "" {
		return false
	}
	if len(cfg.Models) <= 1 {
		return false
	}
	if !isInteractiveTerminal() {
		return false
	}
	return true
}

// PromptSimple presents a simple numbered list of models to the user and returns their selection
// It displays the models with the current model marked as active
// Users can select by number or press Enter to use the current model
func (ms *ModelSelector) PromptSimple(models []string, currentModel string) (string, error) {
	// Create a reader for terminal input
	reader := bufio.NewReader(os.Stdin)

	// Display the available models
	fmt.Fprintln(os.Stderr, "📋 Available models:")
	for i, model := range models {
		selection := fmt.Sprintf("  %2d. %s", i+1, model)
		if model == currentModel {
			selection = fmt.Sprintf("  ➤ %2d. %s (current)", i+1, model)
		}
		fmt.Fprintln(os.Stderr, selection)
	}

	// Prompt the user
	prompt := fmt.Sprintf("\nSelect model (1-%d) [Enter to use '%s']: ", len(models), currentModel)
	fmt.Fprint(os.Stderr, prompt)

	// Read user input
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}

	// Trim whitespace and newline characters
	input = strings.TrimSpace(strings.TrimSuffix(input, "\n"))

	// If user pressed Enter without input, use current model
	if input == "" {
		return currentModel, nil
	}

	// Parse the input as a number
	selectionIndex, err := strconv.Atoi(input)
	if err != nil {
		return "", fmt.Errorf("invalid input, please enter a number between 1 and %d", len(models))
	}

	// Check if the number is within the valid range
	if selectionIndex < 1 || selectionIndex > len(models) {
		return "", fmt.Errorf("invalid selection, please enter a number between 1 and %d", len(models))
	}

	// Return the selected model
	return models[selectionIndex-1], nil
}

// ValidateModelInList checks if the specified model exists in the provided list of models
func (ms *ModelSelector) ValidateModelInList(model string, models []string) error {
	validator := validation.NewModelValidator()
	return validator.ValidateModelInList(model, models)
}

// isInteractiveTerminal checks if the current Stdin is an interactive terminal
func isInteractiveTerminal() bool {
	// Check for CI/non-interactive environments first
	if isCIEnvironment() {
		return false
	}

	// Check TERM environment variable - should be set and not "dumb"
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	// In Claude Code or interactive environments, we can assume interactive
	// if TERM is reasonable and not in CI environment
	return true
}

// isCIEnvironment checks if we're running in a CI/CD environment
func isCIEnvironment() bool {
	ciVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"BUILD_NUMBER",
		"RUN_ID",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_HOME",
		"TRAVIS",
		"CIRCLECI",
		"TEAMCITY_VERSION",
	}

	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	return false
}
