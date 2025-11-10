package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"apimgr/config"
	"apimgr/internal/utils"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var editCmd = &cobra.Command{
	Use:   "edit <alias>",
	Short: "交互式编辑配置",
	Long: `交互式编辑已保存的API配置

此命令将以交互式界面引导您编辑配置的各个字段`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]
		if err := editConfig(alias); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
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

	// Interactive editing loop
	reader := bufio.NewReader(os.Stdin)
	updates := make(map[string]string)

	for {
		showMenu(len(updates))
		choice := getUserChoice(reader)

		// Handle special cases
		if choice == "q" || choice == "Q" {
			fmt.Println("\n已取消编辑，未保存更改")
			return nil
		}

		if choice == "p" || choice == "P" {
			if len(updates) == 0 {
				fmt.Println("\n⚠️  还没有任何更改")
				continue
			}
			previewChanges(*currentConfig, updates)
			continue
		}

		if choice == "0" {
			if len(updates) == 0 {
				fmt.Println("\n没有更改，跳过保存")
				return nil
			}
			if !confirmSave(reader) {
				fmt.Println("\n已取消保存")
				return nil
			}
			break
		}

		// Process field selection
		if err := handleFieldSelection(reader, currentConfig, updates, choice, configManager); err != nil {
			fmt.Printf("\n❌ %v\n", err)
		}
	}

	// Save changes
	if err := applyUpdates(configManager, alias, updates); err != nil {
		return fmt.Errorf("保存失败: %v", err)
	}
	fmt.Printf("\n✅ 配置 '%s' 已更新\n", getUpdatedAlias(alias, updates))
	return nil
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

	switch choice {
	case "1":
		fieldType = FieldAlias
		fieldName = "别名"
	case "2":
		fieldType = FieldAPIKey
		fieldName = "API密钥"
	case "3":
		fieldType = FieldAuthToken
		fieldName = "认证令牌"
	case "4":
		fieldType = FieldBaseURL
		fieldName = "基础URL"
	case "5":
		fieldType = FieldModel
		fieldName = "模型名称"
	default:
		return fmt.Errorf("无效选择，请输入 0-5、p 或 q")
	}

	return editField(reader, currentConfig, updates, fieldType, fieldName, configManager)
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
