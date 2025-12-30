package cmd

import (
	"testing"
)

func TestRemoveCmd(t *testing.T) {
	t.Run("Command definition", func(t *testing.T) {
		expected := "remove [alias]"
		if removeCmd.Use != expected {
			t.Errorf("removeCmd.Use = %q, want %q", removeCmd.Use, expected)
		}
	})

	t.Run("Short description", func(t *testing.T) {
		if removeCmd.Short == "" {
			t.Error("removeCmd.Short should not be empty")
		}
	})

	t.Run("Long description", func(t *testing.T) {
		if removeCmd.Long == "" {
			t.Error("removeCmd.Long should not be empty")
		}
	})

	t.Run("RunE is set", func(t *testing.T) {
		if removeCmd.RunE == nil {
			t.Error("removeCmd.RunE should not be nil")
		}
	})

	t.Run("Args requires exactly 1 argument", func(t *testing.T) {
		// Test with no arguments
		err := removeCmd.Args(removeCmd, []string{})
		if err == nil {
			t.Error("Args should return error when no arguments provided")
		}

		// Test with exactly 1 argument
		err = removeCmd.Args(removeCmd, []string{"test-alias"})
		if err != nil {
			t.Errorf("Args should not return error with 1 argument, got: %v", err)
		}

		// Test with too many arguments
		err = removeCmd.Args(removeCmd, []string{"alias1", "alias2"})
		if err == nil {
			t.Error("Args should return error when more than 1 argument provided")
		}
	})
}
