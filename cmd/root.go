package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "apimgr",
	Short: "API密钥和模型配置管理工具",
	Long:  "一个用于管理Anthropic API密钥和模型配置的命令行工具",
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}
