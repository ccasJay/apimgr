package cmd

import (
	"testing"
)

func TestEditCmd(t *testing.T) {
	t.Run("Command definition", func(t *testing.T) {
		expected := "edit <alias>"
		if editCmd.Use != expected {
			t.Errorf("editCmd.Use = %q, want %q", editCmd.Use, expected)
		}
	})

	t.Run("Short description", func(t *testing.T) {
		if editCmd.Short == "" {
			t.Error("editCmd.Short should not be empty")
		}
	})

	t.Run("Long description", func(t *testing.T) {
		if editCmd.Long == "" {
			t.Error("editCmd.Long should not be empty")
		}
	})

	t.Run("RunE is set", func(t *testing.T) {
		if editCmd.RunE == nil {
			t.Error("editCmd.RunE should not be nil")
		}
	})

	t.Run("Flags are defined", func(t *testing.T) {
		flags := []string{"alias", "sk", "ak", "url", "model"}

		for _, name := range flags {
			flag := editCmd.Flags().Lookup(name)
			if flag == nil {
				t.Errorf("Flag --%s should be defined", name)
			}
		}
	})

	t.Run("Args requires exactly 1 argument", func(t *testing.T) {
		// Test with no arguments
		err := editCmd.Args(editCmd, []string{})
		if err == nil {
			t.Error("Args should return error when no arguments provided")
		}

		// Test with exactly 1 argument
		err = editCmd.Args(editCmd, []string{"test-alias"})
		if err != nil {
			t.Errorf("Args should not return error with 1 argument, got: %v", err)
		}

		// Test with too many arguments
		err = editCmd.Args(editCmd, []string{"alias1", "alias2"})
		if err == nil {
			t.Error("Args should return error when more than 1 argument provided")
		}
	})
}

func TestFieldType(t *testing.T) {
	t.Run("FieldType constants", func(t *testing.T) {
		// Verify FieldType constants are defined
		tests := []struct {
			fieldType FieldType
			expected  int
		}{
			{FieldAlias, 0},
			{FieldAPIKey, 1},
			{FieldAuthToken, 2},
			{FieldBaseURL, 3},
			{FieldModel, 4},
		}

		for _, tt := range tests {
			if int(tt.fieldType) != tt.expected {
				t.Errorf("FieldType %d = %d, want %d", tt.fieldType, int(tt.fieldType), tt.expected)
			}
		}
	})
}

func TestGetFieldKey(t *testing.T) {
	tests := []struct {
		name      string
		fieldType FieldType
		expected  string
	}{
		{"FieldAlias", FieldAlias, "alias"},
		{"FieldAPIKey", FieldAPIKey, "api_key"},
		{"FieldAuthToken", FieldAuthToken, "auth_token"},
		{"FieldBaseURL", FieldBaseURL, "base_url"},
		{"FieldModel", FieldModel, "model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFieldKey(tt.fieldType)
			if got != tt.expected {
				t.Errorf("getFieldKey(%v) = %q, want %q", tt.fieldType, got, tt.expected)
			}
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		name      string
		fieldType FieldType
		expected  bool
	}{
		{"FieldAlias is not sensitive", FieldAlias, false},
		{"FieldAPIKey is sensitive", FieldAPIKey, true},
		{"FieldAuthToken is sensitive", FieldAuthToken, true},
		{"FieldBaseURL is not sensitive", FieldBaseURL, false},
		{"FieldModel is not sensitive", FieldModel, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSensitiveField(tt.fieldType)
			if got != tt.expected {
				t.Errorf("isSensitiveField(%v) = %v, want %v", tt.fieldType, got, tt.expected)
			}
		})
	}
}
