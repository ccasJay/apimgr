package cmd

import (
	"fmt"
	"os"

	"apimgr/config"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
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

		// Get active configuration name
		activeName, _ := configManager.GetActiveName()

		fmt.Println("可用的配置:")
		for _, config := range configs {
			// 脱敏显示API密钥或认证令牌
			var authInfo string
			if config.APIKey != "" {
				authInfo = "API Key: " + utils.MaskAPIKey(config.APIKey)
			} else {
				authInfo = "Auth Token: " + utils.MaskAPIKey(config.AuthToken)
			}

			// Mark active configuration with *
			activeMarker := " "
			if config.Alias == activeName {
				activeMarker = "*"
			}

			fmt.Printf("%s %s: %s (URL: %s, Model: %s)\n",
				activeMarker, config.Alias, authInfo, config.BaseURL, config.Model)
		}

		if activeName != "" {
			fmt.Printf("\n* 表示当前活动配置\n")
		}
	},
}
