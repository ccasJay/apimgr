package cmd

import (
	"fmt"
	"os"

	"apimgr/config"
	"apimgr/config/session"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cleanupSessionCmd)
}

var cleanupSessionCmd = &cobra.Command{
	Use:   "cleanup-session [pid]",
	Short: "Cleanup local session marker (internal use)",
	Long: `This command is used internally by the shell trap mechanism to cleanup session markers when a shell exits.

It is automatically called by the trap command output by 'apimgr switch -l'.
Users typically do not need to call this command directly.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pid := args[0]
		configManager, err := config.NewConfigManager()
		if err != nil {
			// Silently exit - this is called during shell exit
			os.Exit(0)
		}

		if err := session.CleanupSession(configManager.GetConfigPath(), pid); err != nil {
			// Log error but don't fail - this is called during shell exit
			// and we don't want to prevent the shell from exiting
			fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup session: %v\n", err)
			// Exit with 0 to not interfere with shell exit
			return
		}
	},
}
