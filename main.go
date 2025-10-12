package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"apimgr/cmd"
)

var rootCmd = &cobra.Command{
	Use:   "apimgr",
	Short: "API密钥和模型配置管理工具",
	Long:  "一个用于管理Anthropic API密钥和模型配置的命令行工具",
}

func main() {
	// Add all subcommands
	rootCmd.AddCommand(cmd.SwitchCmd)
	rootCmd.AddCommand(cmd.AddCmd)
	rootCmd.AddCommand(cmd.ListCmd)
	rootCmd.AddCommand(cmd.RemoveCmd)
	rootCmd.AddCommand(cmd.StatusCmd)
	rootCmd.AddCommand(cmd.LoadActiveCmd)
	rootCmd.AddCommand(cmd.InstallCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}