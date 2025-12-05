package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"apimgr/config"
	"apimgr/internal/compatibility"
	"apimgr/internal/providers"
	"apimgr/internal/utils"
	"github.com/spf13/cobra"
)

var (
	customURL     string
	outputJSON    bool
	requestMethod string
	timeout       time.Duration
	testRealAPI   bool   // Test real API functionality (simulate ClaudeCode usage)
	apiPath       string // Custom path for real API testing
	streamTest    bool   // Test streaming mode
	verboseOutput bool   // Verbose output
)

var pingCmd = &cobra.Command{
	Use:   "ping [alias]",
	Short: "Test API configuration connectivity",
	Long: `Test API configuration connectivity - support for multiple modes:

1. Test active configuration:
   apimgr ping

2. Test specific configuration:
   apimgr ping my-config

3. Test custom URL:
   apimgr ping -u https://api.example.com

4. Test real API compatibility with Claude Code:
   apimgr ping -T [alias]
   apimgr ping -T --stream [alias]  # Include streaming test
   apimgr ping -T -v [alias]        # Verbose output`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPingCommand,
}

func runPingCommand(cmd *cobra.Command, args []string) {
	configManager := config.NewConfigManager()

	// If -T flag is set, use the compatibility tester
	if testRealAPI {
		runCompatibilityTest(cmd, args, configManager)
		return
	}

	// Original ping logic for basic connectivity test
	runBasicConnectivityTest(cmd, args, configManager)
}

