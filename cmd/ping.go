package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"apimgr/config"
	"apimgr/internal/providers"
	"github.com/spf13/cobra"
)

var (
	customURL     string
	outputJSON    bool
	requestMethod string
	timeout       time.Duration
	testRealAPI   bool   // Test real API functionality (simulate ClaudeCode usage)
	apiPath       string // Custom path for real API testing
)

// Enhanced URL validation: Check protocol and hostname
func isValidURL(u string) bool {
	parsed, err := url.ParseRequestURI(u)
	if err != nil {
		return false
	}
	// Ensure protocol is http or https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	// Ensure hostname exists
	if parsed.Host == "" {
		return false
	}
	return true
}

var pingCmd = &cobra.Command{
	Use:   "ping [alias]",
	Short: "Test API configuration connectivity",
	Long: `Test API configuration connectivity - support for multiple modes:

1. Test active configuration:
   apimgr ping

2. Test specific configuration:
   apimgr ping my-config

3. Test custom URL:
   apimgr ping -u https://api.example.com`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()

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

		// 增强URL验证
		if !isValidURL(baseURL) {
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

		// 为配置的API添加认证头（自定义URL模式不添加）
		var cfg *config.APIConfig
		var apiErr error

		if !isCustomURL {
			// 获取配置
			if len(args) == 1 {
				cfg, apiErr = configManager.Get(args[0])
			} else {
				cfg, apiErr = configManager.GetActive()
			}
		}

		// 构建最终URL（添加自定义路径）
		finalURL := baseURL
		if testRealAPI && apiPath != "" {
			// 如果有自定义路径，添加到URL
			// 确保没有重复的斜杠
			if strings.HasSuffix(baseURL, "/") && strings.HasPrefix(apiPath, "/") {
				finalURL = baseURL + apiPath[1:]
			} else if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(apiPath, "/") {
				finalURL = baseURL + "/" + apiPath
			} else {
				finalURL = baseURL + apiPath
			}
		}

		// 决定要使用的Request method和体
		finalMethod := requestMethod
		var requestBody io.Reader = nil
		var contentType string = ""

		// 如果是真实API测试，准备POST请求
		if testRealAPI && !isCustomURL && apiErr == nil && cfg != nil {
			finalMethod = "POST"
			contentType = "application/json"

			// 使用默认模型或配置中的模型
			model := cfg.Model
			if model == "" {
				model = "doubao-seed-code-preview-latest" // 默认Doubao模型
			}

			// 创建简单的请求体（模拟ClaudeCode的使用方式）
			// 使用标准的聊天API格式
			reqBody := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"ping test"}]}`, model)
			requestBody = strings.NewReader(reqBody)
		}

		// 创建请求
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

		// 设置Content-Type头（如果需要）
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		// 添加适当的认证头
		if !isCustomURL && apiErr == nil && cfg != nil {
			if cfg.AuthToken != "" {
				// 对于使用AuthToken的配置，使用Bearer认证
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AuthToken))
			} else if cfg.APIKey != "" {
				// 对于使用APIKey的配置，使用Anthropic风格的API-Key头
				req.Header.Set("x-api-key", cfg.APIKey)
				// 同时支持Anthropic的格式
				req.Header.Set("API-Key", cfg.APIKey)
			}
		}


		// 进度指示器
		if !outputJSON {
			fmt.Print("Connecting... ")
		}

		resp, err := client.Do(req)
		if err != nil {
			if !outputJSON {
				fmt.Printf("\r") // 清除进度指示器
			}

			// 分类处理错误
			var errMsg string
			errStr := err.Error()

			// 先检查超时情况
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
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
	// Define flag and bind to variable
	pingCmd.Flags().StringVarP(&customURL, "url", "u", "", "Test custom URL")
	pingCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "JSON format output")
	pingCmd.Flags().StringVarP(&requestMethod, "method", "X", "HEAD", "Request method")
	pingCmd.Flags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "Request timeout")
	pingCmd.Flags().BoolVarP(&testRealAPI, "test", "T", false, "Test real API functionality (simulate ClaudeCode usage)")
	pingCmd.Flags().StringVarP(&apiPath, "path", "p", "", "Custom path for real API testing (e.g.: /chat/completions)")
}
