package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"apimgr/config"
)

var (
	customURL     string
	outputJSON    bool
	requestMethod string
	timeout       time.Duration
)

// 增强URL验证：检查协议和主机名
func isValidURL(u string) bool {
	parsed, err := url.ParseRequestURI(u)
	if err != nil {
		return false
	}
	// 确保协议是http或https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	// 确保主机名存在
	if parsed.Host == "" {
		return false
	}
	return true
}

var pingCmd = &cobra.Command{
	Use:   "ping [alias]",
	Short: "测试API配置的连通性",
	Long: `测试API配置的连通性 - 支持多种模式：

1. 测试活动配置:
   apimgr ping

2. 测试特定配置:
   apimgr ping my-config

3. 测试自定义URL:
   apimgr ping -u https://api.example.com`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configManager := config.NewConfigManager()

		var baseURL string

		// 决定要测试的URL
		isCustomURL := cmd.Flags().Lookup("url").Changed
		switch {
		case isCustomURL:
			// 自定义URL模式
			baseURL = customURL
			fmt.Printf("正在测试自定义URL: %s\n", baseURL)

		case len(args) == 1:
			// 特定配置模式
			alias := args[0]
			cfg, err := configManager.Get(alias)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ 错误: %v\n", err)
				os.Exit(1)
			}
			baseURL = cfg.BaseURL
			fmt.Printf("正在测试配置: %s\n", alias)

		default:
			// 活动配置模式
			cfg, err := configManager.GetActive()
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ 错误: %v\n", err)
				os.Exit(1)
			}
			baseURL = cfg.BaseURL
			fmt.Printf("正在测试活动配置: %s\n", cfg.Alias)
		}

		// 确保URL有默认值
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
			fmt.Printf("⚠️  注意: 使用默认URL: %s\n", baseURL)
		}

		// 执行连通性测试
		start := time.Now()

		// 创建优化的HTTP客户端（连接池 + 自定义超时）
		client := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,          // 最大空闲连接数
				IdleConnTimeout:     30 * time.Second, // 空闲连接超时
				TLSHandshakeTimeout: 5 * time.Second,  // TLS握手超时
				ExpectContinueTimeout: 1 * time.Second,
			},
		}

		// 增强URL验证
		if !isValidURL(baseURL) {
			if outputJSON {
				errData, _ := json.Marshal(map[string]interface{}{
					"error":     "无效的URL格式",
					"url":       baseURL,
					"success":   false,
				})
				fmt.Println(string(errData))
			} else {
				fmt.Fprintf(os.Stderr, "❌ 错误: 无效的URL格式: %s\n", baseURL)
				fmt.Fprintln(os.Stderr, "URL必须包含http或https协议和有效的主机名")
			}
			os.Exit(1)
		}

		// 发送请求
		req, err := http.NewRequest(requestMethod, baseURL, nil)
		if err != nil {
			if outputJSON {
				errData, _ := json.Marshal(map[string]interface{}{
					"error":     "创建请求失败",
					"message":   err.Error(),
					"success":   false,
				})
				fmt.Println(string(errData))
			} else {
				fmt.Fprintf(os.Stderr, "❌ 错误: 创建请求失败: %v\n", err)
			}
			os.Exit(1)
		}

		// 进度指示器
		if !outputJSON {
			fmt.Print("正在连接... ")
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
				errMsg = fmt.Sprintf("请求超时 (超过 %ds)", int(timeout.Seconds()))
			} else if strings.Contains(errStr, "connection refused") {
				errMsg = "连接被拒绝 (服务器未监听该端口)"
			} else if strings.Contains(errStr, "network is unreachable") {
				errMsg = "网络不可达"
			} else if strings.Contains(errStr, "EOF") {
				errMsg = "连接异常关闭 (服务器可能不存在或无响应)"
			} else if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "NXDOMAIN") {
				errMsg = "DNS解析失败 (域名不存在或网络配置错误)"
			} else if strings.Contains(errStr, "invalid URL") || strings.Contains(errStr, "parse error") {
				errMsg = "无效的URL格式"
			} else {
				// 其他网络错误
				if netErr, ok := err.(net.Error); ok {
					errMsg = fmt.Sprintf("网络错误: %v", netErr)
				} else {
					errMsg = fmt.Sprintf("请求失败: %v", err)
				}
			}

			if outputJSON {
				errData, _ := json.Marshal(map[string]interface{}{
					"error":     errMsg,
					"url":       baseURL,
					"success":   false,
				})
				fmt.Println(string(errData))
			} else {
				fmt.Fprintf(os.Stderr, "❌ 连通失败: %s\n", errMsg)
			}
			os.Exit(1)
		}
		defer resp.Body.Close()

		duration := time.Since(start)

		// 清除进度指示器
		if !outputJSON {
			fmt.Printf("\r")
		}

		// 输出结果
		isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 300
		if outputJSON {
			result := map[string]interface{}{
				"url":           baseURL,
				"statusCode":    resp.StatusCode,
				"statusText":    http.StatusText(resp.StatusCode),
				"requestMethod": requestMethod,
				"durationMs":    duration.Milliseconds(),
				"timeoutMs":     timeout.Milliseconds(),
				"success":       isSuccess,
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("✅ 连通成功! \n")
			fmt.Printf("   URL: %s\n", baseURL)
			fmt.Printf("   方法: %s\n", requestMethod)
			fmt.Printf("   状态码: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
			fmt.Printf("   响应时间: %dms\n", duration.Milliseconds())
			fmt.Printf("   超时设置: %s\n", timeout)

			// 提供额外提示
			if !isSuccess {
				fmt.Printf("⚠️  注意: 服务器返回非成功状态码\n")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
	// 定义flag并绑定到变量
	pingCmd.Flags().StringVarP(&customURL, "url", "u", "", "测试自定义URL")
	pingCmd.Flags().BoolVarP(&outputJSON, "json", "j", false, "JSON格式输出")
	pingCmd.Flags().StringVarP(&requestMethod, "method", "X", "HEAD", "请求方法")
	pingCmd.Flags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "请求超时时间")
}

