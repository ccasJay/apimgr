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
		
		// Set the active configuration
		err := configManager.SetActive(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

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