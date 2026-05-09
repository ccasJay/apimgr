package sync

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ccasJay/apimgr/config/models"
)

func mustUnmarshalJSON(t *testing.T, content string) map[string]interface{} {
	t.Helper()

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	return data
}

func mustExtractEnvMap(t *testing.T, content string) map[string]interface{} {
	t.Helper()

	data := mustUnmarshalJSON(t, content)
	env, ok := data["env"].(map[string]interface{})
	if !ok {
		t.Fatalf("env field missing or wrong type: %#v", data["env"])
	}

	return env
}

func TestUpdateEnvField_APIKeyPreservesNonAnthropicEnv(t *testing.T) {
	original := `{
		"permissions": {"allow": ["read"]},
		"env": {
			"FOO": "keep",
			"BAR": "stay",
			"ANTHROPIC_API_KEY": "old-key",
			"ANTHROPIC_MODEL": "old-model"
		}
	}`

	cfg := &models.APIConfig{
		APIKey:  "sk-new",
		Model:   "claude-3-7-sonnet",
		BaseURL: "https://api.example.com",
	}

	updated, err := UpdateEnvField(original, cfg, SyncOptions{PreserveOther: true})
	if err != nil {
		t.Fatalf("UpdateEnvField() error: %v", err)
	}

	data := mustUnmarshalJSON(t, updated)
	permissions := data["permissions"].(map[string]interface{})
	if _, ok := permissions["allow"]; !ok {
		t.Fatalf("permissions.allow missing after update")
	}

	env := mustExtractEnvMap(t, updated)
	if env["FOO"] != "keep" || env["BAR"] != "stay" {
		t.Fatalf("non-Anthropic env vars not preserved: %#v", env)
	}
	if env["ANTHROPIC_API_KEY"] != "sk-new" {
		t.Fatalf("ANTHROPIC_API_KEY = %v, want sk-new", env["ANTHROPIC_API_KEY"])
	}
	if _, exists := env["ANTHROPIC_AUTH_TOKEN"]; exists {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN should be absent when API key is used")
	}
	if env["ANTHROPIC_MODEL"] != "claude-3-7-sonnet" {
		t.Fatalf("ANTHROPIC_MODEL = %v, want claude-3-7-sonnet", env["ANTHROPIC_MODEL"])
	}
	if env["ANTHROPIC_BASE_URL"] != "https://api.example.com" {
		t.Fatalf("ANTHROPIC_BASE_URL = %v, want https://api.example.com", env["ANTHROPIC_BASE_URL"])
	}
}

func TestUpdateEnvField_AuthTokenReplacesAPIKey(t *testing.T) {
	original := `{"env":{"FOO":"keep","ANTHROPIC_API_KEY":"old-key"}}`
	cfg := &models.APIConfig{AuthToken: "token-new"}

	updated, err := UpdateEnvField(original, cfg, SyncOptions{PreserveOther: true})
	if err != nil {
		t.Fatalf("UpdateEnvField() error: %v", err)
	}

	env := mustExtractEnvMap(t, updated)
	if env["FOO"] != "keep" {
		t.Fatalf("FOO = %v, want keep", env["FOO"])
	}
	if env["ANTHROPIC_AUTH_TOKEN"] != "token-new" {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN = %v, want token-new", env["ANTHROPIC_AUTH_TOKEN"])
	}
	if _, exists := env["ANTHROPIC_API_KEY"]; exists {
		t.Fatalf("ANTHROPIC_API_KEY should be absent when auth token is used")
	}
}

func TestUpdateEnvField_EmptyCredentialsRemoveAnthropicAuthVars(t *testing.T) {
	original := `{"env":{"FOO":"keep","ANTHROPIC_API_KEY":"old-key","ANTHROPIC_AUTH_TOKEN":"old-token"}}`
	cfg := &models.APIConfig{}

	updated, err := UpdateEnvField(original, cfg, SyncOptions{PreserveOther: true})
	if err != nil {
		t.Fatalf("UpdateEnvField() error: %v", err)
	}

	env := mustExtractEnvMap(t, updated)
	if env["FOO"] != "keep" {
		t.Fatalf("FOO = %v, want keep", env["FOO"])
	}
	if _, exists := env["ANTHROPIC_API_KEY"]; exists {
		t.Fatalf("ANTHROPIC_API_KEY should be removed")
	}
	if _, exists := env["ANTHROPIC_AUTH_TOKEN"]; exists {
		t.Fatalf("ANTHROPIC_AUTH_TOKEN should be removed")
	}
}

