package cmd

import (
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
