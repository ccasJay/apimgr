package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	forceInstall bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "安装shell初始化脚本",
	Long:  "在shell配置文件中添加自动加载命令，使新终端自动加载活动配置",
	Run: func(cmd *cobra.Command, args []string) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 无法获取用户主目录: %v\n", err)
			os.Exit(1)
		}

		// Detect shell
		shell := os.Getenv("SHELL")
		var rcFile string
		
		if strings.Contains(shell, "zsh") {
			rcFile = filepath.Join(homeDir, ".zshrc")
		} else if strings.Contains(shell, "bash") {
			rcFile = filepath.Join(homeDir, ".bashrc")
		} else {
			fmt.Fprintf(os.Stderr, "错误: 不支持的shell: %s\n", shell)
			fmt.Fprintf(os.Stderr, "请手动添加以下内容到你的shell配置文件:\n")
			fmt.Fprintf(os.Stderr, "\nif command -v apimgr &> /dev/null; then\n")
			fmt.Fprintf(os.Stderr, "  eval \"$(apimgr load-active)\"\n")
			fmt.Fprintf(os.Stderr, "fi\n")
			os.Exit(1)
		}

		initScript := `
# apimgr - auto load active API configuration and shell integration
if command -v apimgr &> /dev/null; then
  # Auto-load active configuration on shell startup
  eval "$(command apimgr load-active)"
  
  # Wrap apimgr command to handle 'switch' automatically
  # This allows 'apimgr switch' to directly modify environment variables
  apimgr() {
    if [ "${1-}" = "switch" ]; then
      shift
      local __apimgr_output
      if ! __apimgr_output="$(command apimgr switch "$@")"; then
        return $?
      fi
      eval "$__apimgr_output"
      return $?
    else
      command apimgr "$@"
    fi
  }
fi
`

		// Check if already installed (unless force flag is set)
		if !forceInstall {
			if _, err := os.Stat(rcFile); err == nil {
				content, err := os.ReadFile(rcFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "错误: 无法读取 %s: %v\n", rcFile, err)
					os.Exit(1)
				}
				
				// Check for new version (with apimgr() function wrapper)
				if strings.Contains(string(content), "apimgr load-active") {
					if strings.Contains(string(content), "apimgr() {") {
						fmt.Printf("✓ 已安装最新版本到 %s\n", rcFile)
						fmt.Printf("\n提示: 运行 'source %s' 使其生效\n", rcFile)
						return
					} else {
						fmt.Printf("⚠️  检测到旧版本安装\n")
						fmt.Printf("建议运行 'apimgr install --force' 更新到新版本\n")
						fmt.Printf("或手动更新 %s 中的 apimgr 配置\n", rcFile)
						return
					}
				}
			}
		} else {
			// Force install - remove old configuration
			if _, err := os.Stat(rcFile); err == nil {
				content, err := os.ReadFile(rcFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "错误: 无法读取 %s: %v\n", rcFile, err)
					os.Exit(1)
				}
				
				// Remove old apimgr configuration
				lines := strings.Split(string(content), "\n")
				var newLines []string
				inApimgrBlock := false
				
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					
					// Start of apimgr block
					if strings.Contains(trimmed, "# apimgr") {
						inApimgrBlock = true
						continue
					}
					
					// Inside block
					if inApimgrBlock {
						// End of block (empty line or new section)
						if trimmed == "" || (trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t")) {
							if !strings.Contains(trimmed, "apimgr") && !strings.Contains(trimmed, "command -v") && !strings.Contains(trimmed, "eval") {
								inApimgrBlock = false
							}
						}
						
						// Skip lines in block
						if inApimgrBlock && (strings.Contains(line, "apimgr") || strings.Contains(line, "eval") || strings.Contains(line, "if command") || strings.Contains(line, "fi") || strings.Contains(line, "{") || strings.Contains(line, "}")) {
							continue
						}
					}
					
					newLines = append(newLines, line)
				}
				
				// Write back the cleaned content
				err = os.WriteFile(rcFile, []byte(strings.Join(newLines, "\n")), 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "错误: 无法更新 %s: %v\n", rcFile, err)
					os.Exit(1)
				}
				
				fmt.Printf("✓ 已清除旧配置\n")
			}
		}

		// Append to rc file
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 无法打开 %s: %v\n", rcFile, err)
			os.Exit(1)
		}
		defer f.Close()

		// Write the script to the file
		bytesWritten, err := f.WriteString(initScript)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 无法写入 %s: %v\n", rcFile, err)
			os.Exit(1)
		}
		
		// Close file explicitly to ensure content is flushed to disk
		err = f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 无法关闭文件 %s: %v\n", rcFile, err)
			os.Exit(1)
		}

		fmt.Printf("✓ 成功安装到 %s (写入 %d 字节)\n\n", rcFile, bytesWritten)
		fmt.Printf("请运行以下命令使其生效:\n")
		fmt.Printf("  source %s\n\n", rcFile)
		fmt.Printf("或者重新打开终端\n\n")
		fmt.Printf("安装后，你可以直接使用:\n")
		fmt.Printf("  apimgr switch <配置别名>  # 自动切换并应用环境变量\n")
		fmt.Printf("  apimgr list               # 列出所有配置\n")
		fmt.Printf("  apimgr status             # 查看当前配置状态\n")
		
		// Verify that the file was actually modified by checking if the script exists in the file
		updatedContent, err := os.ReadFile(rcFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "警告: 无法验证 %s 是否已更新: %v\n", rcFile, err)
		} else if strings.Contains(string(updatedContent), "apimgr() {") {
			fmt.Printf("✓ 验证: 配置已成功写入到 %s\n", rcFile)
		} else {
			fmt.Fprintf(os.Stderr, "警告: 验证失败，配置可能未正确写入到 %s\n", rcFile)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&forceInstall, "force", "f", false, "强制重新安装，覆盖现有配置")
}
