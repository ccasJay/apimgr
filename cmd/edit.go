package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"apimgr/config"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

// EditFlags represents the command line flags for edit command
type EditFlags struct {
	Alias     string
	APIKey    string
	AuthToken string
	BaseURL   string
	Model     string
}

func init() {
	rootCmd.AddCommand(editCmd)

	// Define flags for non-interactive editing
	editCmd.Flags().String("alias", "", "Change config alias")
	editCmd.Flags().String("sk", "", "Change API key")
	editCmd.Flags().String("ak", "", "Change auth token")
	editCmd.Flags().String("url", "", "Change base URL")
	editCmd.Flags().String("model", "", "Change model name")
}

var editCmd = &cobra.Command{
	Use:   "edit <alias>",
	Short: "Edit configuration",
	Long: `Edit a saved API configuration

By default, this command will guide you through editing the various fields of the configuration in an interactive interface.
If command line arguments are provided, the changes will be applied directly without entering interactive mode.

Examples:
  # Interactive edit
  apimgr edit myconfig

  # Non-interactive edit (change API key)
  apimgr edit myconfig --sk sk-ant-api03-xxx

  # Non-interactive edit multiple fields
  apimgr edit myconfig --url https://api.anthropic.com --model claude-3-opus-20240229`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]

		// Check if any flags are provided for non-interactive editing
		aliasFlag, _ := cmd.Flags().GetString("alias")
		skFlag, _ := cmd.Flags().GetString("sk")
		akFlag, _ := cmd.Flags().GetString("ak")
		urlFlag, _ := cmd.Flags().GetString("url")
		modelFlag, _ := cmd.Flags().GetString("model")

		// Parse flags into updates map
		updates := make(map[string]string)
		if aliasFlag != "" {
			updates["alias"] = aliasFlag
		}
		if skFlag != "" {
			updates["api_key"] = skFlag
		}
		if akFlag != "" {
			updates["auth_token"] = akFlag
		}
		if urlFlag != "" {
			updates["base_url"] = urlFlag
		}
		if modelFlag != "" {
			updates["model"] = modelFlag
		}

		configManager := config.NewConfigManager()

		if len(updates) > 0 {
			// Non-interactive mode: directly apply changes
			if err := saveAndApplyChanges(configManager, alias, updates); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Show success message with updated alias
			updatedAlias := alias
			if newAlias, ok := updates["alias"]; ok {
				updatedAlias = newAlias
			}
			fmt.Printf("✅ Configuration '%s' updated\n", updatedAlias)
		} else {
			// Interactive mode: guide user through editing
			if err := editConfig(alias); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// FieldType represents the type of field being edited
type FieldType int

const (
	// FieldAlias represents the alias field
	FieldAlias FieldType = iota
	// FieldAPIKey represents the API key field
	FieldAPIKey
	// FieldAuthToken represents the auth token field
	FieldAuthToken
	// FieldBaseURL represents the base URL field
	FieldBaseURL
	// FieldModel represents the model field
	FieldModel
)

func editConfig(alias string) error {
	configManager := config.NewConfigManager()

	// Get the current configuration
	currentConfig, err := configManager.Get(alias)
	if err != nil {
		return fmt.Errorf("Failed to get configuration: %v", err)
	}

	// Display current configuration
	displayConfig(*currentConfig)

	// Collect user edits interactively
	updates, err := collectUserEdits(currentConfig, configManager)
	if err != nil {
		return err
	}

	// Apply and save changes
	if err := saveAndApplyChanges(configManager, alias, updates); err != nil {
		return err
	}

	updatedAlias := getUpdatedAlias(alias, updates)
	fmt.Printf("\n✅ Configuration '%s' updated\n", updatedAlias)
	return nil
}

// collectUserEdits handles the interactive editing loop and returns collected updates
func collectUserEdits(currentConfig *config.APIConfig, configManager *config.Manager) (map[string]string, error) {
	reader := bufio.NewReader(os.Stdin)
	updates := make(map[string]string)

	for {
		showMenu(len(updates))
		choice := getUserChoice(reader)

		// Handle special cases: quit, preview, or save
		if shouldQuit(choice) {
			fmt.Println("\nEdit cancelled, no changes saved")
			return nil, fmt.Errorf("Operation cancelled")
		}

		if shouldPreview(choice) {
			if err := handlePreview(currentConfig, updates); err != nil {
				fmt.Printf("\n❌ %v\n", err)
			}
			continue
		}

		if shouldSave(choice) {
			if len(updates) == 0 {
				fmt.Println("\nNo changes, skipping save")
				return nil, fmt.Errorf("No changes")
			}
			if !confirmSave(reader) {
				fmt.Println("\nSave cancelled")
				return nil, fmt.Errorf("Save cancelled")
			}
			break
		}

		// Process field selection
		if err := handleFieldSelection(reader, currentConfig, updates, choice, configManager); err != nil {
			fmt.Printf("\n❌ %v\n", err)
		}
	}

	return updates, nil
}

// shouldQuit checks if user wants to quit without saving
func shouldQuit(choice string) bool {
	return choice == "q" || choice == "Q"
}

// shouldPreview checks if user wants to preview changes
func shouldPreview(choice string) bool {
	return choice == "p" || choice == "P"
}

// shouldSave checks if user wants to save changes
func shouldSave(choice string) bool {
	return choice == "0"
}

func showMenu(updateCount int) {
	fmt.Println("\n" + strings.Repeat("-", 60))
	if updateCount > 0 {
		fmt.Printf("%d fields changed\n", updateCount)
	}
	fmt.Println("Please select a field to modify (enter number):")
	fmt.Println("1. Alias (alias)")
	fmt.Println("2. API key (api_key)")
	fmt.Println("3. Auth token (auth_token)")
	fmt.Println("4. Base URL (base_url)")
	fmt.Println("5. Model name (model)")
	fmt.Println("p. Preview changes")
	fmt.Println("0. Complete edit and save")
	fmt.Println("q. Exit without saving")
	fmt.Println(strings.Repeat("-", 60))
}

func getUserChoice(reader *bufio.Reader) string {
	fmt.Print("\nEnter your choice: ")
	choice, _ := reader.ReadString('\n')
	return strings.TrimSpace(choice)
}

func displayConfig(config config.APIConfig) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("Current configuration: %s\n", config.Alias)
	fmt.Println(strings.Repeat("=", 60))

	// Use helper function to display field
	displayField("1. Alias", config.Alias, "")
	displayMaskedField("2. API key", config.APIKey, utils.MaskAPIKey(config.APIKey))
	displayMaskedField("3. Auth token", config.AuthToken, utils.MaskAPIKey(config.AuthToken))
	displayField("4. Base URL", config.BaseURL, "https://api.anthropic.com (default)")
	displayField("5. Model name", config.Model, "(not set)")

	fmt.Println(strings.Repeat("=", 60))
}

