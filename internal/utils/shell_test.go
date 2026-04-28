package utils

import "testing"

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: "''"},
		{name: "simple", value: "sk-test", want: "'sk-test'"},
		{name: "spaces", value: "https://api.example.com/v1 model", want: "'https://api.example.com/v1 model'"},
		{name: "single quote", value: "abc'def", want: `'abc'"'"'def'`},
		{name: "double quote", value: `abc"def`, want: `'abc"def'`},
		{name: "command substitution", value: "$(touch /tmp/pwned)", want: "'$(touch /tmp/pwned)'"},
		{name: "backticks", value: "`touch /tmp/pwned`", want: "'`touch /tmp/pwned`'"},
		{name: "newline", value: "line1\nline2", want: "'line1\nline2'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShellQuote(tt.value); got != tt.want {
				t.Fatalf("ShellQuote(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
