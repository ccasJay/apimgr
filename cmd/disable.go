package cmd

import (
	"fmt"

	"github.com/ccasJay/apimgr/config"

	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable API management by apimgr",
	Long:  "Disable API management by apimgr. This will clear the active configuration and environment variables.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configManager, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}

		if err := configManager.Disable(); err != nil {
			return fmt.Errorf("failed to disable API management: %w", err)
		}

		fmt.Println("✅ API management by apimgr has been disabled.")
		fmt.Println("💡 Tip: To fully apply this change to your current session, you may need to restart your terminal or run: source ~/.zshrc (or your shell's config file)")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
