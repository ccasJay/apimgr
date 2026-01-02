package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"testing/quick"
	"time"

	"apimgr/config/models"
	"apimgr/config/session"
)

// setupTestSession creates a test config manager with a temporary directory
func setupTestSession(t *testing.T) (*Manager, string) {
	t.Helper()
	t.Setenv("APIMGR_ACTIVE", "") // Ensure clean environment
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	cm := &Manager{configPath: configPath}
	return cm, tempDir
}

// setupTestClaudeEnv creates a fake home directory with Claude settings for testing
func setupTestClaudeEnv(t *testing.T, tempDir string) (string, func()) {
	t.Helper()
	fakeHome := filepath.Join(tempDir, "home")
	claudeDir := filepath.Join(fakeHome, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create claude dir: %v", err)
	}

	// Override HOME for this test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", fakeHome)
	cleanup := func() {
		os.Setenv("HOME", originalHome)
	}

	return fakeHome, cleanup
}

// Feature: switch-local-mode-fix, Property 5: Local mode creates session marker
// Validates: Requirements 1.5
// For any valid alias, executing `apimgr switch -l <alias>` should create a session marker file
// containing the PID, alias, and timestamp
func TestPropertySessionMarkerCreation(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	_, tempDir := setupTestSession(t)

	// Property: For any valid PID and alias, CreateSessionMarker should create a file
	// with the correct content (PID, alias, and timestamp)
	property := func(pidNum uint16, aliasBytes []byte) bool {
		// Constrain inputs to valid domain
		if pidNum == 0 {
			pidNum = 1
		}
		pid := strconv.Itoa(int(pidNum))

		// Generate a valid alias (non-empty alphanumeric string)
		alias := "test"
		if len(aliasBytes) > 0 {
			// Use first few bytes to create a simple alias
			alias = "alias-" + strconv.Itoa(int(aliasBytes[0]%100))
		}

		// Create a fresh config manager for each test
		configPath := filepath.Join(tempDir, "config.json")
		cm := &Manager{configPath: configPath}

		// Record time before creation
		beforeTime := time.Now().Add(-time.Second)

		// Create session marker
		err := session.CreateSessionMarker(cm.configPath, pid, alias)
		if err != nil {
			t.Logf("CreateSessionMarker failed: %v", err)
			return false
		}

		// Record time after creation
		afterTime := time.Now().Add(time.Second)

		// Verify the session marker file exists
		markerPath := filepath.Join(tempDir, "session-"+pid)
		data, err := os.ReadFile(markerPath)
		if err != nil {
			t.Logf("Failed to read session marker: %v", err)
			return false
		}

		// Parse the session marker
		var marker session.SessionMarker
		if err := json.Unmarshal(data, &marker); err != nil {
			t.Logf("Failed to parse session marker: %v", err)
			return false
		}

		// Verify PID matches
		if marker.PID != pid {
			t.Logf("PID mismatch: expected %s, got %s", pid, marker.PID)
			return false
		}

		// Verify alias matches
		if marker.Alias != alias {
			t.Logf("Alias mismatch: expected %s, got %s", alias, marker.Alias)
			return false
		}

		// Verify timestamp is within expected range
		if marker.Timestamp.Before(beforeTime) || marker.Timestamp.After(afterTime) {
			t.Logf("Timestamp out of range: %v not in [%v, %v]", marker.Timestamp, beforeTime, afterTime)
			return false
		}

		// Clean up for next iteration
		os.Remove(markerPath)

		return true
	}

	// Run the property test with 100 iterations
	config := &quick.Config{MaxCount: 100}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Test CleanupSession removes marker file
func TestCleanupSession(t *testing.T) {
	cm, tempDir := setupTestSession(t)

	// Create a session marker
	pid := "12345"
	err := session.CreateSessionMarker(cm.configPath, pid, "test-alias")
	if err != nil {
		t.Fatalf("Failed to create session marker: %v", err)
	}

	// Verify it exists
	markerPath := filepath.Join(tempDir, "session-"+pid)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Fatal("Session marker should exist after creation")
	}

	// Clean up the session
	err = session.CleanupSession(cm.configPath, pid)
	if err != nil {
		t.Fatalf("Failed to cleanup session: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("Session marker should not exist after cleanup")
	}

	// Cleanup of non-existent session should not error
	err = session.CleanupSession(cm.configPath, "99999")
	if err != nil {
		t.Errorf("Cleanup of non-existent session should not error: %v", err)
	}
}

// Test HasActiveLocalSessions detects active sessions and cleans stale ones
func TestHasActiveLocalSessions(t *testing.T) {
	cm, tempDir := setupTestSession(t)

	// Initially no sessions
	hasActive, err := session.HasActiveLocalSessions(cm.configPath)
	if err != nil {
		t.Fatalf("HasActiveLocalSessions failed: %v", err)
	}
	if hasActive {
		t.Error("Should have no active sessions initially")
	}

	// Create a session marker with current process PID (which is running)
	currentPID := strconv.Itoa(os.Getpid())
	err = session.CreateSessionMarker(cm.configPath, currentPID, "test-alias")
	if err != nil {
		t.Fatalf("Failed to create session marker: %v", err)
	}

	// Now should have active session
	hasActive, err = session.HasActiveLocalSessions(cm.configPath)
	if err != nil {
		t.Fatalf("HasActiveLocalSessions failed: %v", err)
	}
	if !hasActive {
		t.Error("Should have active session for current process")
	}

	// Create a stale session marker with non-existent PID
	stalePID := "999999999" // Very unlikely to be a real PID
	staleMarkerPath := filepath.Join(tempDir, "session-"+stalePID)
	staleMarker := session.SessionMarker{PID: stalePID, Alias: "stale", Timestamp: time.Now()}
	data, _ := json.Marshal(staleMarker)
	os.WriteFile(staleMarkerPath, data, 0600)

	// HasActiveLocalSessions should clean up stale session
	hasActive, err = session.HasActiveLocalSessions(cm.configPath)
	if err != nil {
		t.Fatalf("HasActiveLocalSessions failed: %v", err)
	}
	if !hasActive {
		t.Error("Should still have active session for current process")
	}

	// Stale session file should be cleaned up
	if _, err := os.Stat(staleMarkerPath); !os.IsNotExist(err) {
		t.Error("Stale session marker should have been cleaned up")
	}

	// Clean up current session
	session.CleanupSession(cm.configPath, currentPID)
}

// Feature: switch-local-mode-fix, Property 4: Local mode updates Claude Code settings
// Validates: Requirements 1.4
// For any valid alias, executing `apimgr switch -l <alias>` should update Claude Code settings files
// with the configuration values from the specified alias
func TestPropertySyncClaudeSettingsOnly(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	tempDir := t.TempDir()

	// Setup Claude environment
	fakeHome, cleanup := setupTestClaudeEnv(t, tempDir)
	defer cleanup()

	claudeDir := filepath.Join(fakeHome, ".claude")
	claudeSettingsPath := filepath.Join(claudeDir, "settings.json")

	// Create initial Claude settings file
	initialSettings := map[string]interface{}{
		"env": map[string]interface{}{
			"OTHER_VAR": "keep-this",
		},
	}
	initialData, _ := json.MarshalIndent(initialSettings, "", "  ")
	if err := os.WriteFile(claudeSettingsPath, initialData, 0600); err != nil {
		t.Fatalf("Failed to write initial claude settings: %v", err)
	}

	// Property: For any valid API config, SyncClaudeSettingsOnly should update Claude Code settings
	// with the correct environment variables
	property := func(apiKeyBytes, authTokenBytes, baseURLSuffix, modelBytes []byte) bool {
		// Generate valid config values
		apiKey := ""
		if len(apiKeyBytes) > 0 && apiKeyBytes[0]%2 == 0 {
			apiKey = "sk-test-" + strconv.Itoa(int(apiKeyBytes[0]))
		}

		authToken := ""
		if len(authTokenBytes) > 0 && authTokenBytes[0]%2 == 0 {
			authToken = "token-" + strconv.Itoa(int(authTokenBytes[0]))
		}

		// At least one auth method must be present
		if apiKey == "" && authToken == "" {
			apiKey = "sk-default-key"
		}

		baseURL := ""
		if len(baseURLSuffix) > 0 && baseURLSuffix[0]%2 == 0 {
			baseURL = "https://api-" + strconv.Itoa(int(baseURLSuffix[0])) + ".example.com"
		}

		model := ""
		if len(modelBytes) > 0 && modelBytes[0]%2 == 0 {
			model = "claude-" + strconv.Itoa(int(modelBytes[0]%10))
		}

		cfg := &models.APIConfig{
			Alias:     "test-alias",
			APIKey:    apiKey,
			AuthToken: authToken,
			BaseURL:   baseURL,
			Model:     model,
		}

		// Create config manager
		configPath := filepath.Join(tempDir, "config.json")
		cm := &Manager{configPath: configPath}

		// Sync to Claude settings
		err := cm.SyncClaudeSettingsOnly(cfg)
		if err != nil {
			t.Logf("SyncClaudeSettingsOnly failed: %v", err)
			return false
		}

		// Read and verify Claude settings
		data, err := os.ReadFile(claudeSettingsPath)
		if err != nil {
			t.Logf("Failed to read claude settings: %v", err)
			return false
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Logf("Failed to parse claude settings: %v", err)
			return false
		}

		env, ok := settings["env"].(map[string]interface{})
		if !ok {
			t.Logf("env field not found or wrong type")
			return false
		}

		// Verify OTHER_VAR is preserved
		if env["OTHER_VAR"] != "keep-this" {
			t.Logf("OTHER_VAR was not preserved")
			return false
		}

		// Verify API key
		if apiKey != "" {
			if env["ANTHROPIC_API_KEY"] != apiKey {
				t.Logf("API key mismatch: expected %s, got %v", apiKey, env["ANTHROPIC_API_KEY"])
				return false
			}
		} else {
			if _, exists := env["ANTHROPIC_API_KEY"]; exists {
				t.Logf("API key should not exist when empty")
				return false
			}
		}

		// Verify auth token
		if apiKey == "" && authToken != "" {
			if env["ANTHROPIC_AUTH_TOKEN"] != authToken {
				t.Logf("Auth token mismatch: expected %s, got %v", authToken, env["ANTHROPIC_AUTH_TOKEN"])
				return false
			}
		} else {
			if _, exists := env["ANTHROPIC_AUTH_TOKEN"]; exists {
				t.Logf("Auth token should not exist when empty or API key is present")
				return false
			}
		}

		// Verify base URL
		if baseURL != "" {
			if env["ANTHROPIC_BASE_URL"] != baseURL {
				t.Logf("Base URL mismatch: expected %s, got %v", baseURL, env["ANTHROPIC_BASE_URL"])
				return false
			}
		} else {
			if _, exists := env["ANTHROPIC_BASE_URL"]; exists {
				t.Logf("Base URL should not exist when empty")
				return false
			}
		}

		// Verify model
		if model != "" {
			if env["ANTHROPIC_MODEL"] != model {
				t.Logf("Model mismatch: expected %s, got %v", model, env["ANTHROPIC_MODEL"])
				return false
			}
		} else {
			if _, exists := env["ANTHROPIC_MODEL"]; exists {
				t.Logf("Model should not exist when empty")
				return false
			}
		}

		return true
	}

	// Run the property test with 100 iterations
	config := &quick.Config{MaxCount: 100}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}


// Feature: switch-local-mode-fix, Property 11: Load-active restores to global when sessions exist
// Validates: Requirements 4.1, 4.2
// For any state where active local sessions exist, executing `apimgr load-active` should restore
// Claude Code settings to match the global active configuration
func TestPropertyRestoreClaudeToGlobal(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	t.Setenv("APIMGR_ACTIVE", "")
	tempDir := t.TempDir()

	// Setup Claude environment
	fakeHome, cleanup := setupTestClaudeEnv(t, tempDir)
	defer cleanup()

	claudeDir := filepath.Join(fakeHome, ".claude")
	claudeSettingsPath := filepath.Join(claudeDir, "settings.json")

	// Property: For any global active configuration, RestoreClaudeToGlobal should sync it to Claude Code
	property := func(apiKeyNum, authTokenNum uint8) bool {
		// Create Claude settings file with local config (simulating local mode was used)
		localSettings := map[string]interface{}{
			"env": map[string]interface{}{
				"ANTHROPIC_API_KEY":    "local-key",
				"ANTHROPIC_AUTH_TOKEN": "local-token",
				"ANTHROPIC_BASE_URL":   "https://local.example.com",
				"ANTHROPIC_MODEL":      "local-model",
				"OTHER_VAR":            "keep-this",
			},
		}
		localData, _ := json.MarshalIndent(localSettings, "", "  ")
		if err := os.WriteFile(claudeSettingsPath, localData, 0600); err != nil {
			t.Logf("Failed to write local claude settings: %v", err)
			return false
		}

		// Generate global config values
		globalAPIKey := "sk-global-" + strconv.Itoa(int(apiKeyNum))
		globalAuthToken := ""
		if authTokenNum%2 == 0 {
			globalAuthToken = "global-token-" + strconv.Itoa(int(authTokenNum))
		}

		// Create config file with global active configuration
		configPath := filepath.Join(tempDir, "config.json")
		configFile := models.File{
			Active: "global-alias",
			Configs: []models.APIConfig{
				{
					Alias:     "global-alias",
					APIKey:    globalAPIKey,
					AuthToken: globalAuthToken,
					BaseURL:   "https://global.example.com",
					Model:     "global-model",
				},
			},
		}
		configData, _ := json.MarshalIndent(configFile, "", "  ")
		if err := os.WriteFile(configPath, configData, 0600); err != nil {
			t.Logf("Failed to write config file: %v", err)
			return false
		}

		// Create config manager
		cm := &Manager{configPath: configPath}

		// Restore Claude to global
		err := cm.RestoreClaudeToGlobal()
		if err != nil {
			t.Logf("RestoreClaudeToGlobal failed: %v", err)
			return false
		}

		// Read and verify Claude settings
		data, err := os.ReadFile(claudeSettingsPath)
		if err != nil {
			t.Logf("Failed to read claude settings: %v", err)
			return false
		}

		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err != nil {
			t.Logf("Failed to parse claude settings: %v", err)
			return false
		}

		env, ok := settings["env"].(map[string]interface{})
		if !ok {
			t.Logf("env field not found or wrong type")
			return false
		}

		// Verify OTHER_VAR is preserved
		if env["OTHER_VAR"] != "keep-this" {
			t.Logf("OTHER_VAR was not preserved")
			return false
		}

		// Verify global API key is set
		if env["ANTHROPIC_API_KEY"] != globalAPIKey {
			t.Logf("API key mismatch: expected %s, got %v", globalAPIKey, env["ANTHROPIC_API_KEY"])
			return false
		}

		// Verify global auth token
		if globalAPIKey == "" && globalAuthToken != "" {
			if env["ANTHROPIC_AUTH_TOKEN"] != globalAuthToken {
				t.Logf("Auth token mismatch: expected %s, got %v", globalAuthToken, env["ANTHROPIC_AUTH_TOKEN"])
				return false
			}
		} else {
			if _, exists := env["ANTHROPIC_AUTH_TOKEN"]; exists {
				t.Logf("Auth token should not exist when empty or API key is present")
				return false
			}
		}

		// Verify global base URL
		if env["ANTHROPIC_BASE_URL"] != "https://global.example.com" {
			t.Logf("Base URL mismatch: expected https://global.example.com, got %v", env["ANTHROPIC_BASE_URL"])
			return false
		}

		// Verify global model
		if env["ANTHROPIC_MODEL"] != "global-model" {
			t.Logf("Model mismatch: expected global-model, got %v", env["ANTHROPIC_MODEL"])
			return false
		}

		return true
	}

	// Run the property test with 100 iterations
	config := &quick.Config{MaxCount: 100}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Test RestoreClaudeToGlobal clears settings when no global active exists
func TestRestoreClaudeToGlobalNoActive(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	t.Setenv("APIMGR_ACTIVE", "")
	tempDir := t.TempDir()

	// Setup Claude environment
	fakeHome, cleanup := setupTestClaudeEnv(t, tempDir)
	defer cleanup()

	claudeDir := filepath.Join(fakeHome, ".claude")
	claudeSettingsPath := filepath.Join(claudeDir, "settings.json")

	// Create Claude settings file with local config
	localSettings := map[string]interface{}{
		"env": map[string]interface{}{
			"ANTHROPIC_API_KEY":    "local-key",
			"ANTHROPIC_AUTH_TOKEN": "local-token",
			"ANTHROPIC_BASE_URL":   "https://local.example.com",
			"ANTHROPIC_MODEL":      "local-model",
			"OTHER_VAR":            "keep-this",
		},
	}
	localData, _ := json.MarshalIndent(localSettings, "", "  ")
	if err := os.WriteFile(claudeSettingsPath, localData, 0600); err != nil {
		t.Fatalf("Failed to write local claude settings: %v", err)
	}

	// Create config file with NO global active configuration
	configPath := filepath.Join(tempDir, "config.json")
	configFile := models.File{
		Active:  "", // No active config
		Configs: []models.APIConfig{},
	}
	configData, _ := json.MarshalIndent(configFile, "", "  ")
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create config manager
	cm := &Manager{configPath: configPath}

	// Restore Claude to global (should clear ANTHROPIC_* vars)
	err := cm.RestoreClaudeToGlobal()
	if err != nil {
		t.Fatalf("RestoreClaudeToGlobal failed: %v", err)
	}

	// Read and verify Claude settings
	data, err := os.ReadFile(claudeSettingsPath)
	if err != nil {
		t.Fatalf("Failed to read claude settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to parse claude settings: %v", err)
	}

	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		t.Fatal("env field not found or wrong type")
	}

	// Verify OTHER_VAR is preserved
	if env["OTHER_VAR"] != "keep-this" {
		t.Error("OTHER_VAR was not preserved")
	}

	// Verify ANTHROPIC_* vars are cleared
	if _, exists := env["ANTHROPIC_API_KEY"]; exists {
		t.Error("ANTHROPIC_API_KEY should be cleared")
	}
	if _, exists := env["ANTHROPIC_AUTH_TOKEN"]; exists {
		t.Error("ANTHROPIC_AUTH_TOKEN should be cleared")
	}
	if _, exists := env["ANTHROPIC_BASE_URL"]; exists {
		t.Error("ANTHROPIC_BASE_URL should be cleared")
	}
	if _, exists := env["ANTHROPIC_MODEL"]; exists {
		t.Error("ANTHROPIC_MODEL should be cleared")
	}
}

// Feature: switch-local-mode-fix, Property 13: Load-active cleans up stale sessions
// Validates: Requirements 4.3
// For any session marker files with non-existent PIDs, executing `apimgr load-active` should delete
// those stale session files
func TestPropertyStaleSessionCleanup(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	_, tempDir := setupTestSession(t)

	// Property: For any number of stale session files (with non-existent PIDs),
	// HasActiveLocalSessions should clean them all up
	property := func(numStaleSessions uint8) bool {
		// Constrain to reasonable number of sessions (1-10)
		numSessions := int(numStaleSessions%10) + 1

		configPath := filepath.Join(tempDir, "config.json")
		cm := &Manager{configPath: configPath}

		// Create stale session markers with non-existent PIDs
		// Use very high PIDs that are extremely unlikely to exist
		stalePIDs := make([]string, numSessions)
		for i := 0; i < numSessions; i++ {
			// Use PIDs in the range 900000000-900000009 (very unlikely to be real)
			stalePID := strconv.Itoa(900000000 + i)
			stalePIDs[i] = stalePID

			staleMarkerPath := filepath.Join(tempDir, "session-"+stalePID)
			staleMarker := session.SessionMarker{
				PID:       stalePID,
				Alias:     "stale-alias-" + strconv.Itoa(i),
				Timestamp: time.Now().Add(-time.Hour), // Created an hour ago
			}
			data, _ := json.Marshal(staleMarker)
			if err := os.WriteFile(staleMarkerPath, data, 0600); err != nil {
				t.Logf("Failed to create stale session marker: %v", err)
				return false
			}
		}

		// Verify all stale session files exist before cleanup
		for _, pid := range stalePIDs {
			markerPath := filepath.Join(tempDir, "session-"+pid)
			if _, err := os.Stat(markerPath); os.IsNotExist(err) {
				t.Logf("Stale session marker should exist before cleanup: %s", pid)
				return false
			}
		}

		// Call HasActiveLocalSessions which should clean up stale sessions
		hasActive, err := session.HasActiveLocalSessions(cm.configPath)
		if err != nil {
			t.Logf("HasActiveLocalSessions failed: %v", err)
			return false
		}

		// Should return false since all sessions are stale (non-existent PIDs)
		if hasActive {
			t.Logf("Should have no active sessions when all PIDs are non-existent")
			return false
		}

		// Verify all stale session files are cleaned up
		for _, pid := range stalePIDs {
			markerPath := filepath.Join(tempDir, "session-"+pid)
			if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
				t.Logf("Stale session marker should be cleaned up: %s", pid)
				return false
			}
		}

		return true
	}

	// Run the property test with 100 iterations
	config := &quick.Config{MaxCount: 100}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}


