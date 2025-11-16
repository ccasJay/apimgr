package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"apimgr/config"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync [subcommand]",
	Short: "同步配置到各种工具",
	Long: `同步当前激活的配置到各种工具

子命令:
  status     查看同步状态
  claude     同步到 Claude Code
  init       为项目初始化工具配置文件
  list       列出所有可同步的工具`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// 默认显示状态
		showSyncStatus()
	},
}

// status 子命令
var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看同步状态",
	Long:  `查看当前配置同步到各工具的状态`,
	Run:   runSyncStatus,
}

func init() {
	syncCmd.AddCommand(syncStatusCmd)
}

// claude 子命令
var syncClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "同步到 Claude Code",
	Long:  `强制同步当前激活的配置到 Claude Code`,
	Run:   runSyncClaude,
}

func init() {
	syncCmd.AddCommand(syncClaudeCmd)
}

// init 子命令
var syncInitCmd = &cobra.Command{
	Use:   "init",
	Short: "为项目初始化工具配置文件",
	Long:  `在当前项目目录创建各种工具的配置文件模板`,
	Run:   runSyncInit,
}

func init() {
	syncCmd.AddCommand(syncInitCmd)
}

// list 子命令
var syncListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有可同步的工具",
	Long:  `显示所有支持自动同步的工具列表`,
	Run:   runSyncList,
}

func init() {
	syncCmd.AddCommand(syncListCmd)
}

func showSyncStatus() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("配置同步状态")
	fmt.Println(strings.Repeat("=", 60))

	configManager := config.NewConfigManager()

	// 显示当前激活配置
	active, err := configManager.GetActive()
	if err != nil {
		fmt.Println("\n❌ 没有活动配置")
		return
	}

	fmt.Printf("\n当前配置: %s\n", active.Alias)
	fmt.Printf("模型: %s\n", active.Model)
	fmt.Printf("API Key: %s\n", utils.MaskAPIKey(active.APIKey))
	fmt.Printf("Base URL: %s\n", active.BaseURL)

	// 检查同步状态
	fmt.Println("\n同步状态:")

	// 全局 Claude Code
	globalClaudePath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")
	if _, err := os.Stat(globalClaudePath); err == nil {
		fmt.Println("✅ Claude Code (全局): ~/.claude/settings.json")
	} else {
		fmt.Println("⚪ Claude Code (全局): 未安装")
	}

	// 项目级 Claude Code
	workDir, _ := os.Getwd()
	projectClaudePath := filepath.Join(workDir, ".claude", "settings.json")
	if _, err := os.Stat(projectClaudePath); err == nil {
		fmt.Printf("✅ Claude Code (项目): %s\n", projectClaudePath)
	} else {
		fmt.Printf("⚪ Claude Code (项目): %s (未初始化)\n", projectClaudePath)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

func runSyncStatus(cmd *cobra.Command, args []string) {
	showSyncStatus()
}

func runSyncClaude(cmd *cobra.Command, args []string) {
	configManager := config.NewConfigManager()

	_, err := configManager.GetActive()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("正在同步到 Claude Code...")

	// 同步全局设置
	if err := configManager.GenerateActiveScript(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 同步失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✅ 同步完成!")
}

func runSyncInit(cmd *cobra.Command, args []string) {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 获取当前目录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("正在为项目初始化工具配置文件...")

	// 创建 .claude 目录（如果不存在）
	claudeDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 创建 .claude 目录失败: %v\n", err)
		os.Exit(1)
	}

	// 创建 Claude Code 配置文件
	claudeSettingsPath := filepath.Join(claudeDir, "settings.json")
	if _, err := os.Stat(claudeSettingsPath); os.IsNotExist(err) {
		settings := map[string]interface{}{
			"env": map[string]interface{}{
				"ANTHROPIC_MODEL":            "claude-3-opus",
				"ANTHROPIC_API_KEY":          "",
				"ANTHROPIC_BASE_URL":         "",
				"ANTHROPIC_AUTH_TOKEN":       "",
				"ANTHROPIC_SMALL_FAST_MODEL": "",
			},
			"enabledPlugins":        map[string]interface{}{},
			"alwaysThinkingEnabled": true,
		}

		if err := writeJSONFile(claudeSettingsPath, settings); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 创建 Claude Code 配置文件失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ 创建 Claude Code 配置: %s\n", claudeSettingsPath)
	} else {
		fmt.Printf("ℹ️  Claude Code 配置已存在: %s\n", claudeSettingsPath)
	}

	fmt.Println("\n✅ 项目初始化完成!")
	fmt.Println("\n现在 apimgr 会自动同步配置到此项目。")
}

func runSyncList(cmd *cobra.Command, args []string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("支持同步的工具")
	fmt.Println(strings.Repeat("=", 60))

	tools := []struct {
		Name   string
		Config string
		Status string
	}{
		{"Claude Code", "~/.claude/settings.json", "✅ 已实现"},
		{"Grok (xAI)", "~/.config/grok/config.json", "🚧 规划中"},
		{"GitHub Copilot", "~/.config/copilot/config.json", "🚧 规划中"},
		{"OpenAI CLI", "~/.config/openai/config.json", "🚧 规划中"},
	}

	fmt.Println()
	for _, tool := range tools {
		fmt.Printf("%-20s %-40s %s\n", tool.Name, tool.Config, tool.Status)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

// writeJSONFile 写入 JSON 文件
func writeJSONFile(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonData, 0600)
}
