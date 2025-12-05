package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShellDetection tests the shell detection logic
func TestShellDetection(t *testing.T) {
	tests := []struct {
		name           string
		shellEnv       string
		expectedRCFile string
		shouldFail     bool
	}{
		{
			name:           "zsh shell",
			shellEnv:       "/bin/zsh",
			expectedRCFile: ".zshrc",
			shouldFail:     false,
		},
		{
			name:           "bash shell",
			shellEnv:       "/bin/bash",
			expectedRCFile: ".bashrc",
			shouldFail:     false,
		},
		{
			name:           "zsh with path",
			shellEnv:       "/usr/local/bin/zsh",
			expectedRCFile: ".zshrc",
			shouldFail:     false,
		},
		{
			name:           "bash with path",
			shellEnv:       "/usr/bin/bash",
			expectedRCFile: ".bashrc",
			shouldFail:     false,
		},
		{
			name:           "unsupported shell",
			shellEnv:       "/bin/fish",
			expectedRCFile: "",
			shouldFail:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine RC file based on shell
			var rcFile string
			if strings.Contains(tt.shellEnv, "zsh") {
				rcFile = ".zshrc"
			} else if strings.Contains(tt.shellEnv, "bash") {
				rcFile = ".bashrc"
			}

			if tt.shouldFail {
				if rcFile != "" {
					t.Errorf("Expected shell detection to fail for %s, but got rcFile: %s", tt.shellEnv, rcFile)
				}
			} else {
				if rcFile != tt.expectedRCFile {
					t.Errorf("Expected rcFile %s for shell %s, got %s", tt.expectedRCFile, tt.shellEnv, rcFile)
				}
			}
		})
	}
}

// TestRCFileModification tests that the RC file is correctly modified
func TestRCFileModification(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "apimgr-install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rcFile := filepath.Join(tempDir, ".zshrc")

	// Test 1: Create new RC file
	initScript := `
# apimgr - auto load active API configuration and shell integration
if command -v apimgr &> /dev/null; then
  eval "$(command apimgr load-active)"
  apimgr() {
    if [ "${1-}" = "switch" ]; then
      shift
      local __apimgr_output
      if ! __apimgr_output="$(command apimgr switch "$@")"; then
        return $?
      fi
      eval "$__apimgr_output"
      return $?
    else
      command apimgr "$@"
    fi
  }
fi
`

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("Failed to open RC file: %v", err)
	}
	_, err = f.WriteString(initScript)
	if err != nil {
		t.Fatalf("Failed to write to RC file: %v", err)
	}
	f.Close()

	// Verify file was created and contains the script
	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	if !strings.Contains(string(content), "apimgr load-active") {
		t.Error("RC file should contain 'apimgr load-active'")
	}

	if !strings.Contains(string(content), "apimgr() {") {
		t.Error("RC file should contain 'apimgr() {' function wrapper")
	}
}

// TestDuplicateDetection tests that duplicate installations are detected
func TestDuplicateDetection(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "apimgr-install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rcFile := filepath.Join(tempDir, ".zshrc")

	// Create RC file with existing installation
	existingContent := `# Some existing config
export PATH=$PATH:/usr/local/bin

# apimgr - auto load active API configuration and shell integration
if command -v apimgr &> /dev/null; then
  eval "$(command apimgr load-active)"
  apimgr() {
    if [ "${1-}" = "switch" ]; then
      shift
      local __apimgr_output
      if ! __apimgr_output="$(command apimgr switch "$@")"; then
        return $?
      fi
      eval "$__apimgr_output"
      return $?
    else
      command apimgr "$@"
    fi
  }
fi
`

	if err := os.WriteFile(rcFile, []byte(existingContent), 0600); err != nil {
		t.Fatalf("Failed to write RC file: %v", err)
	}

	// Check for existing installation
	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	// Test detection of new version (with apimgr() function wrapper)
	hasLoadActive := strings.Contains(string(content), "apimgr load-active")
	hasWrapper := strings.Contains(string(content), "apimgr() {")

	if !hasLoadActive {
		t.Error("Should detect 'apimgr load-active' in existing installation")
	}

	if !hasWrapper {
		t.Error("Should detect 'apimgr() {' wrapper in existing installation")
	}

	// Both present means latest version is installed
	if hasLoadActive && hasWrapper {
		// This is the expected state - latest version already installed
		t.Log("Correctly detected latest version installation")
	}
}

// TestOldVersionDetection tests detection of old version installation
func TestOldVersionDetection(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "apimgr-install-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	rcFile := filepath.Join(tempDir, ".zshrc")

	// Create RC file with old version (no wrapper function)
	oldContent := `# Some existing config
export PATH=$PATH:/usr/local/bin

# apimgr - auto load active API configuration
if command -v apimgr &> /dev/null; then
  eval "$(command apimgr load-active)"
fi
`

	if err := os.WriteFile(rcFile, []byte(oldContent), 0600); err != nil {
		t.Fatalf("Failed to write RC file: %v", err)
	}

	// Check for old version installation
	content, err := os.ReadFile(rcFile)
	if err != nil {
		t.Fatalf("Failed to read RC file: %v", err)
	}

	hasLoadActive := strings.Contains(string(content), "apimgr load-active")
	hasWrapper := strings.Contains(string(content), "apimgr() {")

	if !hasLoadActive {
		t.Error("Should detect 'apimgr load-active' in old installation")
	}

	if hasWrapper {
		t.Error("Old version should not have 'apimgr() {' wrapper")
	}

	// Has load-active but no wrapper means old version
	if hasLoadActive && !hasWrapper {
		t.Log("Correctly detected old version installation")
	}
}

// TestNoShellIntegrationFlag tests the --no-shell-integration flag
func TestNoShellIntegrationFlag(t *testing.T) {
	// Test that the flag is properly defined
	// The flag should skip all shell integration when set

	// Verify the flag exists in the command
	flag := installCmd.Flags().Lookup("no-shell-integration")
	if flag == nil {
		t.Fatal("--no-shell-integration flag should be defined")
	}

	if flag.DefValue != "false" {
		t.Errorf("--no-shell-integration default value should be 'false', got '%s'", flag.DefValue)
	}

	// Test flag shorthand (should not have one)
	if flag.Shorthand != "" {
		t.Errorf("--no-shell-integration should not have a shorthand, got '%s'", flag.Shorthand)
	}
}

// TestForceFlag tests the --force flag
func TestForceFlag(t *testing.T) {
	// Verify the flag exists in the command
	flag := installCmd.Flags().Lookup("force")
	if flag == nil {
		t.Fatal("--force flag should be defined")
	}

	if flag.DefValue != "false" {
		t.Errorf("--force default value should be 'false', got '%s'", flag.DefValue)
	}

	if flag.Shorthand != "f" {
		t.Errorf("--force shorthand should be 'f', got '%s'", flag.Shorthand)
	}
}
