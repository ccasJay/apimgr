package cmd

import (
	"fmt"
	"os"

	"apimgr/config"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示当前激活的配置",
	Long:  "显示当前激活的API配置信息，包括全局配置和当前shell环境",
	Run: func(cmd *cobra.Command, args []string) {
		// Get shell environment variables
		shellApiKey := os.Getenv("ANTHROPIC_API_KEY")
		shellAuthToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
		shellApiBase := os.Getenv("ANTHROPIC_BASE_URL")
		shellModel := os.Getenv("ANTHROPIC_MODEL")
		shellActiveAlias := os.Getenv("APIMGR_ACTIVE")

		// Get global configuration
		configManager := config.NewConfigManager()
		globalActiveConfig, globalErr := configManager.GetActive()
		var globalActiveAlias string
		if globalErr == nil {
			globalActiveAlias = globalActiveConfig.Alias
		}

		fmt.Println("当前配置状态:")
		fmt.Println("=========================================")

		// Show global active configuration
		fmt.Println("1. 全局活跃配置 (配置文件):")
		if globalErr != nil {
			fmt.Println("   未设置全局活跃配置")
		} else {
			fmt.Printf("   别名: %s\n", globalActiveConfig.Alias)
			if globalActiveConfig.APIKey != "" {
				fmt.Printf("   API Key: %s\n", utils.MaskAPIKey(globalActiveConfig.APIKey))
			}
			if globalActiveConfig.AuthToken != "" {
				fmt.Printf("   Auth Token: %s\n", utils.MaskAPIKey(globalActiveConfig.AuthToken))
			}
			if globalActiveConfig.BaseURL != "" {
				fmt.Printf("   Base URL: %s\n", globalActiveConfig.BaseURL)
			}
			if globalActiveConfig.Model != "" {
				fmt.Printf("   Model: %s\n", globalActiveConfig.Model)
			}
		}

		// Show shell environment configuration
		fmt.Println("\n2. 当前Shell环境:")
		if shellApiKey == "" && shellAuthToken == "" {
			fmt.Println("   未设置环境变量")
		} else {
			if shellActiveAlias != "" {
				fmt.Printf("   别名: %s\n", shellActiveAlias)
			}
			if shellApiKey != "" {
				fmt.Printf("   API Key: %s\n", utils.MaskAPIKey(shellApiKey))
			}
			if shellAuthToken != "" {
				fmt.Printf("   Auth Token: %s\n", utils.MaskAPIKey(shellAuthToken))
			}
			if shellApiBase != "" {
				fmt.Printf("   Base URL: %s\n", shellApiBase)
			}
			if shellModel != "" {
				fmt.Printf("   Model: %s\n", shellModel)
			}
		}

		// Show configuration source
		fmt.Println("\n=========================================")
		if shellApiKey != "" || shellAuthToken != "" {
			if globalErr != nil || (globalActiveAlias != "" && globalActiveAlias != shellActiveAlias) {
				fmt.Println("💡 当前使用的是Shell环境配置 (覆盖了全局配置)")
			} else {
				fmt.Println("💡 当前使用的是全局配置")
			}
		} else {
			if globalErr != nil {
				fmt.Println("💡 未设置任何配置")
			} else {
				fmt.Println("💡 当前使用的是全局配置 (Shell未设置环境变量)")
			}
		}

		fmt.Println("\n💡 提示: 运行 'apimgr install' 安装shell集成以获得更佳体验")
	},
}
