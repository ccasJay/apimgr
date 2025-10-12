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
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
		apiBase := os.Getenv("ANTHROPIC_BASE_URL")
		model := os.Getenv("ANTHROPIC_MODEL")
		activeAlias := os.Getenv("APIMGR_ACTIVE")

		if apiKey == "" && authToken == "" {
			fmt.Println("当前未设置API配置")
			return
		}

		fmt.Println("当前激活的配置:")
		if activeAlias != "" {
			fmt.Printf("  别名: %s\n", activeAlias)
		}
		if apiKey != "" {
			fmt.Printf("  API Key: %s\n", utils.MaskAPIKey(apiKey))
		}
		if authToken != "" {
			fmt.Printf("  Auth Token: %s\n", utils.MaskAPIKey(authToken))
		}
		if apiBase != "" {
			fmt.Printf("  Base URL: %s\n", apiBase)
		}
		if model != "" {
			fmt.Printf("  Model: %s\n", model)
		}
	},
}