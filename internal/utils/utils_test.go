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
