package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SessionMarker represents a local session marker file
type SessionMarker struct {
	PID       string    `json:"pid"`
	Alias     string    `json:"alias"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateSessionMarker creates a session marker file for local mode
func CreateSessionMarker(configPath string, pid string, alias string) error {
	marker := SessionMarker{
		PID:       pid,
		Alias:     alias,
		Timestamp: time.Now(),
	}

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize session marker: %v", err)
	}

	markerPath := filepath.Join(filepath.Dir(configPath), fmt.Sprintf("session-%s", pid))
	if err := os.WriteFile(markerPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session marker: %v", err)
	}

	return nil
}

// CleanupSession removes a session marker file
func CleanupSession(configPath string, pid string) error {
	markerPath := filepath.Join(filepath.Dir(configPath), fmt.Sprintf("session-%s", pid))
	err := os.Remove(markerPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session marker: %v", err)
	}
	return nil
}

// HasActiveLocalSessions checks if there are any active local sessions
// It also cleans up stale session files (PIDs that no longer exist)
func HasActiveLocalSessions(configPath string) (bool, error) {
	configDir := filepath.Dir(configPath)
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return false, fmt.Errorf("failed to read config directory: %v", err)
	}

	hasActive := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "session-") {
			continue
		}

		// Extract PID from filename
		pidStr := strings.TrimPrefix(name, "session-")
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			// Invalid session file name, clean it up
			os.Remove(filepath.Join(configDir, name))
			continue
		}

		// Check if process is still running
		if isProcessRunning(pid) {
			hasActive = true
		} else {
			// Clean up stale session file
			os.Remove(filepath.Join(configDir, name))
		}
	}

	return hasActive, nil
}

// isProcessRunning checks if a process with the given PID is still running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
