package cmd

import (
	"bytes"
	"strings"
	"testing"

	"apimgr/config/models"
)

func TestWriteShellExportsUsesSafeQuoting(t *testing.T) {
	cfg := &models.APIConfig{
		Alias:   "ignored",
		APIKey:  "sk'$(touch /tmp/pwned)",
		BaseURL: "https://api.example.com/$TOKEN",
		Model:   "`touch /tmp/pwned`",
	}

	var output bytes.Buffer
	writeShellUnsets(&output)
	writeShellExports(&output, cfg, "alias'$(touch /tmp/pwned)")

	got := output.String()
	if strings.Contains(got, `export ANTHROPIC_API_KEY="`) {
		t.Fatalf("exports must not use double quotes:\n%s", got)
	}
	expected := []string{
		"unset ANTHROPIC_API_KEY",
		`export ANTHROPIC_API_KEY='sk'"'"'$(touch /tmp/pwned)'`,
		"export ANTHROPIC_BASE_URL='https://api.example.com/$TOKEN'",
		"export ANTHROPIC_MODEL='`touch /tmp/pwned`'",
		`export APIMGR_ACTIVE='alias'"'"'$(touch /tmp/pwned)'`,
	}
	for _, want := range expected {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestWriteShellExportsPrefersAPIKey(t *testing.T) {
	cfg := &models.APIConfig{
		Alias:     "alias",
		APIKey:    "sk-test",
		AuthToken: "token-test",
	}

	var output bytes.Buffer
	writeShellExports(&output, cfg, cfg.Alias)

	got := output.String()
	if !strings.Contains(got, "export ANTHROPIC_API_KEY='sk-test'") {
		t.Fatalf("expected API key export, got:\n%s", got)
	}
	if strings.Contains(got, "ANTHROPIC_AUTH_TOKEN") {
		t.Fatalf("auth token should not be exported when API key is present:\n%s", got)
	}
}
