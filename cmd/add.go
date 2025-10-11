package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"apimgr/config"
	"apimgr/internal/utils"
)

// isTerminal 检查是否在真正的终端中运行
func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// runInteractiveMode 处理交互式输入
func runInteractiveMode(prefilledAPIKey, prefilledAuthToken, defaultURL, defaultModel, presetAuthType string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入配置别名: ")
	alias, _ := reader.ReadString('\n')
	alias = strings.TrimSpace(alias)
	if alias == "" {
		fmt.Println("错误: 别名不能为空")
		os.Exit(1)
	}

	var apiKey, authToken, url, model string

	// 根据预设类型处理
	if presetAuthType == "api_key" {
		// 预设为API密钥模式
		apiKey = prefilledAPIKey
		fmt.Printf("已设置API密钥: %s\n", utils.MaskAPIKey(apiKey))
		fmt.Print("请输入认证令牌 (可选): ")
		authToken, _ = reader.ReadString('\n')
		authToken = strings.TrimSpace(authToken)
	} else if presetAuthType == "auth_token" {
		// 预设为认证令牌模式
		authToken = prefilledAuthToken
		fmt.Printf("已设置认证令牌: %s\n", utils.MaskAPIKey(authToken))
		fmt.Print("请输入API密钥 (可选): ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
	} else {
		// 完全交互式选择
		fmt.Print("请输入API密钥 (可选，与auth token二选一): ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		fmt.Print("请输入认证令牌 (可选，与API密钥二选一): ")
		authToken, _ = reader.ReadString('\n')
		authToken = strings.TrimSpace(authToken)
	}

	// 验证至少有一种认证方式
	if apiKey == "" && authToken == "" {
		fmt.Println("错误: 必须提供API密钥或认证令牌")
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

	apiConfig := config.APIConfig{
		Alias:     alias,
		APIKey:    apiKey,
		AuthToken: authToken,
		BaseURL:   url,
		Model:     model,
	}

	configManager := config.NewConfigManager()
	err := configManager.Add(apiConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已添加配置: %s\n", alias)
}

var AddCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "添加新的API配置",
	Long: `添加新的API配置

用法1: 完全交互式
  apimgr add

用法2: API密钥预设交互式
  apimgr add --sk

用法3: 认证令牌预设交互式
  apimgr add --ak

用法4: 命令行参数
  apimgr add my-config --sk sk-xxx --url https://api.example.com --model claude-3

用法5: 命令行参数
  apimgr add my-config --ak bearer-token --url https://api.example.com --model claude-3`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var alias, apiKey, authToken, url, model string

		// 从命令行标志获取参数
		apiKeyFlag := cmd.Flags().Lookup("sk").Changed
		authTokenFlag := cmd.Flags().Lookup("ak").Changed
		apiKey, _ = cmd.Flags().GetString("sk")
		authToken, _ = cmd.Flags().GetString("ak")
		url, _ = cmd.Flags().GetString("url")
		model, _ = cmd.Flags().GetString("model")

		// 如果使用了 --sk 或 --ak 标志且没有别名，进入交互式模式
		if (apiKeyFlag || authTokenFlag) && len(args) == 0 {
			if !isTerminal() {
				fmt.Println("当前环境不支持交互式输入，请提供别名:")
				fmt.Println("apimgr add <alias> (--sk <api-key> | --ak <auth-token>) [--url <url>] [--model <model>]")
				os.Exit(1)
			}
			// 根据标志决定预设的认证类型
			if apiKeyFlag {
				runInteractiveMode(apiKey, "", url, model, "api_key")
			} else {
				runInteractiveMode("", authToken, url, model, "auth_token")
			}
			return
		}

		if len(args) == 1 {
			// 命令行模式
			alias = args[0]
			if url == "" {
				url = "https://api.anthropic.com"
			}
			// 验证至少有一种认证方式
			if apiKey == "" && authToken == "" {
				fmt.Println("错误: 必须提供 --sk 或 --ak 参数")
				os.Exit(1)
			}
		} else {
			// 完全交互式模式
			if !isTerminal() {
				fmt.Println("当前环境不支持交互式输入，请提供别名:")
				fmt.Println("apimgr add <alias> (--sk <api-key> | --ak <auth-token>) [--url <url>] [--model <model>]")
				os.Exit(1)
			}
			runInteractiveMode("", "", url, model, "none")
			return
		}

		apiConfig := config.APIConfig{
			Alias:     alias,
			APIKey:    apiKey,
			AuthToken: authToken,
			BaseURL:   url,
			Model:     model,
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
	AddCmd.Flags().String("sk", "", "API密钥 (ANTHROPIC_API_KEY)")
	AddCmd.Flags().String("ak", "", "认证令牌 (ANTHROPIC_AUTH_TOKEN)")
}