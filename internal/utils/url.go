package utils

import (
	"net/url"
	"strings"
)

// ValidateURL validates that a URL has a valid scheme and host
func ValidateURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	parsed, err := url.ParseRequestURI(rawURL)
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

// NormalizeURL ensures URL has a trailing slash
func NormalizeURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	if !strings.HasSuffix(rawURL, "/") {
		return rawURL + "/"
	}
	return rawURL
}

// ExtractHost extracts the host from a URL
func ExtractHost(rawURL string) string {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
