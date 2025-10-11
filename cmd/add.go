package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"apimgr/config"
)

// isTerminal 检查是否在真正的终端中运行
func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

var AddCmd = &cobra.Command{
	Use:   "add [alias] [key]",
	Short: "添加新的API配置",
	Long: `添加新的API配置

用法1: 交互式添加（推荐）
  apimgr add

用法2: 命令行参数
  apimgr add my-config sk-xxx --url https://api.example.com --model claude-3`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var alias, key, url, model string

		if len(args) == 2 {
			// 使用命令行参数
			alias, key = args[0], args[1]
			url, _ = cmd.Flags().GetString("url")
			model, _ = cmd.Flags().GetString("model")
			if url == "" {
				url = "https://api.anthropic.com"
			}
		} else {
			// 交互式输入（在非TTY环境中使用命令行参数作为备选）
			if !isTerminal() {
				fmt.Println("当前环境不支持交互式输入，请使用命令行参数:")
				fmt.Println("apimgr add <alias> <key> [--url <url>] [--model <model>]")
				os.Exit(1)
			}

			reader := bufio.NewReader(os.Stdin)

			fmt.Print("请输入配置别名: ")
			alias, _ = reader.ReadString('\n')
			alias = strings.TrimSpace(alias)
			if alias == "" {
				fmt.Println("错误: 别名不能为空")
				os.Exit(1)
			}

			fmt.Print("请输入API密钥: ")
			key, _ = reader.ReadString('\n')
			key = strings.TrimSpace(key)
			if key == "" {
				fmt.Println("错误: API密钥不能为空")
				os.Exit(1)
			}

			fmt.Print("请输入API基础URL (可选，默认 https://api.anthropic.com): ")
			url, _ = reader.ReadString('\n')
			url = strings.TrimSpace(url)
			if url == "" {
				url = "https://api.anthropic.com"
			}

			fmt.Print("请输入模型名称 (可选): ")
			model, _ = reader.ReadString('\n')
			model = strings.TrimSpace(model)
		}

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
	AddCmd.Flags().StringP("url", "u", "", "API基础URL")
	AddCmd.Flags().StringP("model", "m", "", "模型名称")
}