// Package compatibility provides API compatibility testing functionality
// for validating that API configurations work correctly with Claude Code.
package compatibility

import "time"

// Error category constants for categorizing API errors
const (
	ErrorCategoryAuthFailure        = "authentication_failure"
	ErrorCategoryModelNotFound      = "model_not_found"
	ErrorCategoryRateLimit          = "rate_limit"
	ErrorCategoryFormatIncompatible = "format_incompatibility"
	ErrorCategoryNetworkError       = "network_error"
	ErrorCategoryServerError        = "server_error"
	ErrorCategoryEndpointNotFound   = "endpoint_not_found"
	ErrorCategoryUnknown            = "unknown_error"
)

// Compatibility level constants
const (
	CompatibilityFull    = "full"
	CompatibilityPartial = "partial"
	CompatibilityNone    = "none"
)

// Exit code constants
const (
	ExitCodeSuccess = 0
	ExitCodeFailure = 1
	ExitCodeWarning = 2
)

// TestResult represents the overall result of a compatibility test
type TestResult struct {
	Success            bool          `json:"success"`
	CompatibilityLevel string        `json:"compatibilityLevel"` // "full", "partial", "none"
	Checks             []CheckResult `json:"checks"`
	ResponseTime       time.Duration `json:"responseTimeMs"`
	Error              string        `json:"error,omitempty"`
}

// CheckResult represents the result of a single validation check
type CheckResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	Critical bool   `json:"critical"`
}

// ValidationResult represents the result of response validation
type ValidationResult struct {
	Valid            bool     `json:"valid"`
	HasChoices       bool     `json:"hasChoices"`       // OpenAI format
	HasContent       bool     `json:"hasContent"`       // Anthropic format
	HasModel         bool     `json:"hasModel"`
	HasUsage         bool     `json:"hasUsage"`
	MissingFields    []string `json:"missingFields,omitempty"`
	UnexpectedFields []string `json:"unexpectedFields,omitempty"`
}

// DetermineCompatibilityLevel determines the compatibility level based on check results.
// Returns the compatibility level and the appropriate exit code.
// - If all checks pass → "full" and exit code 0
// - If any critical check fails → "none" and exit code 1
// - If only non-critical checks fail → "partial" and exit code 2
func DetermineCompatibilityLevel(checks []CheckResult) (string, int) {
	if len(checks) == 0 {
		return CompatibilityNone, ExitCodeFailure
	}

	hasCriticalFailure := false
	hasNonCriticalFailure := false
	allPassed := true

	for _, check := range checks {
		if !check.Passed {
			allPassed = false
			if check.Critical {
				hasCriticalFailure = true
			} else {
				hasNonCriticalFailure = true
			}
		}
	}

	if allPassed {
		return CompatibilityFull, ExitCodeSuccess
	}

	if hasCriticalFailure {
		return CompatibilityNone, ExitCodeFailure
	}

	if hasNonCriticalFailure {
		return CompatibilityPartial, ExitCodeWarning
	}

	// Should not reach here, but default to none
	return CompatibilityNone, ExitCodeFailure
}
