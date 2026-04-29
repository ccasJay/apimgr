package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ccasJay/apimgr/config"
	"github.com/ccasJay/apimgr/config/models"
	"github.com/ccasJay/apimgr/internal/crypto"
	"github.com/spf13/cobra"
)

var (
	exportFile string
)

var exportCmd = &cobra.Command{
	Use:   "export [alias]",
	Short: "Export API configurations as encrypted string",
	Long: `Export API configurations as an encrypted Base64 string for cross-device sync.

Usage:
  apimgr export                    # Export all configurations
  apimgr export my-config          # Export single configuration by alias
  apimgr export --file output.txt  # Export to file (avoids for terminal input limits)

The encrypted string can be copied and imported on another device using:
  apimgr import

Or use file mode to avoid terminal input limits:
  apimgr export --file output.txt
  apimgr import --file output.txt

Security:
  - Password-based encryption using PBKDF2 and AES-256-GCM
  - Password must be at least 8 characters
  - Each export uses a random salt for unique encryption`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExport(args)
	},
}

func runExport(args []string) error {
	// Check for terminal support
	if !isTerminal() {
		return fmt.Errorf("interactive input required for export command")
	}

	reader := bufio.NewReader(os.Stdin)

	// Get password
	fmt.Print("Enter encryption password (min 8 characters): ")
	password, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	password = strings.TrimSpace(password)

	// Validate password
	if err := crypto.ValidatePassword(password); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}

	// Confirm password
	fmt.Print("Confirm encryption password: ")
	confirmPassword, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read confirmation password: %w", err)
	}
	confirmPassword = strings.TrimSpace(confirmPassword)

	if password != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	// Initialize config manager
	configManager, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// Get configurations to export
	var configs []models.APIConfig

	if len(args) == 1 {
		// Export single configuration
		alias := args[0]
		config, err := configManager.Get(alias)
		if err != nil {
			return fmt.Errorf("failed to get configuration '%s': %w", alias, err)
		}
		configs = []models.APIConfig{*config}
	} else {
		// Export all configurations
		configs, err = configManager.List()
		if err != nil {
			return fmt.Errorf("failed to list configurations: %w", err)
		}
	}

	if len(configs) == 0 {
		return fmt.Errorf("no configurations to export")
	}

	// Convert to export format
	exportConfigs := make([]crypto.ExportConfig, len(configs))
	for i, cfg := range configs {
		exportConfigs[i] = crypto.NewExportConfig(
			cfg.Alias,
			cfg.Provider,
			cfg.APIKey,
			cfg.AuthToken,
			cfg.BaseURL,
			cfg.Model,
			cfg.Models,
		)
	}

	export := crypto.NewExportFormat(exportConfigs)

	// Encrypt
	pm := crypto.NewPasswordManager(password)
	encrypted, err := pm.Encrypt(export)
	if err != nil {
		return fmt.Errorf("failed to encrypt configurations: %w", err)
	}

	// Display results
	displayExportResults(configs, encrypted, exportFile)

	return nil
}

// displayExportResults displays the export results in a formatted way
func displayExportResults(configs []models.APIConfig, encrypted string, file string) {
	width := 70
	border := strings.Repeat("=", width)

	fmt.Println("\n" + border)
	fmt.Println("Exported Configuration")
	fmt.Println(border)
	fmt.Printf("Version: %s\n", crypto.ExportVersionVersion())
	fmt.Printf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Configurations: %d\n", len(configs))

	fmt.Println("\nConfigurations:")
	for i, cfg := range configs {
		fmt.Printf("  %d. %s\n", i+1, cfg.Alias)
		fmt.Printf("     Provider: %s\n", cfg.Provider)
		fmt.Printf("     Model: %s\n", cfg.Model)
		if len(cfg.Models) > 1 {
			fmt.Printf("     Models: %s\n", strings.Join(cfg.Models, ", "))
		}
	}

	fmt.Println("\nEncrypted string (copy this):")
	fmt.Println(strings.Repeat("-", width))

	// Print encrypted string with word wrap for easier reading
	wrapped := wrapString(encrypted, width)
	fmt.Println(wrapped)

	fmt.Println(strings.Repeat("-", width))

	if file != "" {
		// Write to file
		if err := os.WriteFile(file, []byte(encrypted), 0600); err != nil {
			fmt.Printf("\n⚠️  Warning: Failed to write to file: %v\n", err)
		} else {
			fmt.Printf("\n✅ Exported to file: %s\n", file)
			fmt.Println("\nTo import on another device:")
			fmt.Printf("  apimgr import --file %s\n", file)
			return
		}
	}

	fmt.Println("\nTo import on another device:")
	fmt.Println("  apimgr import")
}

// wrapString wraps a string to a specified width
func wrapString(s string, width int) string {
	var result []string
	var line strings.Builder

	for _, r := range s {
		if line.Len() >= width {
			result = append(result, line.String())
			line.Reset()
		}
		line.WriteRune(r)
	}

	if line.Len() > 0 {
		result = append(result, line.String())
	}

	return strings.Join(result, "\n")
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&exportFile, "file", "f", "", "Export to file instead of stdout (avoids terminal input limits)")
}
