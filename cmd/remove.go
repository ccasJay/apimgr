package cmd

import (
	"fmt"

	"apimgr/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove [alias]",
	Short: "Remove specified API configuration",
	Long:  "Remove API configuration with specified alias",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		configManager, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to initialize config manager: %w", err)
		}
		if err := configManager.Remove(alias); err != nil {
			return err
		}

		fmt.Printf("Configuration removed: %s\n", alias)
		return nil
	},
}
