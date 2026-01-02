package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"apimgr/config"
	"apimgr/config/session"
	"apimgr/config/validation"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(switchCmd)
	// Add local switch parameter
	switchCmd.Flags().BoolP("local", "l", false, "Only take effect in current shell, does not modify global configuration")
	// Add model switch parameter
	switchCmd.Flags().StringP("model", "m", "", "Switch to a specific model within the configuration")
	// Add no-prompt parameter for non-interactive use
	switchCmd.Flags().Bool("no-prompt", false, "Disable interactive model selection even when multiple models are available")
}

var switchCmd = &cobra.Command{
	Use:   "switch [alias]",
	Short: "Switch to specified API configuration",
	Long: `Switch to specified API configuration and output export commands for environment variables

To make environment variables effective in current shell, there are two methods:
1. Using eval: eval "$(apimgr switch <alias>)"
2. Install shell integration: apimgr install (recommended, allows direct use of apimgr switch after installation)

Using -l/--local parameter switches configuration only in current shell session without modifying global configuration:
  apimgr switch -l <alias>
  eval "$(apimgr switch -l <alias>)"

Using -m/--model parameter switches to a specific model within the configuration:
  apimgr switch <alias> --model claude-3-sonnet
  eval "$(apimgr switch <alias> -m gpt-4)"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		// Read the local flag
		local, _ := cmd.Flags().GetBool("local")
		// Read the model flag
		modelFlag, _ := cmd.Flags().GetString("model")

		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

		configManager, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}

		// Get the configuration first (needed for both modes)
		apiConfig, err := configManager.Get(alias)
		if err != nil {
			return err
		}

		// Handle model switch if --model flag is provided
		if modelFlag != "" {
			// Validate model is in supported list
			validator := validation.NewModelValidator()
			if err := validator.ValidateModelInList(modelFlag, apiConfig.Models); err != nil {
				return err
			}

			// Switch the model in the configuration
			if err := configManager.SwitchModel(alias, modelFlag); err != nil {
				return err
			}

			// Refresh the config to get the updated model
			apiConfig, err = configManager.Get(alias)
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, successStyle.Render(fmt.Sprintf("✓ Switched model to: %s", modelFlag)))
		} else {
			// Check if we need to prompt for model selection
			noPrompt, _ := cmd.Flags().GetBool("no-prompt")
			modelSelector := NewModelSelector()

			if modelSelector.ShouldPrompt(apiConfig, modelFlag, noPrompt) {
				// Prompt user for model selection
				selectedModel, err := modelSelector.PromptSimple(apiConfig.Models, apiConfig.Model)
				if err != nil {
					return fmt.Errorf("model selection failed: %w", err)
				}

				// If user selected a different model, update it
				if selectedModel != apiConfig.Model {
					// Switch the model in the configuration
					if err := configManager.SwitchModel(alias, selectedModel); err != nil {
						return err
					}

					// Refresh the config to get the updated model
					apiConfig, err = configManager.Get(alias)
					if err != nil {
						return err
					}

					fmt.Fprintln(os.Stderr, successStyle.Render(fmt.Sprintf("✓ Switched model to: %s", selectedModel)))
				}
			}
		}

		if local {
			// Local mode: update Claude Code but not global active
			pid := fmt.Sprintf("%d", os.Getpid())

			// Create session marker
			if err := session.CreateSessionMarker(configManager.GetConfigPath(), pid, alias); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to create session marker: %v\n", err)
			}

			// Sync to Claude Code only (no global active update)
			if err := configManager.SyncClaudeSettingsOnly(apiConfig); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to sync to Claude Code: %v\n", err)
			}

			// Output trap command for cleanup on shell exit
			fmt.Printf("trap 'apimgr cleanup-session %s' EXIT\n", pid)
		} else {
			// Global mode: update global configuration
			// Set the active configuration
			if err := configManager.SetActive(alias); err != nil {
				return err
			}

			// Generate active.env script for auto-loading
			if err := configManager.GenerateActiveScript(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to generate activation script: %v\n", err)
			}

			// Show sync information
			showSyncInfo(alias)
		}

		// Clear previous environment variables
		fmt.Println("unset ANTHROPIC_API_KEY")
		fmt.Println("unset ANTHROPIC_AUTH_TOKEN")
		fmt.Println("unset ANTHROPIC_BASE_URL")
		fmt.Println("unset ANTHROPIC_MODEL")
		fmt.Println("unset APIMGR_ACTIVE")

		// Export new environment variables
		if apiConfig.APIKey != "" {
			fmt.Printf("export ANTHROPIC_API_KEY=\"%s\"\n", apiConfig.APIKey)
		} else if apiConfig.AuthToken != "" {
			fmt.Printf("export ANTHROPIC_AUTH_TOKEN=\"%s\"\n", apiConfig.AuthToken)
		}
		if apiConfig.BaseURL != "" {
			fmt.Printf("export ANTHROPIC_BASE_URL=\"%s\"\n", apiConfig.BaseURL)
		}
		if apiConfig.Model != "" {
			fmt.Printf("export ANTHROPIC_MODEL=\"%s\"\n", apiConfig.Model)
		}
		fmt.Printf("export APIMGR_ACTIVE=\"%s\"\n", alias)

		if local {
			fmt.Fprintln(os.Stderr, successStyle.Render(fmt.Sprintf("✓ Switched to configuration locally: %s", alias)))
		} else {
			if modelFlag == "" {
				fmt.Fprintln(os.Stderr, successStyle.Render(fmt.Sprintf("✓ Switched to configuration: %s", alias)))
			}
		}
		return nil
	},
}

// showSyncInfo shows sync status information
func showSyncInfo(alias string) {
	// Check sync status
	globalClaudePath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")
	projectClaudePath := filepath.Join(".", ".claude", "settings.json")

	hasGlobal := false
	hasProject := false

	if _, err := os.Stat(globalClaudePath); err == nil {
		hasGlobal = true
	}
	if _, err := os.Stat(projectClaudePath); err == nil {
		hasProject = true
	}

	if hasGlobal || hasProject {
		fmt.Fprintf(os.Stderr, "\n✅ Configuration sync status:\n")
		if hasGlobal {
			fmt.Fprintf(os.Stderr, "   • Global Claude Code: ~/.claude/settings.json\n")
		}
		if hasProject {
			fmt.Fprintf(os.Stderr, "   • Project-level Claude Code: %s\n", projectClaudePath)
		}
		fmt.Fprintf(os.Stderr, "\n💡 Configuration has been automatically synced to Claude Code, ready to use.\n")
	}
}
