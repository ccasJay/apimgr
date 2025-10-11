package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"apimgr/config"
)

var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "添加新的API配置",
	Long:  "添加新的API配置，需要提供别名、API密钥、基础URL和模型",
	Run: func(cmd *cobra.Command, args []string) {
		alias, _ := cmd.Flags().GetString("alias")
		key, _ := cmd.Flags().GetString("key")
		url, _ := cmd.Flags().GetString("url")
		model, _ := cmd.Flags().GetString("model")

		apiConfig := config.APIConfig{
			Alias:   alias,
			APIKey:  key,
			BaseURL: url,
			Model:   model,
		}

		configManager := config.NewConfigManager()
		err := configManager.Add(apiConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("已添加配置: %s\n", alias)
	},
}

func init() {
	AddCmd.Flags().StringP("alias", "n", "", "配置别名")
	AddCmd.Flags().StringP("key", "k", "", "API密钥")
	AddCmd.Flags().StringP("url", "u", "", "API基础URL")
	AddCmd.Flags().StringP("model", "m", "", "模型名称")
	AddCmd.MarkFlagRequired("alias")
	AddCmd.MarkFlagRequired("key")
}