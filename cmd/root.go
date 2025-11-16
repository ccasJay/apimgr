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
	Short: "API密钥和模型配置管理工具",
	Long:  "一个用于管理Anthropic API密钥和模型配置的命令行工具",
	// 版本信息将在Execute函数中设置
}

// Execute executes the root command
func Execute() error {
	// 设置版本信息
	rootCmd.Version = version

	// 设置版本输出格式
	rootCmd.SetVersionTemplate(`apimgr {{.Version}}
Commit: ` + commit + `
Date: ` + date + `
`)

	return rootCmd.Execute()
}
