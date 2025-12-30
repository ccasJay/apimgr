package cmd

import (
	"testing"
)

func TestListCmd(t *testing.T) {
	t.Run("Command definition", func(t *testing.T) {
		if listCmd.Use != "list" {
			t.Errorf("listCmd.Use = %q, want %q", listCmd.Use, "list")
		}
	})

	t.Run("Short description", func(t *testing.T) {
		if listCmd.Short == "" {
			t.Error("listCmd.Short should not be empty")
		}
		expected := "List all API configurations"
		if listCmd.Short != expected {
			t.Errorf("listCmd.Short = %q, want %q", listCmd.Short, expected)
		}
	})

	t.Run("Long description", func(t *testing.T) {
		if listCmd.Long == "" {
			t.Error("listCmd.Long should not be empty")
		}
	})

	t.Run("RunE is set", func(t *testing.T) {
		if listCmd.RunE == nil {
			t.Error("listCmd.RunE should not be nil")
		}
	})
}
