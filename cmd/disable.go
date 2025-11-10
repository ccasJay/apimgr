package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	purgeConfig bool
)

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable automatic configuration application",
	Long:  `Disable automatic configuration application by stopping the daemon and removing shell integration.`,
	Run:   runDisable,
}

func init() {
	rootCmd.AddCommand(disableCmd)
	disableCmd.Flags().BoolVar(&purgeConfig, "purge", false, "Also remove all configuration files")
}

func runDisable(cmd *cobra.Command, args []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
		os.Exit(1)
	}

	uid := os.Getuid()
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}
	runtimeDir = filepath.Join(runtimeDir, fmt.Sprintf("apimgr-%d", uid))

	configDir := filepath.Join(homeDir, ".config", "apimgr")
	pidPath := filepath.Join(runtimeDir, "daemon.pid")
	socketPath := filepath.Join(runtimeDir, "apimgr.sock")

	// Step 1: Stop daemon if running
	fmt.Println("🛑 Stopping daemon...")
	if data, err := os.ReadFile(pidPath); err == nil {
		var pid int
		if _, err := fmt.Sscanf(string(data), "%d", &pid); err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				if err := process.Signal(syscall.SIGTERM); err == nil {
					fmt.Printf("✅ Daemon stopped (PID: %d)\n", pid)
				}
			}
		}
	}

	// Clean up runtime files
	os.Remove(pidPath)
	os.Remove(socketPath)
	os.RemoveAll(runtimeDir)

	// Step 2: Remove shell integration from shell configs
	fmt.Println("📝 Removing shell integration...")
	shellRcFiles := []string{
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".bashrc"),
		filepath.Join(homeDir, ".bash_profile"),
	}

	for _, rcFile := range shellRcFiles {
		if data, err := os.ReadFile(rcFile); err == nil {
			content := string(data)
			if strings.Contains(content, "apimgr/shell-integration.sh") {
				// Remove the integration line
				lines := strings.Split(content, "\n")
				var newLines []string
				for _, line := range lines {
					if !strings.Contains(line, "apimgr/shell-integration.sh") {
						newLines = append(newLines, line)
					}
				}
				newContent := strings.Join(newLines, "\n")
				
				if err := os.WriteFile(rcFile, []byte(newContent), 0644); err == nil {
					fmt.Printf("✅ Removed shell integration from %s\n", rcFile)
				}
			}
		}
	}

	// Step 3: Optionally purge configuration
	if purgeConfig {
		fmt.Println("🗑️  Purging configuration files...")
		if err := os.RemoveAll(configDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to remove config directory: %v\n", err)
		} else {
			fmt.Printf("✅ Removed configuration directory: %s\n", configDir)
		}
		
		// Also remove old config if exists
		oldConfigPath := filepath.Join(homeDir, ".apimgr.json")
		if err := os.Remove(oldConfigPath); err == nil {
			fmt.Printf("✅ Removed old configuration file: %s\n", oldConfigPath)
		}
	} else {
		// Just remove shell integration script
		shellIntegrationPath := filepath.Join(configDir, "shell-integration.sh")
		if err := os.Remove(shellIntegrationPath); err == nil {
			fmt.Printf("✅ Removed shell integration script\n")
		}
	}

	fmt.Println("\n✨ apimgr has been disabled")
	if !purgeConfig {
		fmt.Println("ℹ️  Your configuration files have been preserved")
		fmt.Println("   To completely remove all data, run: apimgr disable --purge")
	}
	fmt.Println("\nPlease restart your terminal for changes to take effect")
}
