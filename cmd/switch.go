package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"apimgr/config"
)

var SwitchCmd = &cobra.Command{
	Use:   "switch [alias]",
	Short: "切换到指定的API配置",
	Long:  "切换到指定的API配置，并输出export命令用于环境变量设置",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]

		configManager := config.NewConfigManager()
		apiConfig, err := configManager.Get(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		// 输出export命令，用户需要通过eval执行
		if apiConfig.APIKey != "" {
			fmt.Printf("export ANTHROPIC_API_KEY=\"%s\"\n", apiConfig.APIKey)
		}
		if apiConfig.AuthToken != "" {
			fmt.Printf("export ANTHROPIC_AUTH_TOKEN=\"%s\"\n", apiConfig.AuthToken)
		}
		if apiConfig.BaseURL != "" {
			fmt.Printf("export ANTHROPIC_API_BASE=\"%s\"\n", apiConfig.BaseURL)
		}
		if apiConfig.Model != "" {
			fmt.Printf("export ANTHROPIC_MODEL=\"%s\"\n", apiConfig.Model)
		}
	},
}