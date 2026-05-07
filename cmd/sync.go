package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccasJay/apimgr/config"
	"github.com/ccasJay/apimgr/internal/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync [subcommand]",
	Short: "Sync configuration to various tools",
	Long: `Sync current active configuration to various tools

Subcommands:
  status     View sync status
  claude     Sync to Claude Code
  init       Initialize tool configuration files for project
  list       List all tools that can be synced`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to show status
		return showSyncStatus()
	},
}

// status subcommand
var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "View sync status",
	Long:  `View current configuration sync status to various tools`,
	RunE:  runSyncStatusE,
}

func init() {
	syncCmd.AddCommand(syncStatusCmd)
}

// claude subcommand
var syncClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Sync to Claude Code",
	Long:  `Force sync current active configuration to Claude Code`,
	RunE:  runSyncClaudeE,
}

func init() {
	syncCmd.AddCommand(syncClaudeCmd)
}

// init subcommand
var syncInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize tool configuration files for project",
	Long:  `Create configuration file templates for various tools in the current project directory`,
	RunE:  runSyncInitE,
}

func init() {
	syncCmd.AddCommand(syncInitCmd)
}

// list subcommand
var syncListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tools that can be synced",
	Long:  `Show list of all tools that support automatic sync`,
	Run:   runSyncList,
}

func init() {
	syncCmd.AddCommand(syncListCmd)
}

func showSyncStatus() error {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Configuration Sync Status")
	fmt.Println(strings.Repeat("=", 60))

	configManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// Show current active configuration
	active, err := configManager.GetActive()
	if err != nil {
		fmt.Println("\n❌ No active configuration")
		return nil
	}

	fmt.Printf("\nCurrent configuration: %s\n", active.Alias)
	fmt.Printf("Model: %s\n", active.Model)
	fmt.Printf("API Key: %s\n", utils.MaskAPIKey(active.APIKey))
	fmt.Printf("Base URL: %s\n", active.BaseURL)

	// Check sync status
	fmt.Println("\nSync status:")

	// Global Claude Code
	globalClaudePath := filepath.Join(os.Getenv("HOME"), ".claude", "settings.json")
	if _, err := os.Stat(globalClaudePath); err == nil {
		fmt.Println("✅ Claude Code (Global): ~/.claude/settings.json")
	} else {
		fmt.Println("⚪ Claude Code (Global): Not installed")
	}

	// Project-level Claude Code
	workDir, _ := os.Getwd()
	projectClaudePath := filepath.Join(workDir, ".claude", "settings.json")
	if _, err := os.Stat(projectClaudePath); err == nil {
		fmt.Printf("✅ Claude Code (Project): %s\n", projectClaudePath)
	} else {
		fmt.Printf("⚪ Claude Code (Project): %s (Not initialized)\n", projectClaudePath)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	return nil
}

func runSyncStatusE(cmd *cobra.Command, args []string) error {
	return showSyncStatus()
}

func runSyncClaudeE(cmd *cobra.Command, args []string) error {
	configManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	if _, err := configManager.GetActive(); err != nil {
		return err
	}

	fmt.Println("Syncing to Claude Code...")

	// Sync global settings
	if err := configManager.GenerateActiveScript(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Println("\n✅ Sync completed!")
	return nil
}

func runSyncInitE(cmd *cobra.Command, args []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Println("Initializing tool configuration files for project...")

	// Create .claude directory (if it doesn't exist)
	claudeDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Create Claude Code configuration file
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
			return fmt.Errorf("failed to create Claude Code configuration file: %w", err)
		}

		fmt.Printf("✅ Created Claude Code configuration: %s\n", claudeSettingsPath)
	} else {
		fmt.Printf("ℹ️  Claude Code configuration already exists: %s\n", claudeSettingsPath)
	}

	fmt.Println("\n✅ Project initialization completed!")
	fmt.Println("\napimgr will now automatically sync configuration to this project.")
	return nil
}

func runSyncList(cmd *cobra.Command, args []string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Supported Sync Tools")
	fmt.Println(strings.Repeat("=", 60))

	tools := []struct {
		Name   string
		Config string
		Status string
	}{
		{"Claude Code", "~/.claude/settings.json", "✅ Implemented"},
		{"Grok (xAI)", "~/.config/grok/config.json", "🚧 Planned"},
		{"GitHub Copilot", "~/.config/copilot/config.json", "🚧 Planned"},
		{"OpenAI CLI", "~/.config/openai/config.json", "🚧 Planned"},
	}

	fmt.Println()
	for _, tool := range tools {
		fmt.Printf("%-20s %-40s %s\n", tool.Name, tool.Config, tool.Status)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

// writeJSONFile writes JSON file
func writeJSONFile(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonData, 0600)
}
