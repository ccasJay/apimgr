package utils

import "strings"

// ShellQuote returns a POSIX shell single-quoted literal safe for eval.
func ShellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
