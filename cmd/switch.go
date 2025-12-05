package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"apimgr/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(switchCmd)
	// Add local switch parameter
	switchCmd.Flags().BoolP("local", "l", false, "Only take effect in current shell, does not modify global configuration")
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
  eval "$(apimgr switch -l <alias>)"`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]

		// Read the local flag
		local, _ := cmd.Flags().GetBool("local")

		configManager := config.NewConfigManager()

		// Get the configuration first (needed for both modes)
		apiConfig, err := configManager.Get(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if local {
			// Local mode: update Claude Code but not global active
			pid := fmt.Sprintf("%d", os.Getpid())

			// Create session marker
			if err := configManager.CreateSessionMarker(pid, alias); err != nil {
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
			err := configManager.SetActive(alias)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
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
		fmt.Printf("export APIMGR_ACTIVE=\"%s\"\n", alias)

		// Print success message to stderr so it doesn't interfere with eval
		if local {
			fmt.Fprintf(os.Stderr, "✓ Switched to configuration locally: %s\n", alias)
		} else {
			fmt.Fprintf(os.Stderr, "✓ Switched to configuration: %s\n", alias)
		}
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