func TestUpdateEnvField_AddsEnvWhenMissing(t *testing.T) {
	original := `{"permissions":{"allow":["read"]}}`
	cfg := &models.APIConfig{APIKey: "sk-new"}

	updated, err := UpdateEnvField(original, cfg, SyncOptions{PreserveOther: true})
	if err != nil {
		t.Fatalf("UpdateEnvField() error: %v", err)
	}

	data := mustUnmarshalJSON(t, updated)
	if _, ok := data["permissions"]; !ok {
		t.Fatalf("permissions missing after env update")
	}
	env := mustExtractEnvMap(t, updated)
	if env["ANTHROPIC_API_KEY"] != "sk-new" {
		t.Fatalf("ANTHROPIC_API_KEY = %v, want sk-new", env["ANTHROPIC_API_KEY"])
	}
}

func TestUpdateEnvField_InvalidJSONReturnsError(t *testing.T) {
	_, err := UpdateEnvField("{invalid", &models.APIConfig{APIKey: "sk-new"}, SyncOptions{PreserveOther: true})
	if err == nil {
		t.Fatal("UpdateEnvField() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "original JSON is invalid") {
		t.Fatalf("error = %v, want original JSON is invalid", err)
	}
}

func TestUpdateEnvField_InvalidEnvTypeReturnsError(t *testing.T) {
	_, err := UpdateEnvField(`{"env":"not-an-object"}`, &models.APIConfig{APIKey: "sk-new"}, SyncOptions{PreserveOther: true})
	if err == nil {
		t.Fatal("UpdateEnvField() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "env field is not a map") {
		t.Fatalf("error = %v, want env field is not a map", err)
	}
}

func TestValidateJSONUpdate_FailsWhenNonEnvFieldChanges(t *testing.T) {
	original := `{"version":1,"env":{"FOO":"keep"}}`
	updated := `{"version":2,"env":{"FOO":"keep"}}`

	err := validateJSONUpdate(original, updated)
	if err == nil {
		t.Fatal("validateJSONUpdate() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected changes to non-env fields") {
		t.Fatalf("error = %v, want unexpected changes to non-env fields", err)
	}
}

func TestValidateJSONUpdate_FailsWhenNonAnthropicEnvModified(t *testing.T) {
	original := `{"env":{"FOO":"a","ANTHROPIC_API_KEY":"old"}}`
	updated := `{"env":{"FOO":"b","ANTHROPIC_API_KEY":"new"}}`

	err := validateJSONUpdate(original, updated)
	if err == nil {
		t.Fatal("validateJSONUpdate() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "non-ANTHROPIC field 'FOO' was modified") {
		t.Fatalf("error = %v, want non-ANTHROPIC field 'FOO' was modified", err)
	}
}

func TestValidateJSONUpdate_FailsWhenNonAnthropicEnvDeleted(t *testing.T) {
	original := `{"env":{"FOO":"a","ANTHROPIC_API_KEY":"old"}}`
	updated := `{"env":{"ANTHROPIC_API_KEY":"new"}}`

	err := validateJSONUpdate(original, updated)
	if err == nil {
		t.Fatal("validateJSONUpdate() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "non-ANTHROPIC field 'FOO' was deleted") {
		t.Fatalf("error = %v, want non-ANTHROPIC field 'FOO' was deleted", err)
	}
}

func TestValidateJSONUpdate_AllowsAddingEnvWhenOriginalMissing(t *testing.T) {
	original := `{"permissions":{"allow":["read"]}}`
	updated := `{"permissions":{"allow":["read"]},"env":{"ANTHROPIC_API_KEY":"sk-new"}}`

	if err := validateJSONUpdate(original, updated); err != nil {
		t.Fatalf("validateJSONUpdate() error: %v", err)
	}
}
