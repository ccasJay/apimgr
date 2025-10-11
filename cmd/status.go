package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"apimgr/internal/utils"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示当前激活的配置",
	Long:  "显示当前激活的API配置信息",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: 实现显示当前配置逻辑
		// 这里可以检查环境变量来显示当前激活的配置
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		apiBase := os.Getenv("ANTHROPIC_API_BASE")
		model := os.Getenv("ANTHROPIC_MODEL")

		if apiKey == "" {
			fmt.Println("当前未设置API配置")
			return
		}

		fmt.Println("当前激活的配置:")
		fmt.Printf("  API Key: %s\n", utils.MaskAPIKey(apiKey))
		if apiBase != "" {
			fmt.Printf("  API Base: %s\n", apiBase)
		}
		if model != "" {
			fmt.Printf("  Model: %s\n", model)
		}
	},
}