// Feature: switch-local-mode-fix, Property 13 (additional): Mixed active and stale sessions
// Validates: Requirements 4.3
// When there are both active and stale sessions, only stale sessions should be cleaned up
func TestPropertyMixedSessionCleanup(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	cm, tempDir := setupTestSession(t)

	// Create an active session marker with current process PID (which is running)
	currentPID := strconv.Itoa(os.Getpid())
	err := session.CreateSessionMarker(cm.configPath, currentPID, "active-alias")
	if err != nil {
		t.Fatalf("Failed to create active session marker: %v", err)
	}

	// Create multiple stale session markers with non-existent PIDs
	stalePIDs := []string{"900000001", "900000002", "900000003"}
	for _, stalePID := range stalePIDs {
		staleMarkerPath := filepath.Join(tempDir, "session-"+stalePID)
		staleMarker := session.SessionMarker{
			PID:       stalePID,
			Alias:     "stale-alias",
			Timestamp: time.Now().Add(-time.Hour),
		}
		data, _ := json.Marshal(staleMarker)
		if err := os.WriteFile(staleMarkerPath, data, 0600); err != nil {
			t.Fatalf("Failed to create stale session marker: %v", err)
		}
	}

	// Call HasActiveLocalSessions
	hasActive, err := session.HasActiveLocalSessions(cm.configPath)
	if err != nil {
		t.Fatalf("HasActiveLocalSessions failed: %v", err)
	}

	// Should return true because current process session is active
	if !hasActive {
		t.Error("Should have active session for current process")
	}

	// Verify active session marker still exists
	activeMarkerPath := filepath.Join(tempDir, "session-"+currentPID)
	if _, err := os.Stat(activeMarkerPath); os.IsNotExist(err) {
		t.Error("Active session marker should still exist")
	}

	// Verify all stale session markers are cleaned up
	for _, stalePID := range stalePIDs {
		staleMarkerPath := filepath.Join(tempDir, "session-"+stalePID)
		if _, err := os.Stat(staleMarkerPath); !os.IsNotExist(err) {
			t.Errorf("Stale session marker should be cleaned up: %s", stalePID)
		}
	}

	// Clean up active session
	session.CleanupSession(cm.configPath, currentPID)
}