func displayField(label, value, defaultValue string) {
	if value != "" {
		fmt.Printf("%s: %s\n", label, value)
	} else {
		fmt.Printf("%s: %s\n", label, defaultValue)
	}
}

func displayMaskedField(label, value, maskedValue string) {
	if value != "" {
		fmt.Printf("%s: %s\n", label, maskedValue)
	} else {
		fmt.Println(label + ": (not set)")
	}
}

func handleFieldSelection(reader *bufio.Reader, currentConfig *config.APIConfig, updates map[string]string, choice string, configManager *config.Manager) error {
	var fieldType FieldType
	var fieldName string

	// Validate choice and get field info
	if err := parseFieldChoice(choice, &fieldType, &fieldName); err != nil {
		return err
	}

	return editField(reader, currentConfig, updates, fieldType, fieldName, configManager)
}

// parseFieldChoice parses user choice and returns field information
func parseFieldChoice(choice string, fieldType *FieldType, fieldName *string) error {
	switch choice {
	case "1":
		*fieldType = FieldAlias
		*fieldName = "Alias"
	case "2":
		*fieldType = FieldAPIKey
		*fieldName = "API Key"
	case "3":
		*fieldType = FieldAuthToken
		*fieldName = "Authentication Token"
	case "4":
		*fieldType = FieldBaseURL
		*fieldName = "Base URL"
	case "5":
		*fieldType = FieldModel
		*fieldName = "Model Name"
	default:
		return fmt.Errorf("Invalid choice, please enter 0-5, p, or q")
	}
	return nil
}

// handlePreview displays preview of changes if any
func handlePreview(currentConfig *config.APIConfig, updates map[string]string) error {
	if len(updates) == 0 {
		return fmt.Errorf("No changes yet")
	}
	previewChanges(*currentConfig, updates)
	return nil
}

func editField(reader *bufio.Reader, currentConfig *config.APIConfig, updates map[string]string, fieldType FieldType, fieldName string, configManager *config.Manager) error {
	// Get current value (either from updates or currentConfig)
	currentValue := getCurrentValue(currentConfig, updates, fieldType)
	prompt := fmt.Sprintf("\nCurrent %s: %s\nEnter new %s (press Enter to keep unchanged): ", fieldName, currentValue, fieldName)
	fmt.Print(prompt)

	newValue, _ := reader.ReadString('\n')
	newValue = strings.TrimSpace(newValue)

	// No change
	if newValue == "" {
		fmt.Println("No change")
		return nil
	}

	// Validate the new value
	if err := validateFieldValue(fieldType, newValue, currentConfig, configManager); err != nil {
		return err
	}

	// Store the update
	updateKey := getFieldKey(fieldType)
	updates[updateKey] = newValue

	// Show success message with masked value if sensitive
	if isSensitiveField(fieldType) {
		fmt.Printf("✓ %s will be updated to: %s\n", fieldName, utils.MaskAPIKey(newValue))
	} else {
		fmt.Printf("✓ %s will be updated to: %s\n", fieldName, newValue)
	}

	return nil
}

