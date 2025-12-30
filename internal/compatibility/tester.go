// Package compatibility provides API compatibility testing functionality
// for validating that API configurations work correctly with Claude Code.
package compatibility

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"apimgr/config"
	"apimgr/internal/providers"
)

// ProviderURLPatterns maps URL patterns to provider names for auto-detection
var ProviderURLPatterns = map[string]string{
	"api.anthropic.com": "anthropic",
	"anthropic.com":     "anthropic",
	"api.openai.com":    "openai",
	"openai.com":        "openai",
}

// DetectProviderFromURL attempts to detect the provider type from a base URL.
// It returns the detected provider name and a boolean indicating if detection was successful.
// If the URL is ambiguous or doesn't match known patterns, it returns empty string and false.
func DetectProviderFromURL(baseURL string) (string, bool) {
	if baseURL == "" {
		return "", false
	}

	// Parse the URL to extract the host
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", false
	}

	host := parsedURL.Host
	if host == "" {
		// Try parsing without scheme
		parsedURL, err = url.Parse("https://" + baseURL)
		if err != nil {
			return "", false
		}
		host = parsedURL.Host
	}

	// Remove port if present
	if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
		// Check if it's not an IPv6 address
		if !strings.Contains(host, "]") || colonIdx > strings.LastIndex(host, "]") {
			host = host[:colonIdx]
		}
	}

	// Convert to lowercase for case-insensitive matching
	host = strings.ToLower(host)

	// Check for exact matches first
	if provider, ok := ProviderURLPatterns[host]; ok {
		return provider, true
	}

	// Check for suffix matches (e.g., subdomain.api.anthropic.com)
	for pattern, provider := range ProviderURLPatterns {
		if strings.HasSuffix(host, "."+pattern) || host == pattern {
			return provider, true
		}
	}

	return "", false
}

// Tester coordinates compatibility testing for API configurations
type Tester struct {
	client     *http.Client
	config     *config.APIConfig
	provider   providers.Provider
	verbose    bool
	customPath string
}

// TesterOption is a functional option for configuring a Tester
type TesterOption func(*Tester)

// WithVerbose enables verbose output
func WithVerbose(verbose bool) TesterOption {
	return func(t *Tester) {
		t.verbose = verbose
	}
}

