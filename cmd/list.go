package cmd

import (
	"fmt"
	"os"

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
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()
		configs, err := configManager.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(configs) == 0 {
			fmt.Println("No configurations available")
			return
		}

		// Get active configuration name
		activeName, _ := configManager.GetActiveName()

		fmt.Println("Available configurations:")
		for _, config := range configs {
			// Display masked API key or auth token
			var authInfo string
			if config.APIKey != "" {
				authInfo = "API Key: " + utils.MaskAPIKey(config.APIKey)
			} else {
				authInfo = "Auth Token: " + utils.MaskAPIKey(config.AuthToken)
			}

			// Mark active configuration with *
			activeMarker := " "
			if config.Alias == activeName {
				activeMarker = "*"
			}

			fmt.Printf("%s %s: %s (URL: %s, Model: %s)\n",
				activeMarker, config.Alias, authInfo, config.BaseURL, config.Model)
		}

		if activeName != "" {
			fmt.Printf("\n* indicates the currently active configuration\n")
		}
	},
}