// Feature: switch-local-mode-fix, Property 14: Cleanup-session removes marker
// Validates: Requirements 4.1
// For any valid PID, executing `apimgr cleanup-session <pid>` should delete the corresponding
// session marker file
func TestPropertyCleanupSessionRemovesMarker(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	_, tempDir := setupTestSession(t)

	// Property: For any valid PID with an existing session marker,
	// CleanupSession should remove the marker file
	property := func(pidNum uint16, aliasBytes []byte) bool {
		// Constrain inputs to valid domain
		if pidNum == 0 {
			pidNum = 1
		}
		pid := strconv.Itoa(int(pidNum))

		// Generate a valid alias (non-empty alphanumeric string)
		alias := "test"
		if len(aliasBytes) > 0 {
			alias = "alias-" + strconv.Itoa(int(aliasBytes[0]%100))
		}

		// Create a fresh config manager for each test
		configPath := filepath.Join(tempDir, "config.json")
		cm := &Manager{configPath: configPath}

		// Create session marker first
		err := session.CreateSessionMarker(cm.configPath, pid, alias)
		if err != nil {
			t.Logf("CreateSessionMarker failed: %v", err)
			return false
		}

		// Verify the session marker file exists
		markerPath := filepath.Join(tempDir, "session-"+pid)
		if _, err := os.Stat(markerPath); os.IsNotExist(err) {
			t.Logf("Session marker should exist after creation")
			return false
		}

		// Cleanup the session
		err = session.CleanupSession(cm.configPath, pid)
		if err != nil {
			t.Logf("CleanupSession failed: %v", err)
			return false
		}

		// Verify the session marker file is removed
		if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
			t.Logf("Session marker should not exist after cleanup")
			return false
		}

		return true
	}

	// Run the property test with 100 iterations
	config := &quick.Config{MaxCount: 100}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Feature: switch-local-mode-fix, Property 14 (additional): Cleanup-session handles non-existent markers gracefully
// Validates: Requirements 4.1
// For any PID without an existing session marker, CleanupSession should not error
func TestPropertyCleanupSessionNonExistent(t *testing.T) {
	// Use t.TempDir() for automatic cleanup
	_, tempDir := setupTestSession(t)

	// Property: For any PID without an existing session marker,
	// CleanupSession should complete without error
	property := func(pidNum uint16) bool {
		// Constrain inputs to valid domain
		if pidNum == 0 {
			pidNum = 1
		}
		pid := strconv.Itoa(int(pidNum))

		// Create a fresh config manager for each test
		configPath := filepath.Join(tempDir, "config.json")
		cm := &Manager{configPath: configPath}

		// Verify the session marker file does NOT exist
		markerPath := filepath.Join(tempDir, "session-"+pid)
		if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
			// Clean it up first if it exists from a previous iteration
			os.Remove(markerPath)
		}

		// Cleanup the session (should not error even though marker doesn't exist)
		err := session.CleanupSession(cm.configPath, pid)
		if err != nil {
			t.Logf("CleanupSession should not error for non-existent marker: %v", err)
			return false
		}

		return true
	}

	// Run the property test with 100 iterations
	config := &quick.Config{MaxCount: 100}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}
