package sync

import (
	"encoding/json"
	"fmt"
	"strings"

	"apimgr/config/models"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SyncOptions provides options for synchronization
type SyncOptions struct {
	DryRun        bool  // 仅验证，不写入
	CreateBackup  bool  // 更新前创建备份
	PreserveOther bool  // 保留非 ANTHROPIC 环境变量
}

// UpdateEnvField updates the env field in Claude Code configuration JSON
// It only updates ANTHROPIC_ fields and preserves non-ANTHROPIC fields when PreserveOther is true
func UpdateEnvField(originalContent string, cfg *models.APIConfig, opts SyncOptions) (string, error) {
	// Parse the JSON content to verify it's valid
	result := gjson.Parse(originalContent)
	if !result.Exists() {
		return "", fmt.Errorf("invalid JSON content")
	}

	// Extract existing env field if it exists
	existingEnv := make(map[string]string)
	if envResult := result.Get("env"); envResult.Exists() {
		envResult.ForEach(func(key, value gjson.Result) bool {
			existingEnv[key.Str] = value.Str
			return true
		})
	}

	// Create updated env map
	updatedEnv := make(map[string]string)

	// Preserve non-ANTHROPIC environment variables if requested
	if opts.PreserveOther {
		for key, value := range existingEnv {
			if !strings.HasPrefix(strings.ToUpper(key), "ANTHROPIC_") {
				updatedEnv[key] = value
			}
		}
	}

	// Set new ANTHROPIC values (only non-empty values)
	if cfg.APIKey != "" {
		updatedEnv["ANTHROPIC_API_KEY"] = cfg.APIKey
	}
	if cfg.Model != "" {
		updatedEnv["ANTHROPIC_MODEL"] = cfg.Model
	}
	if cfg.AuthToken != "" {
		updatedEnv["ANTHROPIC_AUTH_TOKEN"] = cfg.AuthToken
	}
	if cfg.BaseURL != "" {
		updatedEnv["ANTHROPIC_BASE_URL"] = cfg.BaseURL
	}

	// Convert updatedEnv to JSON string
	envJSON, err := json.Marshal(updatedEnv)
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated env: %w", err)
	}

	// Use sjson.SetRaw to update the env field precisely
	updatedContent, err := sjson.SetRaw(originalContent, "env", string(envJSON))
	if err != nil {
		return "", fmt.Errorf("failed to update env field: %w", err)
	}

	// Validate the update to ensure only env field has changed and non-ANTHROPIC fields are preserved
	if err := validateJSONUpdate(originalContent, updatedContent); err != nil {
		return "", fmt.Errorf("update validation failed: %w", err)
	}

	return updatedContent, nil
}

// validateJSONUpdate validates that only the env field has changed in the JSON
func validateJSONUpdate(originalContent string, updatedContent string) error {
	// 1. Validate JSON validity
	if !json.Valid([]byte(originalContent)) {
		return fmt.Errorf("original JSON is invalid")
	}

	if !json.Valid([]byte(updatedContent)) {
		return fmt.Errorf("updated JSON is invalid")
	}

	// 2. Ensure only env field has changed (excluding ANTHROPIC_ fields)
	original, updated, err := parseToMaps(originalContent, updatedContent)
	if err != nil {
		return err
	}

	// Compare all fields except env
	differences := deepCompare(original, updated)
	if len(differences) > 0 {
		return fmt.Errorf("unexpected changes to non-env fields: %s", strings.Join(differences, ", "))
	}

	// 3. Check if non-ANTHROPIC fields were preserved in env
	originalEnv, err := extractEnv(originalContent)
	if err != nil {
		return err
	}

	updatedEnv, err := extractEnv(updatedContent)
	if err != nil {
		return err
	}

	// Check that all non-ANTHROPIC fields are preserved
	for key, originalVal := range originalEnv {
		if !strings.HasPrefix(strings.ToUpper(key), "ANTHROPIC_") {
			if updatedVal, exists := updatedEnv[key]; exists {
				if fmt.Sprintf("%v", originalVal) != fmt.Sprintf("%v", updatedVal) {
					return fmt.Errorf("non-ANTHROPIC field '%s' was modified", key)
				}
			} else {
				return fmt.Errorf("non-ANTHROPIC field '%s' was deleted", key)
			}
		}
	}

	return nil
}

// parseToMaps parses two JSON strings to maps for deep comparison
func parseToMaps(originalStr, updatedStr string) (map[string]interface{}, map[string]interface{}, error) {
	var original map[string]interface{}
	if err := json.Unmarshal([]byte(originalStr), &original); err != nil {
		return nil, nil, fmt.Errorf("failed to parse original JSON: %w", err)
	}

	var updated map[string]interface{}
	if err := json.Unmarshal([]byte(updatedStr), &updated); err != nil {
		return nil, nil, fmt.Errorf("failed to parse updated JSON: %w", err)
	}

	return original, updated, nil
}

// deepCompare compares two maps and returns a list of differing fields
func deepCompare(original, updated map[string]interface{}) []string {
	var differences []string

	// Check all keys in original
	for key, originalVal := range original {
		if key == "env" {
			// Skip env field for now, it will be checked separately
			continue
		}

		if updatedVal, exists := updated[key]; exists {
			// Check if values are maps for deep comparison
			originalMap, originalIsMap := originalVal.(map[string]interface{})
			updatedMap, updatedIsMap := updatedVal.(map[string]interface{})

			if originalIsMap && updatedIsMap {
				// Recursively compare nested maps
				nestedDiffs := deepCompare(originalMap, updatedMap)
				for _, diff := range nestedDiffs {
					differences = append(differences, key+"."+diff)
				}
			} else {
				// Compare values directly
				if fmt.Sprintf("%v", originalVal) != fmt.Sprintf("%v", updatedVal) {
					differences = append(differences, key)
				}
			}
		} else {
			// Key missing in updated
			differences = append(differences, key+" (missing)")
		}
	}

	// Check for new keys in updated
	for key := range updated {
		if key == "env" {
			continue // Skip env field, checked separately
		}
		if _, exists := original[key]; !exists {
			differences = append(differences, key+" (new)")
		}
	}

	return differences
}

// extractEnv extracts the env field from JSON content
func extractEnv(jsonContent string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return nil, err
	}

	env, exists := data["env"]
	if !exists {
		return make(map[string]interface{}), nil // Return empty map if env doesn't exist
	}

	envMap, ok := env.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("env field is not a map")
	}

	return envMap, nil
}
