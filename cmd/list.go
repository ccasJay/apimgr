package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"apimgr/config"
	"apimgr/internal/utils"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有API配置",
	Long:  "列出所有已保存的API配置",
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()
		configs, err := configManager.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		if len(configs) == 0 {
			fmt.Println("暂无配置")
			return
		}

		fmt.Println("可用的配置:")
		for _, config := range configs {
			// 脱敏显示API密钥
			maskedKey := utils.MaskAPIKey(config.APIKey)
			fmt.Printf("  %s: %s (URL: %s, Model: %s)\n",
				config.Alias, maskedKey, config.BaseURL, config.Model)
		}
	},
}