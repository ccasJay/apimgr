package cmd

import (
	"fmt"
	"strings"

	"apimgr/config"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API configurations",
	Long:  "List all saved API configurations",
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}
		configs, err := configManager.List()
		if err != nil {
			return err
		}

		if len(configs) == 0 {
			fmt.Println("No configurations available")
			return nil
		}

		// Get active configuration name
		activeName, _ := configManager.GetActiveName()

		fmt.Println("Available configurations:")
		for _, cfg := range configs {
			// Display masked API key or auth token
			var authInfo string
			if cfg.APIKey != "" {
				authInfo = "API Key: " + utils.MaskAPIKey(cfg.APIKey)
			} else {
				authInfo = "Auth Token: " + utils.MaskAPIKey(cfg.AuthToken)
			}

			// Mark active configuration with *
			activeMarker := " "
			if cfg.Alias == activeName {
				activeMarker = "*"
			}

			// Format models display with active model marker
			modelsDisplay := formatModelsDisplay(cfg.Models, cfg.Model)

			fmt.Printf("%s %s: %s (URL: %s, Models: %s)\n",
				activeMarker, cfg.Alias, authInfo, cfg.BaseURL, modelsDisplay)
		}

		if activeName != "" {
			fmt.Printf("\n* indicates the currently active configuration\n")
		}
		fmt.Printf("[active] indicates the currently active model within a configuration\n")
		return nil
	},
}

// formatModelsDisplay formats the models list for display, marking the active model.
// Requirements: 3.1, 3.3
func formatModelsDisplay(models []string, activeModel string) string {
	if len(models) == 0 {
		if activeModel != "" {
			return activeModel + " [active]"
		}
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
