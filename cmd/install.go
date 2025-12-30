package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	forceInstall         bool
	noShellIntegration   bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install shell initialization script",
	Long:  "Add auto-load command to shell configuration file, so new terminals automatically load active configuration",
	Run: func(cmd *cobra.Command, args []string) {
		// Skip shell integration if flag is set
		if noShellIntegration {
			fmt.Println("Shell integration skipped (--no-shell-integration flag set)")
			return
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to get user home directory: %v\n", err)
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
			fmt.Fprintf(os.Stderr, "Error: Unsupported shell: %s\n", shell)
			fmt.Fprintf(os.Stderr, "Please manually add the following to your shell configuration file:\n")
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
					fmt.Fprintf(os.Stderr, "Error: Failed to read %s: %v\n", rcFile, err)
					os.Exit(1)
				}

				// Check for new version (with apimgr() function wrapper)
				if strings.Contains(string(content), "apimgr load-active") {
					if strings.Contains(string(content), "apimgr() {") {
						fmt.Printf("✓ Latest version already installed to %s\n", rcFile)
						fmt.Printf("\nTip: Run 'source %s' to take effect\n", rcFile)
						return
					}
					fmt.Printf("⚠️  Detected old version installation\n")
					fmt.Printf("Suggested to run 'apimgr install --force' to update to new version\n")
					fmt.Printf("Or manually update apimgr configuration in %s\n", rcFile)
					return
				}
			}
		} else {
			// Force install - remove old configuration
			if _, err := os.Stat(rcFile); err == nil {
				content, err := os.ReadFile(rcFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: Failed to read %s: %v\n", rcFile, err)
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
				err = os.WriteFile(rcFile, []byte(strings.Join(newLines, "\n")), 0600)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: Failed to update %s: %v\n", rcFile, err)
					os.Exit(1)
				}

				fmt.Printf("✓ Old configuration cleared\n")
			}
		}

		// Append to rc file
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to open %s: %v\n", rcFile, err)
			os.Exit(1)
		}
		defer f.Close()

		// Write the script to the file
		bytesWritten, err := f.WriteString(initScript)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write to %s: %v\n", rcFile, err)
			os.Exit(1)
		}

		// Close file explicitly to ensure content is flushed to disk
		err = f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to close file %s: %v\n", rcFile, err)
			os.Exit(1)
		}

		fmt.Printf("✓ Successfully installed to %s (%d bytes written)\n\n", rcFile, bytesWritten)
		fmt.Printf("Please run the following command to take effect:\n")
		fmt.Printf("  source %s\n\n", rcFile)
		fmt.Printf("Or reopen the terminal\n\n")
		fmt.Printf("After installation, you can directly use:\n")
		fmt.Printf("  apimgr switch <config_alias>  # Automatically switch and apply environment variables\n")
		fmt.Printf("  apimgr list               # List all configurations\n")
		fmt.Printf("  apimgr status             # View current configuration status\n")

		// Verify that the file was actually modified by checking if the script exists in the file
		updatedContent, err := os.ReadFile(rcFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to verify if %s was updated: %v\n", rcFile, err)
		} else if strings.Contains(string(updatedContent), "apimgr() {") {
			fmt.Printf("✓ Verification: Configuration successfully written to %s\n", rcFile)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Verification failed, configuration may not be correctly written to %s\n", rcFile)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&forceInstall, "force", "f", false, "Force reinstall, overwrite existing configuration")
	installCmd.Flags().BoolVar(&noShellIntegration, "no-shell-integration", false, "Skip shell integration (do not modify shell RC files)")
}