func getCurrentValue(config *config.APIConfig, updates map[string]string, fieldType FieldType) string {
	key := getFieldKey(fieldType)
	if val, ok := updates[key]; ok {
		return val
	}

	switch fieldType {
	case FieldAlias:
		return config.Alias
	case FieldAPIKey:
		return config.APIKey
	case FieldAuthToken:
		return config.AuthToken
	case FieldBaseURL:
		return config.BaseURL
	case FieldModel:
		return config.Model
	default:
		return ""
	}
}

func getFieldKey(fieldType FieldType) string {
	switch fieldType {
	case FieldAlias:
		return "alias"
	case FieldAPIKey:
		return "api_key"
	case FieldAuthToken:
		return "auth_token"
	case FieldBaseURL:
		return "base_url"
	case FieldModel:
		return "model"
	default:
		return ""
	}
}

func isSensitiveField(fieldType FieldType) bool {
	return fieldType == FieldAPIKey || fieldType == FieldAuthToken
}

func validateFieldValue(fieldType FieldType, value string, currentConfig *config.APIConfig, configManager *config.Manager) error {
	switch fieldType {
	case FieldAlias:
		// Check if alias already exists (excluding current config)
		if value == currentConfig.Alias {
			return fmt.Errorf("New alias is the same as current alias")
		}
		if _, err := configManager.Get(value); err == nil {
			return fmt.Errorf("Alias '%s' already exists", value)
		}
	case FieldBaseURL:
		// Validate URL format
		if _, err := url.ParseRequestURI(value); err != nil {
			return fmt.Errorf("Invalid URL format: %v", err)
		}
	case FieldAPIKey, FieldAuthToken:
		// Validate that at least one auth method is set
		otherAuth := getOtherAuthValue(fieldType, currentConfig, value)
		if otherAuth == "" && value == "" {
			return fmt.Errorf("API key and auth token cannot both be empty")
		}
	}
	return nil
}

func getOtherAuthValue(fieldType FieldType, config *config.APIConfig, newValue string) string {
	if fieldType == FieldAPIKey {
		// After update, api_key will be newValue, check if auth_token is set
		return config.AuthToken
	}
	// After update, auth_token will be newValue, check if api_key is set
	return config.APIKey
}

func previewChanges(currentConfig config.APIConfig, updates map[string]string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Preview changes:")
	fmt.Println(strings.Repeat("=", 60))

	// Show each changed field
	if newAlias, ok := updates["alias"]; ok {
		fmt.Printf("Alias: %s → %s\n", currentConfig.Alias, newAlias)
	}
	if newAPIKey, ok := updates["api_key"]; ok {
		fmt.Printf("API Key: %s → %s\n", utils.MaskAPIKey(currentConfig.APIKey), utils.MaskAPIKey(newAPIKey))
	}
	if newAuthToken, ok := updates["auth_token"]; ok {
		fmt.Printf("Authentication Token: %s → %s\n", utils.MaskAPIKey(currentConfig.AuthToken), utils.MaskAPIKey(newAuthToken))
	}
	if newBaseURL, ok := updates["base_url"]; ok {
		fmt.Printf("Base URL: %s → %s\n", currentConfig.BaseURL, newBaseURL)
	}
	if newModel, ok := updates["model"]; ok {
		fmt.Printf("Model Name: %s → %s\n", currentConfig.Model, newModel)
	}

	fmt.Println(strings.Repeat("=", 60))
}

func confirmSave(reader *bufio.Reader) bool {
	fmt.Print("\nConfirm saving changes? (y/N): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	return choice == "y" || choice == "Y"
}

func getUpdatedAlias(originalAlias string, updates map[string]string) string {
	if newAlias, ok := updates["alias"]; ok {
		return newAlias
	}
	return originalAlias
}

// saveAndApplyChanges applies updates to config and regenerates the active script
func saveAndApplyChanges(configManager *config.Manager, alias string, updates map[string]string) error {
	// Apply field updates
	if err := applyUpdates(configManager, alias, updates); err != nil {
		return fmt.Errorf("Save failed: %v", err)
	}

	// Generate active.env script
	if err := configManager.GenerateActiveScript(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to generate activation script: %v\n", err)
	}

	return nil
}

func applyUpdates(configManager *config.Manager, alias string, updates map[string]string) error {
	// Handle alias update separately
	if newAlias, ok := updates["alias"]; ok {
		if err := configManager.RenameAlias(alias, newAlias); err != nil {
			return fmt.Errorf("Failed to rename alias: %v", err)
		}
		alias = newAlias // Update alias for subsequent updates
	}

	// Remove alias from updates
	delete(updates, "alias")

	// Apply other field updates
	if len(updates) > 0 {
		if err := configManager.UpdatePartial(alias, updates); err != nil {
			return err
		}
	}

	return nil
}
