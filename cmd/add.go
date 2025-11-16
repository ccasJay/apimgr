package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"apimgr/config"
	"github.com/spf13/cobra"
)

// APIConfigBuilder 负责构建和验证 APIConfig
type APIConfigBuilder struct {
	config *config.APIConfig
}

// NewAPIConfigBuilder 创建新的构建器
func NewAPIConfigBuilder() *APIConfigBuilder {
	return &APIConfigBuilder{
		config: &config.APIConfig{},
	}
}

// SetAlias 设置别名
func (b *APIConfigBuilder) SetAlias(alias string) *APIConfigBuilder {
	b.config.Alias = alias
	return b
}

// SetAPIKey 设置API密钥
func (b *APIConfigBuilder) SetAPIKey(apiKey string) *APIConfigBuilder {
	b.config.APIKey = apiKey
	return b
}

// SetAuthToken 设置认证令牌
func (b *APIConfigBuilder) SetAuthToken(authToken string) *APIConfigBuilder {
	b.config.AuthToken = authToken
	return b
}

// SetBaseURL 设置基础URL
func (b *APIConfigBuilder) SetBaseURL(url string) *APIConfigBuilder {
	b.config.BaseURL = url
	return b
}

// SetModel 设置模型
func (b *APIConfigBuilder) SetModel(model string) *APIConfigBuilder {
	b.config.Model = model
	return b
}

// Build 构建配置
func (b *APIConfigBuilder) Build() (*config.APIConfig, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	return b.config, nil
}

// validate 验证配置
func (b *APIConfigBuilder) validate() error {
	if b.config.Alias == "" {
		return fmt.Errorf("别名不能为空")
	}
	if b.config.APIKey == "" && b.config.AuthToken == "" {
		return fmt.Errorf("API密钥和认证令牌不能同时为空")
	}
	if b.config.BaseURL != "" {
		if _, err := url.ParseRequestURI(b.config.BaseURL); err != nil {
			return fmt.Errorf("无效的URL格式: %s", b.config.BaseURL)
		}
	}
	return nil
}

// InputCollector 负责收集用户输入
type InputCollector struct{}

// isTerminal 检查是否在真正的终端中运行
func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// CollectInteractively 交互式收集输入
func (ic *InputCollector) CollectInteractively(presetType string) (*config.APIConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入配置别名: ")
	alias, _ := reader.ReadString('\n')
	alias = strings.TrimSpace(alias)

	var apiKey, authToken, url, model string

	// 根据预设类型处理
	switch presetType {
	case "api_key":
		// API密钥已通过命令行提供
		fmt.Print("请输入认证令牌 (可选): ")
		authToken, _ = reader.ReadString('\n')
		authToken = strings.TrimSpace(authToken)
	case "auth_token":
		// 认证令牌已通过命令行提供
		fmt.Print("请输入API密钥 (可选): ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
	default:
		// 完全交互式
		fmt.Print("请输入API密钥 (可选，与auth token二选一): ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		fmt.Print("请输入认证令牌 (可选，与API密钥二选一): ")
		authToken, _ = reader.ReadString('\n')
		authToken = strings.TrimSpace(authToken)
	}

	// 验证至少有一种认证方式
	if apiKey == "" && authToken == "" {
		return nil, fmt.Errorf("必须提供API密钥或认证令牌")
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

	// 使用构建器创建配置
	builder := NewAPIConfigBuilder().
		SetAlias(alias).
		SetAPIKey(apiKey).
		SetAuthToken(authToken).
		SetBaseURL(url).
		SetModel(model)

	return builder.Build()
}

var addCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "添加新的API配置",
	Long: `添加新的API配置 - 支持多种模式：

1. 完全交互式:
   apimgr add

2. 命令行快速添加:
   apimgr add my-config --sk sk-xxx --url https://api.anthropic.com --model claude-3
   apimgr add my-config --ak bearer-token -u https://api.anthropic.com -m claude-3

3. 预设模式 (有预设但缺少别名):
   apimgr add --sk sk-xxx -u https://api.anthropic.com -m claude-3
   apimgr add --ak bearer-token`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()
		collector := &InputCollector{}

		// 决定输入模式
		var cfg *config.APIConfig
		var err error

		hasSK := cmd.Flags().Lookup("sk").Changed
		hasAK := cmd.Flags().Lookup("ak").Changed
		hasAlias := len(args) == 1

		switch {
		case hasAlias:
			// 命令行模式 - 有别名和参数
			alias := args[0]
			apiKey, _ := cmd.Flags().GetString("sk")
			authToken, _ := cmd.Flags().GetString("ak")
			url, _ := cmd.Flags().GetString("url")
			model, _ := cmd.Flags().GetString("model")

			// 设置默认值
			if url == "" {
				url = "https://api.anthropic.com"
			}

			// 验证至少有一种认证方式
			if apiKey == "" && authToken == "" {
				fmt.Println("❌ 错误: 必须提供 --sk 或 --ak 参数")
				fmt.Println("\n💡 用法示例:")
				fmt.Println("  apimgr add my-config --sk sk-xxx")
				fmt.Println("  apimgr add my-config --ak token-xxx")
				os.Exit(1)
			}

			builder := NewAPIConfigBuilder().
				SetAlias(alias).
				SetAPIKey(apiKey).
				SetAuthToken(authToken).
				SetBaseURL(url).
				SetModel(model)

			cfg, err = builder.Build()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ 错误: %v\n", err)
				os.Exit(1)
			}

		case hasSK || hasAK:
			// 预设模式 - 有预设参数但没有别名，进入交互式
			presetType := ""
			if hasSK {
				presetType = "api_key"
			} else {
				presetType = "auth_token"
			}

			if !isTerminal() {
				fmt.Println("❌ 当前环境不支持交互式输入，请提供别名:")
				fmt.Printf("  apimgr add <alias> --%s <value> [--url <url>] [--model <model>]\n",
					map[bool]string{true: "sk", false: "ak"}[hasSK])
				os.Exit(1)
			}

			cfg, err = collector.CollectInteractively(presetType)
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}

		default:
			// 完全交互式模式
			if !isTerminal() {
				fmt.Println("❌ 当前环境不支持交互式输入")
				fmt.Printf("  apimgr add <alias> --sk <key> [--url <url>] [--model <model>]\n")
				os.Exit(1)
			}

			cfg, err = collector.CollectInteractively("")
			if err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		}

		// 保存配置
		err = configManager.Add(*cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 保存配置失败: %v\n", err)
			os.Exit(1)
		}

		// 生成激活脚本
		if err := configManager.GenerateActiveScript(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  警告: 生成激活脚本失败: %v\n", err)
		}

		fmt.Printf("✅ 已添加配置: %s\n", cfg.Alias)
		fmt.Println("\n💡 提示: 运行 'apimgr switch <alias>' 切换到此配置")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("url", "u", "", "API基础URL")
	addCmd.Flags().StringP("model", "m", "", "模型名称")
	addCmd.Flags().String("sk", "", "API密钥 (ANTHROPIC_API_KEY)")
	addCmd.Flags().String("ak", "", "认证令牌 (ANTHROPIC_AUTH_TOKEN)")
}
