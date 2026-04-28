package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"apimgr/config"
	"apimgr/config/models"
	"apimgr/internal/crypto"
	"github.com/spf13/cobra"
)

var (
	importSkip      bool
	importOverwrite bool
	importDryRun    bool
	importFile      string
)

// readMultiLine reads multiple lines from stdin until an empty line is encountered
// Returns the concatenated lines with newlines removed
func readMultiLine(reader *bufio.Reader) (string, error) {
	var lines []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if len(lines) == 0 {
				return "", err
			}
			break
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(lines) > 0 {
			// Empty line signals end of input
			break
		}
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	return strings.Join(lines, ""), nil
}

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import API configurations from encrypted string",
	Long: `Import API configurations from an encrypted Base64 string for cross-device sync.

Usage:
  apimgr import                        # Interactive mode (prompt for conflicts)
  apimgr import --skip                  # Skip existing configurations
  apimgr import --overwrite              # Overwrite existing configurations
  apimgr import --dry-run               # Preview without importing
  apimgr import --file exported.txt     # Import from file (avoids terminal input limits)

The encrypted string is obtained using:
  apimgr export [alias]

Or use file mode to avoid terminal input limits:
  apimgr export --file output.txt
  apimgr import --file output.txt

Conflict Resolution:
  --skip      Skip configurations that already exist (default in interactive mode)
  --overwrite Overwrite existing configurations
  --dry-run   Preview what would be imported without making changes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runImport()
	},
}

func runImport() error {
	// Check for terminal support
	if !isTerminal() && importFile == "" {
		return fmt.Errorf("interactive input required for import command")
	}

	reader := bufio.NewReader(os.Stdin)

	// Get encrypted string
	var encrypted string
	var err error

	if importFile != "" {
		// Read from file
		data, err := os.ReadFile(importFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		encrypted = strings.Trim(string(data), "\n\r\t ")
	} else {
		// Read from stdin (interactive mode)
		fmt.Print("Paste encrypted string (press Enter twice to finish): ")
		encrypted, err = readMultiLine(reader)
		if err != nil {
			return fmt.Errorf("failed to read encrypted string: %w", err)
		}
	}

	if encrypted == "" {
		return fmt.Errorf("encrypted string cannot be empty")
	}

	// Get password
	fmt.Print("Enter encryption password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password = strings.TrimSpace(password)

	// Validate password
	if err := crypto.ValidatePassword(password); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	// Decrypt
	pm := crypto.NewPasswordManager(password)
	decrypted, err := pm.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt configurations: %w", err)
	}

	// Initialize config manager
	configManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// Get existing configurations
	existingConfigs, err := configManager.List()
	if err != nil {
		return fmt.Errorf("failed to list existing configurations: %w", err)
	}

	// Build map of existing config aliases
	existingMap := make(map[string]bool)
	for _, cfg := range existingConfigs {
		existingMap[cfg.Alias] = true
	}

	// Display import preview
	displayImportPreview(decrypted, existingMap)

	// In dry-run mode, just show preview and exit
	if importDryRun {
		fmt.Println("\n🔍 Dry-run mode: No changes made")
		return nil
	}

	// Handle conflicts
	if importSkip || importOverwrite {
		// Automatic mode
		fmt.Println()
	} else {
		// Interactive mode - ask to proceed
		proceed := promptProceed()
		if !proceed {
			fmt.Println("Import cancelled")
			return nil
		}

		// If there are conflicts, ask how to handle
		conflicts := findConflicts(decrypted, existingMap)
		if len(conflicts) > 0 {
			overwrite := promptOverwrite(conflicts)
			if overwrite {
				importOverwrite = true
			} else {
				importSkip = true
			}
		}
	}

	// Import configurations
	imported, skipped, overwritten, err := importConfigs(
		configManager,
		decrypted,
		existingMap,
		importSkip,
		importOverwrite,
	)
	if err != nil {
		return fmt.Errorf("failed to import configurations: %w", err)
	}

	// Generate active script
	if err := configManager.GenerateActiveScript(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to generate activation script: %v\n", err)
	}

	// Display import summary
	displayImportSummary(imported, skipped, overwritten)

	return nil
}

// displayImportPreview displays a preview of the configurations to import
func displayImportPreview(export *crypto.ExportFormat, existingMap map[string]bool) {
	width := 70
	border := strings.Repeat("=", width)

	fmt.Println("\n" + border)
	fmt.Println("Import Preview")
	fmt.Println(border)
	fmt.Printf("Version: %s\n", export.Version)
	fmt.Printf("Configurations: %d\n", len(export.Configs))

	if len(export.Configs) > 0 {
		fmt.Println("\nConfigurations to import:")
		for i, cfg := range export.Configs {
			status := ""
			if existingMap[cfg.Alias] {
				status = " (exists)"
			}
			fmt.Printf("%d. %s%s\n", i+1, cfg.Alias, status)
			fmt.Printf("   Provider: %s\n", cfg.Provider)
			fmt.Printf("   Model: %s\n", cfg.Model)
			if len(cfg.Models) > 1 {
				fmt.Printf("   Models: %s\n", strings.Join(cfg.Models, ", "))
			}
			fmt.Printf("   Base URL: %s\n", cfg.BaseURL)
			if cfg.APIKey != "" {
				maskedKey := maskCredential(cfg.APIKey)
				fmt.Printf("   API Key: %s\n", maskedKey)
			}
			if cfg.AuthToken != "" {
				maskedToken := maskCredential(cfg.AuthToken)
				fmt.Printf("   Auth Token: %s\n", maskedToken)
			}
		}
	}
	fmt.Println(border)
}

// maskCredential masks a credential for display
func maskCredential(credential string) string {
	if len(credential) <= 8 {
		return "********"
	}
	return credential[:4] + "..." + credential[len(credential)-4:]
}

// findConflicts returns a list of configuration aliases that already exist
func findConflicts(export *crypto.ExportFormat, existingMap map[string]bool) []string {
	var conflicts []string
	for _, cfg := range export.Configs {
		if existingMap[cfg.Alias] {
			conflicts = append(conflicts, cfg.Alias)
		}
	}
	return conflicts
}

// promptProceed prompts the user to proceed with import
func promptProceed() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nProceed with import? (y/N): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// promptOverwrite prompts the user to overwrite existing configurations
func promptOverwrite(conflicts []string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n⚠️  The following configurations already exist:\n")
	for _, c := range conflicts {
		fmt.Printf("   - %s\n", c)
	}
	fmt.Print("Overwrite existing configurations? (y/N): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// importConfigs imports configurations according to the specified mode
func importConfigs(
	configManager *config.Manager,
	export *crypto.ExportFormat,
	existingMap map[string]bool,
	skip, overwrite bool,
) (imported, skipped, overwritten int, err error) {
	for _, exportCfg := range export.Configs {
		// Convert to APIConfig
		cfg := models.APIConfig{
			Alias:     exportCfg.Alias,
			Provider:  exportCfg.Provider,
			APIKey:    exportCfg.APIKey,
			AuthToken: exportCfg.AuthToken,
			BaseURL:   exportCfg.BaseURL,
			Model:     exportCfg.Model,
			Models:    exportCfg.Models,
		}

		if existingMap[cfg.Alias] {
			// Configuration already exists
			if skip {
				skipped++
				continue
			}
			if overwrite {
				// Update existing configuration
				if err := configManager.Add(cfg); err != nil {
					return imported, skipped, overwritten, err
				}
				overwritten++
				fmt.Printf("  ✅ Updated %s\n", cfg.Alias)
				continue
			}
			// Default: skip
			skipped++
			continue
		}

		// New configuration
		if err := configManager.Add(cfg); err != nil {
			return imported, skipped, overwritten, err
		}
		imported++
		fmt.Printf("  ✅ Imported %s\n", cfg.Alias)
	}

	return imported, skipped, overwritten, nil
}

// displayImportSummary displays a summary of the import operation
func displayImportSummary(imported, skipped, overwritten int) {
	fmt.Println("\n======================================================================")
	fmt.Println("Import Summary")
	fmt.Println("======================================================================")
	fmt.Printf("  Imported: %d\n", imported)
	fmt.Printf("  Skipped: %d\n", skipped)
	fmt.Printf("  Overwritten: %d\n", overwritten)
	fmt.Println("======================================================================")

	if imported > 0 || overwritten > 0 {
		fmt.Println("\n💡 Tip: Run 'apimgr switch <alias>' to switch to a configuration")
	}
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().BoolVar(&importSkip, "skip", false, "Skip existing configurations")
	importCmd.Flags().BoolVar(&importOverwrite, "overwrite", false, "Overwrite existing configurations")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Preview without importing")
	importCmd.Flags().StringVarP(&importFile, "file", "f", "", "Import from file instead of stdin (avoids terminal input limits)")
}
