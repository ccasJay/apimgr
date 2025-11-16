package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"apimgr/config"
	"github.com/spf13/cobra"
)

// APIConfigBuilder is responsible for building and validating APIConfig
type APIConfigBuilder struct {
	config *config.APIConfig
}

// NewAPIConfigBuilder creates a new builder
func NewAPIConfigBuilder() *APIConfigBuilder {
	return &APIConfigBuilder{
		config: &config.APIConfig{},
	}
}

// SetAlias sets the alias
func (b *APIConfigBuilder) SetAlias(alias string) *APIConfigBuilder {
	b.config.Alias = alias
	return b
}

// SetAPIKey sets the API key
func (b *APIConfigBuilder) SetAPIKey(apiKey string) *APIConfigBuilder {
	b.config.APIKey = apiKey
	return b
}

// SetAuthToken sets the auth token
func (b *APIConfigBuilder) SetAuthToken(authToken string) *APIConfigBuilder {
	b.config.AuthToken = authToken
	return b
}

// SetBaseURL sets the base URL
func (b *APIConfigBuilder) SetBaseURL(url string) *APIConfigBuilder {
	b.config.BaseURL = url
	return b
}

// SetModel sets the model
func (b *APIConfigBuilder) SetModel(model string) *APIConfigBuilder {
	b.config.Model = model
	return b
}

// Build builds the config
func (b *APIConfigBuilder) Build() (*config.APIConfig, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	return b.config, nil
}

// validate validates the config
func (b *APIConfigBuilder) validate() error {
	if b.config.Alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if b.config.APIKey == "" && b.config.AuthToken == "" {
		return fmt.Errorf("API key and auth token cannot both be empty")
	}
	if b.config.BaseURL != "" {
		if _, err := url.ParseRequestURI(b.config.BaseURL); err != nil {
			return fmt.Errorf("invalid URL format: %s", b.config.BaseURL)
		}
	}
	return nil
}

// InputCollector is responsible for collecting user input
type InputCollector struct{}

// isTerminal checks if running in a real terminal
func isTerminal() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// CollectInteractively collects input interactively
func (ic *InputCollector) CollectInteractively(presetType string) (*config.APIConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter config alias: ")
	alias, _ := reader.ReadString('\n')
	alias = strings.TrimSpace(alias)

	var apiKey, authToken, url, model string

	// Handle based on preset type
	switch presetType {
	case "api_key":
		// API key was provided via command line
		fmt.Print("Enter auth token (optional): ")
		authToken, _ = reader.ReadString('\n')
		authToken = strings.TrimSpace(authToken)
	case "auth_token":
		// Auth token was provided via command line
		fmt.Print("Enter API key (optional): ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
	default:
		// Fully interactive
		fmt.Print("Enter API key (optional, either api_key or auth_token is required): ")
		apiKey, _ = reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		fmt.Print("Enter auth token (optional, either api_key or auth_token is required): ")
		authToken, _ = reader.ReadString('\n')
		authToken = strings.TrimSpace(authToken)
	}

	// Validate at least one authentication method
	if apiKey == "" && authToken == "" {
		return nil, fmt.Errorf("must provide either API key or auth token")
	}

	fmt.Print("Enter API base URL (optional, default https://api.anthropic.com): ")
	url, _ = reader.ReadString('\n')
	url = strings.TrimSpace(url)
	if url == "" {
		url = "https://api.anthropic.com"
	}

	fmt.Print("Enter model name (optional): ")
	model, _ = reader.ReadString('\n')
	model = strings.TrimSpace(model)

	// Use builder to create config
	builder := NewAPIConfigBuilder().
		SetAlias(alias).
		SetAPIKey(apiKey).
		SetAuthToken(authToken).
		SetBaseURL(url).
		SetModel(model)

	return builder.Build()
}

var addCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "Add a new API configuration",
	Long: `Add a new API configuration - supports multiple modes:

1. Fully interactive:
   apimgr add

2. Quick command line add:
   apimgr add my-config --sk sk-xxx --url https://api.anthropic.com --model claude-3
   apimgr add my-config --ak bearer-token -u https://api.anthropic.com -m claude-3

3. Preset mode (has preset but missing alias):
   apimgr add --sk sk-xxx -u https://api.anthropic.com -m claude-3
   apimgr add --ak bearer-token`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()
		collector := &InputCollector{}

		// Determine input mode
		var cfg *config.APIConfig
		var err error

		hasSK := cmd.Flags().Lookup("sk").Changed
		hasAK := cmd.Flags().Lookup("ak").Changed
		hasAlias := len(args) == 1

		switch {
		case hasAlias:
			// Command line mode - has alias and parameters
			alias := args[0]
			apiKey, _ := cmd.Flags().GetString("sk")
			authToken, _ := cmd.Flags().GetString("ak")
			url, _ := cmd.Flags().GetString("url")
			model, _ := cmd.Flags().GetString("model")

			// Set default value
			if url == "" {
				url = "https://api.anthropic.com"
			}

			// Validate at least one authentication method
			if apiKey == "" && authToken == "" {
				fmt.Println("❌ Error: Must provide either --sk or --ak parameter")
				fmt.Println("\n💡 Usage examples:")
				fmt.Println("  apimgr add my-config --sk sk-xxx")
				fmt.Println("  apimgr add my-config --ak token-xxx")
				os.Exit(1)
			}

			builder := NewAPIConfigBuilder().
				SetAlias(alias).
				SetAPIKey(apiKey).
				SetAuthToken(authToken).
				SetBaseURL(url).
				SetModel(model)

			cfg, err = builder.Build()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				os.Exit(1)
			}

		case hasSK || hasAK:
			// Preset mode - has preset parameters but no alias, enter interactive
			presetType := ""
			if hasSK {
				presetType = "api_key"
			} else {
				presetType = "auth_token"
			}

			if !isTerminal() {
				fmt.Println("❌ Interactive input is not supported in the current environment, please provide an alias:")
				fmt.Printf("  apimgr add <alias> --%s <value> [--url <url>] [--model <model>]\n",
					map[bool]string{true: "sk", false: "ak"}[hasSK])
				os.Exit(1)
			}

			cfg, err = collector.CollectInteractively(presetType)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

		default:
			// Fully interactive mode
			if !isTerminal() {
				fmt.Println("❌ Interactive input is not supported in the current environment")
				fmt.Printf("  apimgr add <alias> --sk <key> [--url <url>] [--model <model>]\n")
				os.Exit(1)
			}

			cfg, err = collector.CollectInteractively("")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

		// Save the configuration
		err = configManager.Add(*cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to save configuration: %v\n", err)
			os.Exit(1)
		}

		// Generate active script
		if err := configManager.GenerateActiveScript(); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to generate activation script: %v\n", err)
		}

		fmt.Printf("✅ Configuration added: %s\n", cfg.Alias)
		fmt.Println("\n💡 Tip: Run 'apimgr switch <alias>' to switch to this configuration")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("url", "u", "", "API base URL")
	addCmd.Flags().StringP("model", "m", "", "Model name")
	addCmd.Flags().String("sk", "", "API key (ANTHROPIC_API_KEY)")
	addCmd.Flags().String("ak", "", "Auth token (ANTHROPIC_AUTH_TOKEN)")
}
