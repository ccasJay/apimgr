package utils

import (
	"strings"
	"testing"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "Empty key",
			key:      "",
			expected: "****",
		},
		{
			name:     "Short key (4 chars)",
			key:      "1234",
			expected: "****",
		},
		{
			name:     "Short key (8 chars)",
			key:      "12345678",
			expected: "****",
		},
		{
			name:     "Normal key (12 chars)",
			key:      "123456789012",
			expected: "1234****9012",
		},
		{
			name:     "Long API key",
			key:      "test-key-abcdefghijklmnopqrstuvwxyz",
			expected: "test****wxyz",
		},
		{
			name:     "OpenAI style key",
			key:      "test-proj-abcdefghijklmnopqrstuvwxyz123456",
			expected: "test****3456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskAPIKey(tt.key)
			if got != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.key, got, tt.expected)
			}
		})
	}
}

func TestMaskAPIKeySecure(t *testing.T) {
	// Test that the masked key doesn't expose sensitive information
	t.Run("Masked key should not contain middle characters", func(t *testing.T) {
		key := "test-key-supersecretkey123"
		masked := MaskAPIKey(key)

		// Check that middle part is completely masked
		if strings.Contains(masked, "supersecret") {
			t.Errorf("Masked key contains sensitive middle part: %q", masked)
		}

		// Check that only start and end are visible
		if !strings.HasPrefix(masked, "test") {
			t.Errorf("Masked key should start with first 4 chars: %q", masked)
		}
		if !strings.HasSuffix(masked, "y123") {
			t.Errorf("Masked key should end with last 4 chars: %q", masked)
		}
	})

	t.Run("Short keys should be completely masked", func(t *testing.T) {
		shortKeys := []string{"", "1", "12", "123", "1234", "12345", "123456", "1234567", "12345678"}
		for _, key := range shortKeys {
			masked := MaskAPIKey(key)
			if masked != "****" {
				t.Errorf("Short key %q should be completely masked as ****, got %q", key, masked)
			}
		}
	})
}

func BenchmarkMaskAPIKey(b *testing.B) {
	keys := []string{
		"",
		"1234",
		"12345678",
		"test-key-abcdefghijklmnopqrstuvwxyz",
		"test-proj-verylongfakekeywithmanycharacters123456789",
	}

	for _, key := range keys {
		b.Run("key_len_"+string(rune(len(key))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = MaskAPIKey(key)
			}
		})
	}
}


// TestValidateURL tests the ValidateURL function with various URL formats
func TestValidateURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Valid URLs
		{
			name:     "Valid HTTPS URL",
			url:      "https://api.example.com",
			expected: true,
		},
		{
			name:     "Valid HTTP URL",
			url:      "http://api.example.com",
			expected: true,
		},
		{
			name:     "Valid URL with port",
			url:      "https://api.example.com:8080",
			expected: true,
		},
		{
			name:     "Valid URL with path",
			url:      "https://api.example.com/v1/chat",
			expected: true,
		},
		{
			name:     "Valid URL with trailing slash",
			url:      "https://api.example.com/",
			expected: true,
		},
		{
			name:     "Valid localhost URL",
			url:      "http://localhost:8080",
			expected: true,
		},
		{
			name:     "Valid IP address URL",
			url:      "http://192.168.1.1:3000",
			expected: true,
		},
		// Invalid URLs
		{
			name:     "Empty string",
			url:      "",
			expected: false,
		},
		{
			name:     "No scheme",
			url:      "api.example.com",
			expected: false,
		},
		{
			name:     "No host",
			url:      "https://",
			expected: false,
		},
		{
			name:     "Invalid scheme - ftp",
			url:      "ftp://files.example.com",
			expected: false,
		},
		{
			name:     "Invalid scheme - file",
			url:      "file:///path/to/file",
			expected: false,
		},
		{
			name:     "Just scheme",
			url:      "https",
			expected: false,
		},
		{
			name:     "Malformed URL",
			url:      "not a url at all",
			expected: false,
		},
		{
			name:     "Missing colon after scheme",
			url:      "https//example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateURL(tt.url)
			if got != tt.expected {
				t.Errorf("ValidateURL(%q) = %v, want %v", tt.url, got, tt.expected)
			}
		})
	}
}

// TestNormalizeURL tests the NormalizeURL function
func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "URL without trailing slash",
			url:      "https://api.example.com",
			expected: "https://api.example.com/",
		},
		{
			name:     "URL with trailing slash",
			url:      "https://api.example.com/",
			expected: "https://api.example.com/",
		},
		{
			name:     "URL with path without trailing slash",
			url:      "https://api.example.com/v1",
			expected: "https://api.example.com/v1/",
		},
		{
			name:     "URL with path with trailing slash",
			url:      "https://api.example.com/v1/",
			expected: "https://api.example.com/v1/",
		},
		{
			name:     "Empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "URL with port",
			url:      "http://localhost:8080",
			expected: "http://localhost:8080/",
		},
		{
			name:     "URL with query params",
			url:      "https://api.example.com?key=value",
			expected: "https://api.example.com?key=value/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeURL(tt.url)
			if got != tt.expected {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

// TestExtractHost tests the ExtractHost function
func TestExtractHost(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Simple HTTPS URL",
			url:      "https://api.example.com",
			expected: "api.example.com",
		},
		{
			name:     "URL with port",
			url:      "https://api.example.com:8080",
			expected: "api.example.com:8080",
		},
		{
			name:     "URL with path",
			url:      "https://api.example.com/v1/chat",
			expected: "api.example.com",
		},
		{
			name:     "Localhost URL",
			url:      "http://localhost:3000",
			expected: "localhost:3000",
		},
		{
			name:     "IP address URL",
			url:      "http://192.168.1.1:8080",
			expected: "192.168.1.1:8080",
		},
		{
			name:     "URL with query params",
			url:      "https://api.example.com?key=value",
			expected: "api.example.com",
		},
		// Invalid URLs should return empty string
		{
			name:     "Empty string",
			url:      "",
			expected: "",
		},
		{
			name:     "Invalid URL - no scheme",
			url:      "api.example.com",
			expected: "",
		},
		{
			name:     "Malformed URL",
			url:      "not a url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractHost(tt.url)
			if got != tt.expected {
				t.Errorf("ExtractHost(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}
