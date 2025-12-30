package cmd

import (
	"fmt"
	"os"
	"strings"

	"apimgr/config"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show currently active configuration",
	Long:  "Show currently active API configuration information, including global configuration and current shell environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get shell environment variables
		shellAPIKey := os.Getenv("ANTHROPIC_API_KEY")
		shellAuthToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
		shellAPIBase := os.Getenv("ANTHROPIC_BASE_URL")
		shellModel := os.Getenv("ANTHROPIC_MODEL")
		shellActiveAlias := os.Getenv("APIMGR_ACTIVE")

		// Get global configuration
		configManager, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}
		globalActiveConfig, globalErr := configManager.GetActive()
		var globalActiveAlias string
		if globalErr == nil {
			globalActiveAlias = globalActiveConfig.Alias
		}

		fmt.Println("Current configuration status:")
		fmt.Println("=========================================")

		// Show global active configuration
		fmt.Println("1. Global active configuration (config file):")
		if globalErr != nil {
			fmt.Println("   No global active configuration set")
		} else {
			fmt.Printf("   Alias: %s\n", globalActiveConfig.Alias)
			if globalActiveConfig.APIKey != "" {
				fmt.Printf("   API Key: %s\n", utils.MaskAPIKey(globalActiveConfig.APIKey))
			}
			if globalActiveConfig.AuthToken != "" {
				fmt.Printf("   Auth Token: %s\n", utils.MaskAPIKey(globalActiveConfig.AuthToken))
			}
			if globalActiveConfig.BaseURL != "" {
				fmt.Printf("   Base URL: %s\n", globalActiveConfig.BaseURL)
			}
			// Show active model
			if globalActiveConfig.Model != "" {
				fmt.Printf("   Active Model: %s\n", globalActiveConfig.Model)
			}
			// Show all supported models (Requirements: 3.2, 3.3)
			if len(globalActiveConfig.Models) > 0 {
				fmt.Printf("   Supported Models: %s\n", formatModelsListForStatus(globalActiveConfig.Models, globalActiveConfig.Model))
			}
		}

		// Show shell environment configuration
		fmt.Println("\n2. Current Shell environment:")
		if shellAPIKey == "" && shellAuthToken == "" {
			fmt.Println("   No environment variables set")
		} else {
			if shellActiveAlias != "" {
				fmt.Printf("   Alias: %s\n", shellActiveAlias)
			}
			if shellAPIKey != "" {
				fmt.Printf("   API Key: %s\n", utils.MaskAPIKey(shellAPIKey))
			}
			if shellAuthToken != "" {
				fmt.Printf("   Auth Token: %s\n", utils.MaskAPIKey(shellAuthToken))
			}
			if shellAPIBase != "" {
				fmt.Printf("   Base URL: %s\n", shellAPIBase)
			}
			if shellModel != "" {
				fmt.Printf("   Model: %s\n", shellModel)
			}
		}

		// Show configuration source
		fmt.Println("\n=========================================")
		if shellAPIKey != "" || shellAuthToken != "" {
			if globalErr != nil || (globalActiveAlias != "" && globalActiveAlias != shellActiveAlias) {
				fmt.Println("💡 Currently using Shell environment configuration (overrides global configuration)")
			} else {
				fmt.Println("💡 Currently using global configuration")
			}
		} else {
			if globalErr != nil {
				fmt.Println("💡 No configuration set")
			} else {
				fmt.Println("💡 Currently using global configuration (Shell has no environment variables set)")
			}
		}

		fmt.Println("\n💡 Tip: Run 'apimgr install' to install shell integration for better experience")
		return nil
	},
}

// formatModelsListForStatus formats the models list for status display, marking the active model.
// Requirements: 3.2, 3.3
func formatModelsListForStatus(models []string, activeModel string) string {
	if len(models) == 0 {
		return "(none)"
	}

	var parts []string
	for _, model := range models {
		if model == activeModel {
			parts = append(parts, model+" [active]")
		} else {
			parts = append(parts, model)
		}
	}
	return strings.Join(parts, ", ")
}
