package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"apimgr/config"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

// EditFlags represents the command line flags for edit command
type EditFlags struct {
	Alias     string
	APIKey    string
	AuthToken string
	BaseURL   string
	Model     string
}

func init() {
	rootCmd.AddCommand(editCmd)

	// Define flags for non-interactive editing
	editCmd.Flags().String("alias", "", "修改配置别名")
	editCmd.Flags().String("sk", "", "修改API密钥")
	editCmd.Flags().String("ak", "", "修改认证令牌")
	editCmd.Flags().String("url", "", "修改基础URL")
	editCmd.Flags().String("model", "", "修改模型名称")
}

var editCmd = &cobra.Command{
	Use:   "edit <alias>",
	Short: "编辑配置",
	Long: `编辑已保存的API配置

默认情况下，此命令将以交互式界面引导您编辑配置的各个字段。
如果提供了命令行参数，则会直接应用更改，而不进入交互模式。

示例：
  # 交互式编辑
  apimgr edit myconfig

  # 非交互式编辑（修改API密钥）
  apimgr edit myconfig --sk sk-ant-api03-xxx

  # 非交互式编辑多个字段
  apimgr edit myconfig --url https://api.anthropic.com --model claude-3-opus-20240229`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]

		// Check if any flags are provided for non-interactive editing
		aliasFlag, _ := cmd.Flags().GetString("alias")
		skFlag, _ := cmd.Flags().GetString("sk")
		akFlag, _ := cmd.Flags().GetString("ak")
		urlFlag, _ := cmd.Flags().GetString("url")
		modelFlag, _ := cmd.Flags().GetString("model")

		// Parse flags into updates map
		updates := make(map[string]string)
		if aliasFlag != "" {
			updates["alias"] = aliasFlag
		}
		if skFlag != "" {
			updates["api_key"] = skFlag
		}
		if akFlag != "" {
			updates["auth_token"] = akFlag
		}
		if urlFlag != "" {
			updates["base_url"] = urlFlag
		}
		if modelFlag != "" {
			updates["model"] = modelFlag
		}

		configManager := config.NewConfigManager()

		if len(updates) > 0 {
			// Non-interactive mode: directly apply changes
			if err := saveAndApplyChanges(configManager, alias, updates); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}

			// Show success message with updated alias
			updatedAlias := alias
			if newAlias, ok := updates["alias"]; ok {
				updatedAlias = newAlias
			}
			fmt.Printf("✅ 配置 '%s' 已更新\n", updatedAlias)
		} else {
			// Interactive mode: guide user through editing
			if err := editConfig(alias); err != nil {
				fmt.Fprintf(os.Stderr, "错误: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

// FieldType represents the type of field being edited
type FieldType int

const (
	FieldAlias FieldType = iota
	FieldAPIKey
	FieldAuthToken
	FieldBaseURL
	FieldModel
)

func editConfig(alias string) error {
	configManager := config.NewConfigManager()

	// Get the current configuration
	currentConfig, err := configManager.Get(alias)
	if err != nil {
		return fmt.Errorf("获取配置失败: %v", err)
	}

	// Display current configuration
	displayConfig(*currentConfig)

	// Collect user edits interactively
	updates, err := collectUserEdits(currentConfig, configManager)
	if err != nil {
		return err
	}

	// Apply and save changes
	if err := saveAndApplyChanges(configManager, alias, updates); err != nil {
		return err
	}

	updatedAlias := getUpdatedAlias(alias, updates)
	fmt.Printf("\n✅ 配置 '%s' 已更新\n", updatedAlias)
	return nil
}

// collectUserEdits handles the interactive editing loop and returns collected updates
func collectUserEdits(currentConfig *config.APIConfig, configManager *config.ConfigManager) (map[string]string, error) {
	reader := bufio.NewReader(os.Stdin)
	updates := make(map[string]string)

	for {
		showMenu(len(updates))
		choice := getUserChoice(reader)

		// Handle special cases: quit, preview, or save
		if shouldQuit(choice) {
			fmt.Println("\n已取消编辑，未保存更改")
			return nil, fmt.Errorf("操作已取消")
		}

		if shouldPreview(choice) {
			if err := handlePreview(currentConfig, updates); err != nil {
				fmt.Printf("\n❌ %v\n", err)
			}
			continue
		}

		if shouldSave(choice) {
			if len(updates) == 0 {
				fmt.Println("\n没有更改，跳过保存")
				return nil, fmt.Errorf("没有更改")
			}
			if !confirmSave(reader) {
				fmt.Println("\n已取消保存")
				return nil, fmt.Errorf("保存已取消")
			}
			break
		}

		// Process field selection
		if err := handleFieldSelection(reader, currentConfig, updates, choice, configManager); err != nil {
			fmt.Printf("\n❌ %v\n", err)
		}
	}

	return updates, nil
}

// shouldQuit checks if user wants to quit without saving
func shouldQuit(choice string) bool {
	return choice == "q" || choice == "Q"
}

// shouldPreview checks if user wants to preview changes
func shouldPreview(choice string) bool {
	return choice == "p" || choice == "P"
}

// shouldSave checks if user wants to save changes
func shouldSave(choice string) bool {
	return choice == "0"
}

func showMenu(updateCount int) {
	fmt.Println("\n" + strings.Repeat("-", 60))
	if updateCount > 0 {
		fmt.Printf("已更改 %d 个字段\n", updateCount)
	}
	fmt.Println("请选择要修改的字段 (输入数字):")
	fmt.Println("1. 别名 (alias)")
	fmt.Println("2. API密钥 (api_key)")
	fmt.Println("3. 认证令牌 (auth_token)")
	fmt.Println("4. 基础URL (base_url)")
	fmt.Println("5. 模型名称 (model)")
	fmt.Println("p. 预览更改")
	fmt.Println("0. 完成编辑并保存")
	fmt.Println("q. 退出不保存")
	fmt.Println(strings.Repeat("-", 60))
}

func getUserChoice(reader *bufio.Reader) string {
	fmt.Print("\n请输入选择: ")
	choice, _ := reader.ReadString('\n')
	return strings.TrimSpace(choice)
}

func displayConfig(config config.APIConfig) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("当前配置: %s\n", config.Alias)
	fmt.Println(strings.Repeat("=", 60))

	// Use helper function to display field
	displayField("1. 别名", config.Alias, "")
	displayMaskedField("2. API密钥", config.APIKey, utils.MaskAPIKey(config.APIKey))
	displayMaskedField("3. 认证令牌", config.AuthToken, utils.MaskAPIKey(config.AuthToken))
	displayField("4. 基础URL", config.BaseURL, "https://api.anthropic.com (默认)")
	displayField("5. 模型名称", config.Model, "(未设置)")

	fmt.Println(strings.Repeat("=", 60))
}

func displayField(label, value, defaultValue string) {
	if value != "" {
		fmt.Printf("%s: %s\n", label, value)
	} else {
		fmt.Printf("%s: %s\n", label, defaultValue)
	}
}

func displayMaskedField(label, value, maskedValue string) {
	if value != "" {
		fmt.Printf("%s: %s\n", label, maskedValue)
	} else {
		fmt.Println(label + ": (未设置)")
	}
}

func handleFieldSelection(reader *bufio.Reader, currentConfig *config.APIConfig, updates map[string]string, choice string, configManager *config.ConfigManager) error {
	var fieldType FieldType
	var fieldName string

	// Validate choice and get field info
	if err := parseFieldChoice(choice, &fieldType, &fieldName); err != nil {
		return err
	}

	return editField(reader, currentConfig, updates, fieldType, fieldName, configManager)
}

// parseFieldChoice parses user choice and returns field information
func parseFieldChoice(choice string, fieldType *FieldType, fieldName *string) error {
	switch choice {
	case "1":
		*fieldType = FieldAlias
		*fieldName = "别名"
	case "2":
		*fieldType = FieldAPIKey
		*fieldName = "API密钥"
	case "3":
		*fieldType = FieldAuthToken
		*fieldName = "认证令牌"
	case "4":
		*fieldType = FieldBaseURL
		*fieldName = "基础URL"
	case "5":
		*fieldType = FieldModel
		*fieldName = "模型名称"
	default:
		return fmt.Errorf("无效选择，请输入 0-5、p 或 q")
	}
	return nil
}

// handlePreview displays preview of changes if any
func handlePreview(currentConfig *config.APIConfig, updates map[string]string) error {
	if len(updates) == 0 {
		return fmt.Errorf("还没有任何更改")
	}
	previewChanges(*currentConfig, updates)
	return nil
}

func editField(reader *bufio.Reader, currentConfig *config.APIConfig, updates map[string]string, fieldType FieldType, fieldName string, configManager *config.ConfigManager) error {
	// Get current value (either from updates or currentConfig)
	currentValue := getCurrentValue(currentConfig, updates, fieldType)
	prompt := fmt.Sprintf("\n当前%s: %s\n请输入新%s (回车保持不变): ", fieldName, currentValue, fieldName)
	fmt.Print(prompt)

	newValue, _ := reader.ReadString('\n')
	newValue = strings.TrimSpace(newValue)

	// No change
	if newValue == "" {
		fmt.Println("未更改")
		return nil
	}

	// Validate the new value
	if err := validateFieldValue(fieldType, newValue, currentConfig, configManager); err != nil {
		return err
	}

	// Store the update
	updateKey := getFieldKey(fieldType)
	updates[updateKey] = newValue

	// Show success message with masked value if sensitive
	if isSensitiveField(fieldType) {
		fmt.Printf("✓ %s将更新为: %s\n", fieldName, utils.MaskAPIKey(newValue))
	} else {
		fmt.Printf("✓ %s将更新为: %s\n", fieldName, newValue)
	}

	return nil
}

func getCurrentValue(config *config.APIConfig, updates map[string]string, fieldType FieldType) string {
	key := getFieldKey(fieldType)
	if val, ok := updates[key]; ok {
		return val
	}

	switch fieldType {
	case FieldAlias:
		return config.Alias
	case FieldAPIKey:
		return config.APIKey
	case FieldAuthToken:
		return config.AuthToken
	case FieldBaseURL:
		return config.BaseURL
	case FieldModel:
		return config.Model
	default:
		return ""
	}
}

func getFieldKey(fieldType FieldType) string {
	switch fieldType {
	case FieldAlias:
		return "alias"
	case FieldAPIKey:
		return "api_key"
	case FieldAuthToken:
		return "auth_token"
	case FieldBaseURL:
		return "base_url"
	case FieldModel:
		return "model"
	default:
		return ""
	}
}

func isSensitiveField(fieldType FieldType) bool {
	return fieldType == FieldAPIKey || fieldType == FieldAuthToken
}

func validateFieldValue(fieldType FieldType, value string, currentConfig *config.APIConfig, configManager *config.ConfigManager) error {
	switch fieldType {
	case FieldAlias:
		// Check if alias already exists (excluding current config)
		if value == currentConfig.Alias {
			return fmt.Errorf("新别名与当前别名相同")
		}
		if _, err := configManager.Get(value); err == nil {
			return fmt.Errorf("别名 '%s' 已存在", value)
		}
	case FieldBaseURL:
		// Validate URL format
		if _, err := url.ParseRequestURI(value); err != nil {
			return fmt.Errorf("无效的URL格式: %v", err)
		}
	case FieldAPIKey, FieldAuthToken:
		// Validate that at least one auth method is set
		otherAuth := getOtherAuthValue(fieldType, currentConfig, value)
		if otherAuth == "" && value == "" {
			return fmt.Errorf("API密钥和认证令牌不能同时为空")
		}
	}
	return nil
}

func getOtherAuthValue(fieldType FieldType, config *config.APIConfig, newValue string) string {
	if fieldType == FieldAPIKey {
		// After update, api_key will be newValue, check if auth_token is set
		return config.AuthToken
	} else {
		// After update, auth_token will be newValue, check if api_key is set
		return config.APIKey
	}
}

func previewChanges(currentConfig config.APIConfig, updates map[string]string) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("预览更改:")
	fmt.Println(strings.Repeat("=", 60))

	// Show each changed field
	if newAlias, ok := updates["alias"]; ok {
		fmt.Printf("别名: %s → %s\n", currentConfig.Alias, newAlias)
	}
	if newAPIKey, ok := updates["api_key"]; ok {
		fmt.Printf("API密钥: %s → %s\n", utils.MaskAPIKey(currentConfig.APIKey), utils.MaskAPIKey(newAPIKey))
	}
	if newAuthToken, ok := updates["auth_token"]; ok {
		fmt.Printf("认证令牌: %s → %s\n", utils.MaskAPIKey(currentConfig.AuthToken), utils.MaskAPIKey(newAuthToken))
	}
	if newBaseURL, ok := updates["base_url"]; ok {
		fmt.Printf("基础URL: %s → %s\n", currentConfig.BaseURL, newBaseURL)
	}
	if newModel, ok := updates["model"]; ok {
		fmt.Printf("模型名称: %s → %s\n", currentConfig.Model, newModel)
	}

	fmt.Println(strings.Repeat("=", 60))
}

func confirmSave(reader *bufio.Reader) bool {
	fmt.Print("\n确认保存更改? (y/N): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	return choice == "y" || choice == "Y"
}

func getUpdatedAlias(originalAlias string, updates map[string]string) string {
	if newAlias, ok := updates["alias"]; ok {
		return newAlias
	}
	return originalAlias
}

// saveAndApplyChanges applies updates to config and regenerates the active script
func saveAndApplyChanges(configManager *config.ConfigManager, alias string, updates map[string]string) error {
	// Apply field updates
	if err := applyUpdates(configManager, alias, updates); err != nil {
		return fmt.Errorf("保存失败: %v", err)
	}

	// Generate active.env script
	updatedAlias := getUpdatedAlias(alias, updates)
	if err := configManager.GenerateActiveScript(); err != nil {
		fmt.Fprintf(os.Stderr, "警告: 生成激活脚本失败: %v\n", err)
	}

	// Note: Active script regeneration is best-effort and doesn't fail the command
	_ = updatedAlias
	return nil
}

func applyUpdates(configManager *config.ConfigManager, alias string, updates map[string]string) error {
	// Handle alias update separately
	if newAlias, ok := updates["alias"]; ok {
		if err := configManager.RenameAlias(alias, newAlias); err != nil {
			return fmt.Errorf("重命名别名失败: %v", err)
		}
		alias = newAlias // Update alias for subsequent updates
	}

	// Remove alias from updates
	delete(updates, "alias")

	// Apply other field updates
	if len(updates) > 0 {
		if err := configManager.UpdatePartial(alias, updates); err != nil {
			return err
		}
	}

	return nil
}
