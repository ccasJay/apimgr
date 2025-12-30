package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncCmd(t *testing.T) {
	t.Run("Command definition", func(t *testing.T) {
		expected := "sync [subcommand]"
		if syncCmd.Use != expected {
			t.Errorf("syncCmd.Use = %q, want %q", syncCmd.Use, expected)
		}
	})

	t.Run("Short description", func(t *testing.T) {
		if syncCmd.Short == "" {
			t.Error("syncCmd.Short should not be empty")
		}
	})

	t.Run("Long description", func(t *testing.T) {
		if syncCmd.Long == "" {
			t.Error("syncCmd.Long should not be empty")
		}
	})

	t.Run("RunE is set", func(t *testing.T) {
		if syncCmd.RunE == nil {
			t.Error("syncCmd.RunE should not be nil")
		}
	})
}

func TestSyncSubcommands(t *testing.T) {
	subcommands := []struct {
		name    string
		cmd     string
		hasRun  bool
		hasRunE bool
	}{
		{"status", "status", true, false},
		{"claude", "claude", true, false},
		{"init", "init", true, false},
		{"list", "list", true, false},
	}

	for _, sc := range subcommands {
		t.Run(sc.name+" subcommand", func(t *testing.T) {
			// Find the subcommand
			var found bool
			for _, cmd := range syncCmd.Commands() {
				if cmd.Use == sc.cmd {
					found = true

					if cmd.Short == "" {
						t.Errorf("sync %s: Short should not be empty", sc.name)
					}

					if cmd.Long == "" {
						t.Errorf("sync %s: Long should not be empty", sc.name)
					}

					if sc.hasRun && cmd.Run == nil {
						t.Errorf("sync %s: Run should not be nil", sc.name)
					}

					if sc.hasRunE && cmd.RunE == nil {
						t.Errorf("sync %s: RunE should not be nil", sc.name)
					}

					break
				}
			}

			if !found {
				t.Errorf("sync %s subcommand not found", sc.name)
			}
		})
	}
}

func TestWriteJSONFile(t *testing.T) {
	t.Run("Write valid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.json")

		data := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"nested": map[string]interface{}{
				"inner": "data",
			},
		}

		err := writeJSONFile(filePath, data)
		if err != nil {
			t.Fatalf("writeJSONFile() error = %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatal("File was not created")
		}

		// Verify content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(content, &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if result["key1"] != "value1" {
			t.Errorf("key1 = %v, want %v", result["key1"], "value1")
		}
	})

	t.Run("Write to nested directory", func(t *testing.T) {
		tempDir := t.TempDir()
		nestedDir := filepath.Join(tempDir, "nested", "dir")
		if err := os.MkdirAll(nestedDir, 0755); err != nil {
			t.Fatalf("Failed to create nested dir: %v", err)
		}

		filePath := filepath.Join(nestedDir, "test.json")
		data := map[string]string{"test": "value"}

		err := writeJSONFile(filePath, data)
		if err != nil {
			t.Fatalf("writeJSONFile() error = %v", err)
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatal("File was not created in nested directory")
		}
	})

	t.Run("File permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.json")

		data := map[string]string{"test": "value"}
		err := writeJSONFile(filePath, data)
		if err != nil {
			t.Fatalf("writeJSONFile() error = %v", err)
		}

		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		// Check file permissions (0600)
		expectedPerm := os.FileMode(0600)
		if info.Mode().Perm() != expectedPerm {
			t.Errorf("File permissions = %v, want %v", info.Mode().Perm(), expectedPerm)
		}
	})
}
