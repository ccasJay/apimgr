package cmd

import (
	"fmt"
	"os"

	"github.com/ccasJay/apimgr/config"
	"github.com/ccasJay/apimgr/config/session"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loadActiveCmd)
}

var loadActiveCmd = &cobra.Command{
	Use:   "load-active",
	Short: "Load global active configuration (for shell initialization)",
	Long:  "This command is used in shell initialization scripts to load the global active configuration and restore Claude Code settings if needed. Use: eval \"$(apimgr load-active)\"",
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}

		// Check for active local sessions and clean up stale ones
		// This also restores Claude Code to global config if there are active sessions
		hasActiveSessions, err := session.HasActiveLocalSessions(configManager.GetConfigPath())
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
			writeShellUnsets(os.Stdout)
			return nil
		}

		// Output unset commands first to clear any stale env vars
		writeShellUnsets(os.Stdout)
		writeShellExports(os.Stdout, apiConfig, apiConfig.Alias)
		return nil
	},
}
