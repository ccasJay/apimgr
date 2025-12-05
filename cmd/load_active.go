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
	Short: "Load global active configuration (for shell initialization)",
	Long:  "This command is used in shell initialization scripts to load the global active configuration and restore Claude Code settings if needed. Use: eval \"$(apimgr load-active)\"",
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()

		// Check for active local sessions and clean up stale ones
		// This also restores Claude Code to global config if there are active sessions
		hasActiveSessions, err := configManager.HasActiveLocalSessions()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to check for active sessions: %v\n", err)
		}

		// If there are active local sessions in other terminals, restore Claude Code to global config
		// This ensures new shells use the global configuration, not a local one from another terminal
		if hasActiveSessions {
			if err := configManager.RestoreClaudeToGlobal(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to restore Claude Code to global: %v\n", err)
			}
		}

		// Get the global active configuration
		apiConfig, err := configManager.GetActive()
		if err != nil {
			// If no active config, output unset commands to clear any stale env vars
			fmt.Println("unset ANTHROPIC_API_KEY")
			fmt.Println("unset ANTHROPIC_AUTH_TOKEN")
			fmt.Println("unset ANTHROPIC_BASE_URL")
			fmt.Println("unset ANTHROPIC_MODEL")
			fmt.Println("unset APIMGR_ACTIVE")
			return
		}

		// Output unset commands first to clear any stale env vars
		fmt.Println("unset ANTHROPIC_API_KEY")
		fmt.Println("unset ANTHROPIC_AUTH_TOKEN")
		fmt.Println("unset ANTHROPIC_BASE_URL")
		fmt.Println("unset ANTHROPIC_MODEL")
		fmt.Println("unset APIMGR_ACTIVE")

		// Export environment variables for the global active configuration
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