// runCompatibilityTest runs the full API compatibility test using the compatibility package
func runCompatibilityTest(cmd *cobra.Command, args []string, configManager *config.Manager) {
	var cfg *config.APIConfig
	var err error
	var alias string

	// Determine which configuration to test
	isCustomURL := cmd.Flags().Lookup("url").Changed
	if isCustomURL {
		fmt.Fprintf(os.Stderr, "❌ Error: -T flag cannot be used with custom URL (-u)\n")
		os.Exit(1)
	}

	if len(args) == 1 {
		alias = args[0]
		cfg, err = configManager.Get(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg, err = configManager.GetActive()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		alias = cfg.Alias
	}

	if !outputJSON {
		fmt.Printf("Testing API compatibility for: %s\n", alias)
	}

	// Create tester with options
	opts := []compatibility.TesterOption{
		compatibility.WithVerbose(verboseOutput),
	}
	if apiPath != "" {
		opts = append(opts, compatibility.WithCustomPath(apiPath))
	}

	tester, err := compatibility.NewTester(cfg, opts...)
	if err != nil {
		if outputJSON {
			errData, _ := json.Marshal(map[string]interface{}{
				"error":   err.Error(),
				"success": false,
			})
			fmt.Println(string(errData))
		} else {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Run the compatibility test
	result, err := tester.RunFullTest(streamTest)
	if err != nil {
		if outputJSON {
			errData, _ := json.Marshal(map[string]interface{}{
				"error":   err.Error(),
				"success": false,
			})
			fmt.Println(string(errData))
		} else {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		}
		os.Exit(1)
	}

	// Create reporter and output results
	reporter := compatibility.NewReporter(
		os.Stdout,
		compatibility.WithJSONOutput(outputJSON),
		compatibility.WithVerboseOutput(verboseOutput),
	)

	if err := reporter.Report(result); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error reporting results: %v\n", err)
		os.Exit(1)
	}

	// Determine exit code based on compatibility level
	_, exitCode := compatibility.DetermineCompatibilityLevel(result.Checks)
	os.Exit(exitCode)
}

// runBasicConnectivityTest runs the original basic connectivity test
func runBasicConnectivityTest(cmd *cobra.Command, args []string, configManager *config.Manager) {
	var baseURL string

	// Decide which URL to test
	isCustomURL := cmd.Flags().Lookup("url").Changed
	switch {
	case isCustomURL:
		// Custom URL mode
		baseURL = customURL
		fmt.Printf("Testing custom URL: %s\n", baseURL)

	case len(args) == 1:
		// Specific configuration mode
		alias := args[0]
		cfg, err := configManager.Get(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		baseURL = cfg.BaseURL
		// Apply URL normalization for configured API
		if cfg.Provider != "" {
			if provider, err := providers.Get(cfg.Provider); err == nil {
				baseURL = provider.NormalizeConfig(baseURL)
			}
		}
		fmt.Printf("Testing configuration: %s\n", alias)

	default:
		// Active configuration mode
		cfg, err := configManager.GetActive()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}
		baseURL = cfg.BaseURL
		// Apply URL normalization for configured API
		if cfg.Provider != "" {
			if provider, err := providers.Get(cfg.Provider); err == nil {
				baseURL = provider.NormalizeConfig(baseURL)
			}
		}
		fmt.Printf("Testing active configuration: %s\n", cfg.Alias)
	}

	// Ensure URL has default value
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
		fmt.Printf("⚠️  Note: Using default URL: %s\n", baseURL)
	}

	// Perform connectivity test
	start := time.Now()

	// Create optimized HTTP client (connection pooling + custom timeout)
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:          10,               // Maximum idle connections
			IdleConnTimeout:       30 * time.Second, // Idle connection timeout
			TLSHandshakeTimeout:   5 * time.Second,  // TLS handshake timeout
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Enhanced URL validation
	if !utils.ValidateURL(baseURL) {
		if outputJSON {
			errData, _ := json.Marshal(map[string]interface{}{
				"error":   "Invalid URL format",
				"url":     baseURL,
				"success": false,
			})
			fmt.Println(string(errData))
		} else {
			fmt.Fprintf(os.Stderr, "❌ Error: Invalid URL format: %s\n", baseURL)
			fmt.Fprintln(os.Stderr, "URL must include http or https protocol and valid hostname")
		}
		os.Exit(1)
	}

	// Add auth headers for configured API (not for custom URL mode)
	var cfg *config.APIConfig
	var apiErr error

	if !isCustomURL {
		// Get configuration
		if len(args) == 1 {
			cfg, apiErr = configManager.Get(args[0])
		} else {
			cfg, apiErr = configManager.GetActive()
		}
	}

	// Build final URL (add custom path)
	finalURL := baseURL
	if apiPath != "" {
		// If custom path provided, append to URL
		// Ensure no duplicate slashes
		if strings.HasSuffix(baseURL, "/") && strings.HasPrefix(apiPath, "/") {
			finalURL = baseURL + apiPath[1:]
		} else if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(apiPath, "/") {
			finalURL = baseURL + "/" + apiPath
		} else {
			finalURL = baseURL + apiPath
		}
	}

	// Determine request method and body
	finalMethod := requestMethod
	var requestBody io.Reader = nil
	var contentType string = ""

	// Create request
	req, err := http.NewRequest(finalMethod, finalURL, requestBody)
	if err != nil {
		if outputJSON {
			errData, _ := json.Marshal(map[string]interface{}{
				"error":   "failed to create request",
				"message": err.Error(),
				"success": false,
			})
			fmt.Println(string(errData))
		} else {
			fmt.Fprintf(os.Stderr, "❌ Error: Failed to create request: %v\n", err)
		}
		os.Exit(1)
	}

	// Set Content-Type header (if needed)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Add appropriate auth headers
	if !isCustomURL && apiErr == nil && cfg != nil {
		if cfg.AuthToken != "" {
			// For configs using AuthToken, use Bearer authentication
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AuthToken))
		} else if cfg.APIKey != "" {
			// For configs using APIKey, use Anthropic-style API-Key header
			req.Header.Set("x-api-key", cfg.APIKey)
			// Also support Anthropic format
			req.Header.Set("API-Key", cfg.APIKey)
		}
	}

	// Progress indicator
	if !outputJSON {
		fmt.Print("Connecting... ")
	}

	resp, err := client.Do(req)
	if err != nil {
		if !outputJSON {
			fmt.Printf("\r") // Clear progress indicator
		}

		// Categorize errors
		var errMsg string
		errStr := err.Error()

		// Check timeout first
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			errMsg = fmt.Sprintf("Request timed out (more than %ds)", int(timeout.Seconds()))
		} else if strings.Contains(errStr, "connection refused") {
			errMsg = "Connection refused (server not listening on this port)"
		} else if strings.Contains(errStr, "network is unreachable") {
			errMsg = "Network unreachable"
		} else if strings.Contains(errStr, "EOF") {
			errMsg = "Connection closed unexpectedly (server may not exist or be unresponsive)"
		} else if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "NXDOMAIN") {
			errMsg = "DNS resolution failed (domain does not exist or network configuration error)"
		} else if strings.Contains(errStr, "invalid URL") || strings.Contains(errStr, "parse error") {
			errMsg = "Invalid URL format"
		} else {
			// Other network errors
			if netErr, ok := err.(net.Error); ok {
				errMsg = fmt.Sprintf("Network error: %v", netErr)
			} else {
				errMsg = fmt.Sprintf("Request failed: %v", err)
			}
		}

		if outputJSON {
			errData, _ := json.Marshal(map[string]interface{}{
				"error":   errMsg,
				"url":     baseURL,
				"success": false,
			})
			fmt.Println(string(errData))
		} else {
			fmt.Fprintf(os.Stderr, "❌ Connection failed: %s\n", errMsg)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Clear progress indicator
	if !outputJSON {
		fmt.Printf("\r")
	}

	// Output result
	isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 300
	if outputJSON {
		result := map[string]interface{}{
			"url":           finalURL,
			"statusCode":    resp.StatusCode,
			"statusText":    http.StatusText(resp.StatusCode),
			"requestMethod": req.Method,
			"durationMs":    duration.Milliseconds(),
			"timeoutMs":     timeout.Milliseconds(),
			"success":       isSuccess,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("✅ Connection successful! \n")
		fmt.Printf("   URL: %s\n", finalURL)
		fmt.Printf("   Method: %s\n", req.Method)
		fmt.Printf("   Status Code: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		fmt.Printf("   Response Time: %dms\n", duration.Milliseconds())
		fmt.Printf("   Timeout Setting: %s\n", timeout)

		// Provide additional tips
		if !isSuccess {
			fmt.Printf("⚠️  Note: Server returned non-success status code\n")
			fmt.Printf("   - This is usually because the API's base URL doesn't support simple HEAD/GET requests\n")
			fmt.Printf("   - But the API's core functionality may still be available (e.g., POST requests used by ClaudeCode)\n")
			fmt.Printf("   - Try using this configuration in actual scenarios\n")
		}
	}
}

func init() {
	rootCmd.AddCommand(pingCmd)
	// Define flag and bind to variable
	pingCmd.Flags().StringVarP(&customURL, "url", "u", "", "Test custom URL")
	pingCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "JSON format output")
	pingCmd.Flags().StringVarP(&requestMethod, "method", "X", "HEAD", "Request method")
	pingCmd.Flags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "Request timeout")
	pingCmd.Flags().BoolVarP(&testRealAPI, "test", "T", false, "Test real API compatibility with Claude Code")
	pingCmd.Flags().StringVarP(&apiPath, "path", "p", "", "Custom endpoint path for API testing (e.g.: /v1/chat/completions)")
	pingCmd.Flags().BoolVar(&streamTest, "stream", false, "Include streaming test (use with -T)")
	pingCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "Verbose output (show request/response details)")
}
