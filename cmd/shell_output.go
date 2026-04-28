package cmd

import (
	"fmt"
	"io"

	"apimgr/config/models"
	"apimgr/internal/utils"
)

func writeShellUnsets(w io.Writer) {
	fmt.Fprintln(w, "unset ANTHROPIC_API_KEY")
	fmt.Fprintln(w, "unset ANTHROPIC_AUTH_TOKEN")
	fmt.Fprintln(w, "unset ANTHROPIC_BASE_URL")
	fmt.Fprintln(w, "unset ANTHROPIC_MODEL")
	fmt.Fprintln(w, "unset APIMGR_ACTIVE")
}

func writeShellExports(w io.Writer, apiConfig *models.APIConfig, activeAlias string) {
	if apiConfig.APIKey != "" {
		fmt.Fprintf(w, "export ANTHROPIC_API_KEY=%s\n", utils.ShellQuote(apiConfig.APIKey))
	} else if apiConfig.AuthToken != "" {
		fmt.Fprintf(w, "export ANTHROPIC_AUTH_TOKEN=%s\n", utils.ShellQuote(apiConfig.AuthToken))
	}
	if apiConfig.BaseURL != "" {
		fmt.Fprintf(w, "export ANTHROPIC_BASE_URL=%s\n", utils.ShellQuote(apiConfig.BaseURL))
	}
	if apiConfig.Model != "" {
		fmt.Fprintf(w, "export ANTHROPIC_MODEL=%s\n", utils.ShellQuote(apiConfig.Model))
	}
	fmt.Fprintf(w, "export APIMGR_ACTIVE=%s\n", utils.ShellQuote(activeAlias))
}
