// Package compatibility provides API compatibility testing functionality
// for validating that API configurations work correctly with Claude Code.
package compatibility

import (
	"net/http"
	"strings"
)

// ErrorInfo contains detailed information about a categorized error
type ErrorInfo struct {
	Category    string `json:"category"`
	StatusCode  int    `json:"statusCode,omitempty"`
	Message     string `json:"message"`
	UserMessage string `json:"userMessage"`
}

// User-facing error messages for each category
var errorUserMessages = map[string]string{
	ErrorCategoryAuthFailure:        "Authentication failed. Please check your API key or token.",
	ErrorCategoryModelNotFound:      "Model not found. Please verify the model name.",
	ErrorCategoryRateLimit:          "Rate limit exceeded. Please try again later.",
	ErrorCategoryFormatIncompatible: "Response format is not compatible with Claude Code.",
	ErrorCategoryNetworkError:       "Network error: unable to connect to the API.",
	ErrorCategoryServerError:        "Server error occurred. Please try again later.",
	ErrorCategoryEndpointNotFound:   "API endpoint not found. Please verify the base URL.",
	ErrorCategoryUnknown:            "An unknown error occurred.",
}

// CategorizeError categorizes an HTTP response error based on status code and response body.
// It returns the error category string that can be used for reporting.
//
// The categorization logic:
// - 401, 403 → authentication_failure
// - 404 → model_not_found (if body contains "model") or endpoint_not_found
// - 429 → rate_limit
// - 200 with invalid body → format_incompatibility
// - 500+ → server_error
// - Other → unknown_error
func CategorizeError(statusCode int, body []byte) string {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrorCategoryAuthFailure
	case http.StatusNotFound:
		// Check if it's model not found vs endpoint not found
		bodyStr := strings.ToLower(string(body))
		if strings.Contains(bodyStr, "model") {
			return ErrorCategoryModelNotFound
		}
		return ErrorCategoryEndpointNotFound
	case http.StatusTooManyRequests:
		return ErrorCategoryRateLimit
	case http.StatusOK:
		// Successful HTTP but potentially invalid format
		return ErrorCategoryFormatIncompatible
	default:
		if statusCode >= http.StatusInternalServerError {
			return ErrorCategoryServerError
		}
		return ErrorCategoryUnknown
	}
}

// CategorizeErrorWithInfo categorizes an error and returns detailed ErrorInfo
func CategorizeErrorWithInfo(statusCode int, body []byte, errMsg string) *ErrorInfo {
	category := CategorizeError(statusCode, body)

	userMessage := errorUserMessages[category]
	if userMessage == "" {
		userMessage = errorUserMessages[ErrorCategoryUnknown]
	}

	message := errMsg
	if message == "" {
		message = userMessage
	}

	return &ErrorInfo{
		Category:    category,
		StatusCode:  statusCode,
		Message:     message,
		UserMessage: userMessage,
	}
}

// CategorizeNetworkError returns error info for network-related errors
func CategorizeNetworkError(err error) *ErrorInfo {
	message := "Network error"
	if err != nil {
		message = err.Error()
	}

	return &ErrorInfo{
		Category:    ErrorCategoryNetworkError,
		StatusCode:  0,
		Message:     message,
		UserMessage: errorUserMessages[ErrorCategoryNetworkError],
	}
}

// GetUserMessage returns the user-friendly message for an error category
func GetUserMessage(category string) string {
	if msg, ok := errorUserMessages[category]; ok {
		return msg
	}
	return errorUserMessages[ErrorCategoryUnknown]
}