// WithCustomPath sets a custom endpoint path
func WithCustomPath(path string) TesterOption {
	return func(t *Tester) {
		t.customPath = path
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) TesterOption {
	return func(t *Tester) {
		t.client = client
	}
}

// NewTester creates a new compatibility tester for the given API configuration.
// It resolves the provider based on the config's Provider field, or auto-detects
// from the base URL if the provider is not explicitly set.
func NewTester(cfg *config.APIConfig, opts ...TesterOption) (*Tester, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Resolve provider
	providerName := cfg.Provider
	if providerName == "" {
		// Try to auto-detect provider from base URL
		if detectedProvider, ok := DetectProviderFromURL(cfg.BaseURL); ok {
			providerName = detectedProvider
		} else {
			// Fall back to anthropic as default
			providerName = "anthropic"
		}
	}

	provider, err := providers.Get(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve provider: %w", err)
	}

	t := &Tester{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config:   cfg,
		provider: provider,
		verbose:  false,
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	return t, nil
}

// getModel returns the model to use for testing.
// If the config has a model specified, it uses that.
// Otherwise, it falls back to the provider's default model.
func (t *Tester) getModel() string {
	if t.config.Model != "" {
		return t.config.Model
	}
	return t.provider.DefaultModel()
}

// getRequestBuilder returns the appropriate request builder for the provider
func (t *Tester) getRequestBuilder() RequestBuilder {
	if t.customPath != "" {
		return NewRequestBuilderWithCustomPath(t.config, t.provider, t.customPath)
	}
	return NewRequestBuilder(t.config, t.provider)
}

// getValidator returns the appropriate response validator for the provider
func (t *Tester) getValidator() ResponseValidator {
	return NewValidator(t.provider.Name())
}

// getSSEValidator returns the appropriate SSE validator for the provider
func (t *Tester) getSSEValidator() SSEValidator {
	return NewSSEValidator(t.provider.Name())
}

// TestBasic performs a non-streaming compatibility test.
// It sends a chat completion request and validates the response format.
func (t *Tester) TestBasic() (*TestResult, error) {
	result := &TestResult{
		Success: false,
		Checks:  []CheckResult{},
	}

	startTime := time.Now()

	// Build the request
	builder := t.getRequestBuilder()
	model := t.getModel()
	req, err := builder.BuildChatRequest(model, false)
	if err != nil {
		result.Error = fmt.Sprintf("failed to build request: %v", err)
		result.ResponseTime = time.Since(startTime)
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Request Construction",
			Passed:   false,
			Message:  result.Error,
			Critical: true,
		})
		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}

	// Add connection check
	result.Checks = append(result.Checks, CheckResult{
		Name:     "Request Construction",
		Passed:   true,
		Message:  fmt.Sprintf("Request built for %s API", t.provider.Name()),
		Critical: true,
	})

	// Send the request
	resp, err := t.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("network error: %v", err)
		result.ResponseTime = time.Since(startTime)
		errInfo := CategorizeNetworkError(err)
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Connection",
			Passed:   false,
			Message:  errInfo.UserMessage,
			Critical: true,
		})
		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}
	defer resp.Body.Close()

	result.ResponseTime = time.Since(startTime)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Connection",
			Passed:   false,
			Message:  "Failed to read response body",
			Critical: true,
		})
		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}

	// Connection succeeded
	result.Checks = append(result.Checks, CheckResult{
		Name:     "Connection",
		Passed:   true,
		Message:  fmt.Sprintf("Connected successfully (HTTP %d)", resp.StatusCode),
		Critical: true,
	})

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		errCategory := CategorizeError(resp.StatusCode, body)
		errInfo := CategorizeErrorWithInfo(resp.StatusCode, body, "")
		
		isCritical := errCategory == ErrorCategoryAuthFailure || 
			errCategory == ErrorCategoryEndpointNotFound ||
			errCategory == ErrorCategoryModelNotFound

		result.Checks = append(result.Checks, CheckResult{
			Name:     "Authentication",
			Passed:   errCategory != ErrorCategoryAuthFailure,
			Message:  errInfo.UserMessage,
			Critical: isCritical,
		})

		if errCategory != ErrorCategoryAuthFailure {
			result.Error = errInfo.UserMessage
		}

		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}

	// Authentication passed
	result.Checks = append(result.Checks, CheckResult{
		Name:     "Authentication",
		Passed:   true,
		Message:  "Authentication successful",
		Critical: true,
	})

	// Validate response format
	validator := t.getValidator()
	validationResult, err := validator.ValidateBasicResponse(body)
	if err != nil {
		result.Error = fmt.Sprintf("validation error: %v", err)
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Response Format",
			Passed:   false,
			Message:  result.Error,
			Critical: true,
		})
		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}

	// Add response format check
	if validationResult.Valid {
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Response Format",
			Passed:   true,
			Message:  fmt.Sprintf("Response format is valid for %s API", t.provider.Name()),
			Critical: true,
		})
	} else {
		missingFields := strings.Join(validationResult.MissingFields, ", ")
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Response Format",
			Passed:   false,
			Message:  fmt.Sprintf("Missing or malformed fields: %s", missingFields),
			Critical: true,
		})
	}

	// Determine final result
	result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
	result.Success = result.CompatibilityLevel == CompatibilityFull

	return result, nil
}


