package cmd

import (
	"fmt"
	"os"

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
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]

		configManager := config.NewConfigManager()
		err := configManager.Remove(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration removed: %s\n", alias)
	},
}
