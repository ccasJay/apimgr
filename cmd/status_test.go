package cmd

import (
	"testing"
)

func TestStatusCmd(t *testing.T) {
	t.Run("Command definition", func(t *testing.T) {
		if statusCmd.Use != "status" {
			t.Errorf("statusCmd.Use = %q, want %q", statusCmd.Use, "status")
		}
	})

	t.Run("Short description", func(t *testing.T) {
		if statusCmd.Short == "" {
			t.Error("statusCmd.Short should not be empty")
		}
		expected := "Show currently active configuration"
		if statusCmd.Short != expected {
			t.Errorf("statusCmd.Short = %q, want %q", statusCmd.Short, expected)
		}
	})

	t.Run("Long description", func(t *testing.T) {
		if statusCmd.Long == "" {
			t.Error("statusCmd.Long should not be empty")
		}
	})

	t.Run("RunE is set", func(t *testing.T) {
		if statusCmd.RunE == nil {
			t.Error("statusCmd.RunE should not be nil")
		}
	})
}