// TestStreaming performs a streaming compatibility test.
// It sends a streaming chat completion request and validates the SSE response format.
func (t *Tester) TestStreaming() (*TestResult, error) {
	result := &TestResult{
		Success: false,
		Checks:  []CheckResult{},
	}

	startTime := time.Now()

	// Build the streaming request
	builder := t.getRequestBuilder()
	model := t.getModel()
	req, err := builder.BuildChatRequest(model, true)
	if err != nil {
		result.Error = fmt.Sprintf("failed to build streaming request: %v", err)
		result.ResponseTime = time.Since(startTime)
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Streaming Request Construction",
			Passed:   false,
			Message:  result.Error,
			Critical: true,
		})
		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}

	// Add request construction check
	result.Checks = append(result.Checks, CheckResult{
		Name:     "Streaming Request Construction",
		Passed:   true,
		Message:  fmt.Sprintf("Streaming request built for %s API", t.provider.Name()),
		Critical: true,
	})

	// Send the request
	resp, err := t.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("network error: %v", err)
		result.ResponseTime = time.Since(startTime)
		errInfo := CategorizeNetworkError(err)
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Streaming Connection",
			Passed:   false,
			Message:  errInfo.UserMessage,
			Critical: true,
		})
		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}
	defer resp.Body.Close()

	result.ResponseTime = time.Since(startTime)

	// Connection succeeded
	result.Checks = append(result.Checks, CheckResult{
		Name:     "Streaming Connection",
		Passed:   true,
		Message:  fmt.Sprintf("Connected successfully (HTTP %d)", resp.StatusCode),
		Critical: true,
	})

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errCategory := CategorizeError(resp.StatusCode, body)
		errInfo := CategorizeErrorWithInfo(resp.StatusCode, body, "")

		isCritical := errCategory == ErrorCategoryAuthFailure ||
			errCategory == ErrorCategoryEndpointNotFound ||
			errCategory == ErrorCategoryModelNotFound

		result.Checks = append(result.Checks, CheckResult{
			Name:     "Streaming Authentication",
			Passed:   errCategory != ErrorCategoryAuthFailure,
			Message:  errInfo.UserMessage,
			Critical: isCritical,
		})

		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}

	// Authentication passed
	result.Checks = append(result.Checks, CheckResult{
		Name:     "Streaming Authentication",
		Passed:   true,
		Message:  "Authentication successful",
		Critical: true,
	})

	// Validate SSE format
	sseValidator := t.getSSEValidator()
	sseResult, err := sseValidator.ValidateStream(resp.Body)
	if err != nil {
		result.Error = fmt.Sprintf("SSE validation error: %v", err)
		result.Checks = append(result.Checks, CheckResult{
			Name:     "SSE Format",
			Passed:   false,
			Message:  result.Error,
			Critical: true,
		})
		result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
		return result, nil
	}

	// Add SSE format check
	if len(sseResult.MalformedLines) == 0 && sseResult.EventCount > 0 {
		result.Checks = append(result.Checks, CheckResult{
			Name:     "SSE Format",
			Passed:   true,
			Message:  fmt.Sprintf("SSE format is valid (%d events received)", sseResult.EventCount),
			Critical: true,
		})
	} else {
		message := "SSE format validation failed"
		if len(sseResult.MalformedLines) > 0 {
			message = fmt.Sprintf("Malformed SSE lines detected: %v", sseResult.MalformedLines)
		} else if sseResult.EventCount == 0 {
			message = "No SSE events received"
		}
		result.Checks = append(result.Checks, CheckResult{
			Name:     "SSE Format",
			Passed:   false,
			Message:  message,
			Critical: true,
		})
	}

	// Add completion signal check
	if sseResult.HasCompletionSignal {
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Completion Signal",
			Passed:   true,
			Message:  fmt.Sprintf("Completion signal received (%s)", sseResult.CompletionType),
			Critical: false, // Non-critical as some proxies may not forward this
		})
	} else {
		result.Checks = append(result.Checks, CheckResult{
			Name:     "Completion Signal",
			Passed:   false,
			Message:  "No completion signal received (expected [DONE] or message_stop)",
			Critical: false,
		})
	}

	// Determine final result
	result.CompatibilityLevel, _ = DetermineCompatibilityLevel(result.Checks)
	result.Success = result.CompatibilityLevel == CompatibilityFull

	return result, nil
}

// RunFullTest runs a complete compatibility test including both basic and streaming tests.
// If includeStreaming is false, only the basic test is run.
func (t *Tester) RunFullTest(includeStreaming bool) (*TestResult, error) {
	// Run basic test first
	basicResult, err := t.TestBasic()
	if err != nil {
		return basicResult, err
	}

	// If basic test failed critically, don't proceed with streaming
	if basicResult.CompatibilityLevel == CompatibilityNone {
		return basicResult, nil
	}

	// If streaming test is not requested, return basic result
	if !includeStreaming {
		return basicResult, nil
	}

	// Run streaming test
	streamingResult, err := t.TestStreaming()
	if err != nil {
		// Merge basic checks with streaming error
		streamingResult.Checks = append(basicResult.Checks, streamingResult.Checks...)
		streamingResult.CompatibilityLevel, _ = DetermineCompatibilityLevel(streamingResult.Checks)
		return streamingResult, err
	}

	// Merge results
	combinedResult := &TestResult{
		Checks:       append(basicResult.Checks, streamingResult.Checks...),
		ResponseTime: basicResult.ResponseTime + streamingResult.ResponseTime,
	}

	// Determine combined compatibility level
	combinedResult.CompatibilityLevel, _ = DetermineCompatibilityLevel(combinedResult.Checks)
	combinedResult.Success = combinedResult.CompatibilityLevel == CompatibilityFull

	// Combine errors if any
	if basicResult.Error != "" && streamingResult.Error != "" {
		combinedResult.Error = fmt.Sprintf("%s; %s", basicResult.Error, streamingResult.Error)
	} else if basicResult.Error != "" {
		combinedResult.Error = basicResult.Error
	} else if streamingResult.Error != "" {
		combinedResult.Error = streamingResult.Error
	}

	return combinedResult, nil
}

// GetProvider returns the resolved provider for this tester
func (t *Tester) GetProvider() providers.Provider {
	return t.provider
}

// GetConfig returns the API configuration for this tester
func (t *Tester) GetConfig() *config.APIConfig {
	return t.config
}

// GetModel returns the model that will be used for testing
func (t *Tester) GetModel() string {
	return t.getModel()
}

// WasProviderAutoDetected returns true if the provider was auto-detected from the URL
// rather than explicitly specified in the config
func (t *Tester) WasProviderAutoDetected() bool {
	return t.config.Provider == ""
}
