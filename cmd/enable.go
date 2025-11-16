package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"apimgr/config"
	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable automatic configuration application",
	Long: `Enable automatic configuration application by setting up:
- XDG-compliant directory structure
- Configuration file migration
- Shell integration
- Active environment file generation`,
	Run: runEnable,
}

func init() {
	rootCmd.AddCommand(enableCmd)
}

func runEnable(cmd *cobra.Command, args []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(homeDir, ".config", "apimgr")
	oldConfigPath := filepath.Join(homeDir, ".apimgr.json")
	newConfigPath := filepath.Join(configDir, "config.json")
	activeEnvPath := filepath.Join(configDir, "active.env")

	// Step 1: Create XDG directory structure
	fmt.Println("📁 Creating XDG-compliant directory structure...")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create config directory: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Migrate configuration if needed
	if _, err := os.Stat(oldConfigPath); err == nil {
		if _, err := os.Stat(newConfigPath); os.IsNotExist(err) {
			fmt.Printf("📦 Migrating configuration from %s to %s...\n", oldConfigPath, newConfigPath)

			data, err := os.ReadFile(oldConfigPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to read old config: %v\n", err)
				os.Exit(1)
			}

			if err := os.WriteFile(newConfigPath, data, 0600); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to write new config: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Configuration migrated successfully")
			fmt.Printf("   You can safely remove the old config file: rm %s\n", oldConfigPath)
		} else {
			fmt.Println("ℹ️  Configuration already exists at new location")
		}
	} else {
		// Create empty config if neither exists
		if _, err := os.Stat(newConfigPath); os.IsNotExist(err) {
			defaultConfig := `{"active":"","configs":[]}`
			if err := os.WriteFile(newConfigPath, []byte(defaultConfig), 0600); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to create config file: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✅ Created new configuration file")
		}
	}

	// Step 3: Create initial active.env if config exists
	fmt.Println("🔧 Setting up configuration...")
	if _, err := os.Stat(newConfigPath); err == nil {
		// Load config and generate active.env
		configManager := config.NewConfigManager()
		if err := configManager.GenerateActiveScript(); err == nil {
			fmt.Printf("✅ Configuration ready at %s\n", newConfigPath)
		}
	}

	// Step 4: Check shell configuration
	fmt.Println("\n📝 Checking shell configuration...")
	shellRcFiles := []string{
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".bash_profile"),
	}

	integrationLine := fmt.Sprintf("[[ -f %s ]] && source %s", activeEnvPath, activeEnvPath)
	shellConfigured := false

	for _, rcFile := range shellRcFiles {
		if data, err := os.ReadFile(rcFile); err == nil {
			if strings.Contains(string(data), "apimgr/active.env") {
				fmt.Printf("✅ Shell integration already configured in %s\n", rcFile)
				shellConfigured = true
				break
			}
		}
	}

	if !shellConfigured {
		fmt.Println("\n⚠️  Shell integration not configured. Add this line to your shell config:")
		fmt.Printf("\n    %s\n\n", integrationLine)

		// Detect current shell
		shell := os.Getenv("SHELL")
		if strings.Contains(shell, "zsh") {
			fmt.Println("For Zsh, add to ~/.zshrc:")
			fmt.Printf("    echo '%s' >> ~/.zshrc\n", integrationLine)
		} else if strings.Contains(shell, "bash") {
			fmt.Println("For Bash, add to ~/.bashrc:")
			fmt.Printf("    echo '%s' >> ~/.bashrc\n", integrationLine)
		}
	}

	// Step 5: Instructions
	fmt.Println("\n✨ Setup complete! Next steps:")
	fmt.Println("1. If not done already, add the shell integration line to your shell config")
	fmt.Println("2. Restart your terminal or run: source ~/.zshrc (or ~/.bashrc)")
	fmt.Println("3. Use 'apimgr add' to add API configurations")
	fmt.Println("4. Use 'apimgr switch' to switch between configurations")
	fmt.Println("5. Configuration changes automatically apply to new terminal sessions")
	fmt.Println("\nTo verify the setup, run: apimgr status")
}
