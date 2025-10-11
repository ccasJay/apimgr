package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"apimgr/config"
)

var RemoveCmd = &cobra.Command{
	Use:   "remove [alias]",
	Short: "删除指定的API配置",
	Long:  "删除指定别名的API配置",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]

		configManager := config.NewConfigManager()
		err := configManager.Remove(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("已删除配置: %s\n", alias)
	},
}