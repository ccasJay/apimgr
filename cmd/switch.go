package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"apimgr/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(switchCmd)
}

var switchCmd = &cobra.Command{
	Use:   "switch [alias]",
	Short: "切换到指定的API配置",
	Long: `切换到指定的API配置，并输出export命令用于环境变量设置

要使环境变量在当前shell中生效，有以下两种方式：
1. 使用 eval: eval "$(apimgr switch <alias>)"
2. 安装shell集成: apimgr install （推荐，安装后可直接使用 apimgr switch）`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]

		configManager := config.NewConfigManager()

		// Set the active configuration
		err := configManager.SetActive(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		// Generate active.env script for auto-loading
		if err := configManager.GenerateActiveScript(); err != nil {
			fmt.Fprintf(os.Stderr, "警告: 生成激活脚本失败: %v\n", err)
		}

		// 显示同步信息
		showSyncInfo(alias)

		// Get the configuration
		apiConfig, err := configManager.Get(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		// Clear previous environment variables
		fmt.Println("unset ANTHROPIC_API_KEY")
		fmt.Println("unset ANTHROPIC_AUTH_TOKEN")
		fmt.Println("unset ANTHROPIC_BASE_URL")
		fmt.Println("unset ANTHROPIC_MODEL")
		fmt.Println("unset APIMGR_ACTIVE")

		// Export new environment variables
		if apiConfig.APIKey != "" {
			fmt.Printf("export ANTHROPIC_API_KEY=\"%s\"\n", apiConfig.APIKey)
		}
		if apiConfig.AuthToken != "" {
			fmt.Printf("export ANTHROPIC_AUTH_TOKEN=\"%s\"\n", apiConfig.AuthToken)
		}
		if apiConfig.BaseURL != "" {
			fmt.Printf("export ANTHROPIC_BASE_URL=\"%s\"\n", apiConfig.BaseURL)
		}
		if apiConfig.Model != "" {
			fmt.Printf("export ANTHROPIC_MODEL=\"%s\"\n", apiConfig.Model)
		}
		fmt.Printf("export APIMGR_ACTIVE=\"%s\"\n", alias)

		// Print success message to stderr so it doesn't interfere with eval
		fmt.Fprintf(os.Stderr, "✓ 已切换到配置: %s\n", alias)
	},
}

// showSyncInfo 显示同步状态信息
func showSyncInfo(alias string) {
	// 检查同步状态
	globalClaudePath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")
	projectClaudePath := filepath.Join(".", ".claude", "settings.json")

	hasGlobal := false
	hasProject := false

	if _, err := os.Stat(globalClaudePath); err == nil {
		hasGlobal = true
	}
	if _, err := os.Stat(projectClaudePath); err == nil {
		hasProject = true
	}

	if hasGlobal || hasProject {
		fmt.Fprintf(os.Stderr, "\n✅ 配置同步状态:\n")
		if hasGlobal {
			fmt.Fprintf(os.Stderr, "   • 全局 Claude Code: ~/.claude/settings.json\n")
		}
		if hasProject {
			fmt.Fprintf(os.Stderr, "   • 项目级 Claude Code: %s\n", projectClaudePath)
		}
		fmt.Fprintf(os.Stderr, "\n💡 配置已自动同步到 Claude Code，可以直接使用。\n")
	}
}
