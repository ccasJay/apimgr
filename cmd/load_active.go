package cmd

import (
	"fmt"
	"os"

	"apimgr/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loadActiveCmd)
}

var loadActiveCmd = &cobra.Command{
	Use:   "load-active",
	Short: "Load environment variables for active configuration",
	Long:  "Output export commands for active configuration's environment variables. Use in shell initialization scripts: eval \"$(apimgr load-active)\"",
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()

		// Get the active configuration
		apiConfig, err := configManager.GetActive()
		if err != nil {
			// If no active config, silently exit
			os.Exit(0)
		}

		// Export environment variables
		if apiConfig.APIKey != "" {
			fmt.Printf("export ANTHROPIC_API_KEY=\"%s\"\n", apiConfig.APIKey)
		}
		if apiConfig.AuthToken != "" {
			fmt.Printf("export ANTHROPIC_AUTH_TOKEN=\"%s\"\n", apiConfig.AuthToken)
		}
		if apiConfig.BaseURL != "" {
			fmt.Printf("export ANTHROPIC_BASE_URL=\"%s\"\n", apiConfig.BaseURL)
		}
		if apiConfig.Model != "" {
			fmt.Printf("export ANTHROPIC_MODEL=\"%s\"\n", apiConfig.Model)
		}
		fmt.Printf("export APIMGR_ACTIVE=\"%s\"\n", apiConfig.Alias)
	},
}
