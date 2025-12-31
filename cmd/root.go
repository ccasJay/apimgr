package cmd

import (
	"apimgr/internal/tui"

	"github.com/spf13/cobra"
)

// Version information
var (
	version string
	commit  string
	date    string
)

// SetVersionInfo sets the version information
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}

var rootCmd = &cobra.Command{
	Use:   "apimgr",
	Short: "API key and model configuration management tool",
	Long:  "A command line tool for managing Anthropic API keys and model configurations",
	// Version information will be set in the Execute function
	RunE: func(cmd *cobra.Command, args []string) error {
		// When no subcommand is provided, launch the TUI interface
		// Requirements: 1.1, 1.4
		return tui.Run()
	},
}

// Execute executes the root command
func Execute() error {
	// Set version info
	rootCmd.Version = version

	// Set version output format
	rootCmd.SetVersionTemplate(`apimgr {{.Version}}
Commit: ` + commit + `
Date: ` + date + `
`)

	return rootCmd.Execute()
}
