// Package compatibility provides API compatibility testing functionality
// for validating that API configurations work correctly with Claude Code.
package compatibility

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Reporter formats and outputs diagnostic results from compatibility tests.
type Reporter struct {
	jsonOutput bool
	verbose    bool
	writer     io.Writer
}

// ReporterOption is a functional option for configuring a Reporter
type ReporterOption func(*Reporter)

// WithJSONOutput enables JSON output format
func WithJSONOutput(jsonOutput bool) ReporterOption {
	return func(r *Reporter) {
		r.jsonOutput = jsonOutput
	}
}

// WithVerboseOutput enables verbose output
func WithVerboseOutput(verbose bool) ReporterOption {
	return func(r *Reporter) {
		r.verbose = verbose
	}
}

// WithWriter sets a custom writer for output
func WithWriter(w io.Writer) ReporterOption {
	return func(r *Reporter) {
		r.writer = w
	}
}

// NewReporter creates a new diagnostic reporter.
func NewReporter(writer io.Writer, opts ...ReporterOption) *Reporter {
	r := &Reporter{
		jsonOutput: false,
		verbose:    false,
		writer:     writer,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// DiagnosticOutput represents the structured output for JSON format
type DiagnosticOutput struct {
	ConnectionStatus     string        `json:"connectionStatus"`
	AuthenticationStatus string        `json:"authenticationStatus"`
	ResponseFormatValid  bool          `json:"responseFormatValid"`
	StreamingSupport     string        `json:"streamingSupport,omitempty"`
	CompatibilityLevel   string        `json:"compatibilityLevel"`
	Checks               []CheckResult `json:"checks"`
	ResponseTimeMs       int64         `json:"responseTimeMs"`
	Error                string        `json:"error,omitempty"`
}

// VerboseData holds request/response data for verbose output
type VerboseData struct {
	RequestBody  string `json:"requestBody,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`
}

// Report outputs the test result in the configured format.
func (r *Reporter) Report(result *TestResult) error {
	if r.jsonOutput {
		return r.reportJSON(result)
	}
	return r.reportText(result)
}


// ReportWithVerbose outputs the test result with optional verbose data.
func (r *Reporter) ReportWithVerbose(result *TestResult, verboseData *VerboseData) error {
	if r.jsonOutput {
		return r.reportJSONWithVerbose(result, verboseData)
	}
	return r.reportTextWithVerbose(result, verboseData)
}

// reportJSON outputs the result in JSON format
func (r *Reporter) reportJSON(result *TestResult) error {
	output := r.buildDiagnosticOutput(result)
	return r.writeJSON(output)
}

// reportJSONWithVerbose outputs the result in JSON format with verbose data
func (r *Reporter) reportJSONWithVerbose(result *TestResult, verboseData *VerboseData) error {
	output := r.buildDiagnosticOutput(result)
	
	// Create extended output with verbose data
	type ExtendedOutput struct {
		DiagnosticOutput
		Verbose *VerboseData `json:"verbose,omitempty"`
	}
	
	extOutput := ExtendedOutput{
		DiagnosticOutput: output,
	}
	
	if r.verbose && verboseData != nil {
		extOutput.Verbose = verboseData
	}
	
	return r.writeJSON(extOutput)
}

// buildDiagnosticOutput creates a DiagnosticOutput from TestResult
func (r *Reporter) buildDiagnosticOutput(result *TestResult) DiagnosticOutput {
	output := DiagnosticOutput{
		ConnectionStatus:     r.getConnectionStatus(result),
		AuthenticationStatus: r.getAuthenticationStatus(result),
		ResponseFormatValid:  r.getResponseFormatValid(result),
		StreamingSupport:     r.getStreamingSupport(result),
		CompatibilityLevel:   result.CompatibilityLevel,
		Checks:               result.Checks,
		ResponseTimeMs:       result.ResponseTime.Milliseconds(),
		Error:                result.Error,
	}
	return output
}

// writeJSON writes the output as JSON
func (r *Reporter) writeJSON(output interface{}) error {
	encoder := json.NewEncoder(r.writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// reportText outputs the result in human-readable text format
func (r *Reporter) reportText(result *TestResult) error {
	return r.reportTextWithVerbose(result, nil)
}

// reportTextWithVerbose outputs the result in text format with optional verbose data
func (r *Reporter) reportTextWithVerbose(result *TestResult, verboseData *VerboseData) error {
	var sb strings.Builder

	// Header with compatibility verdict
	sb.WriteString(r.getCompatibilityVerdict(result))
	sb.WriteString("\n\n")

	// Summary section
	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Connection:     %s\n", r.getConnectionStatusText(result)))
	sb.WriteString(fmt.Sprintf("  Authentication: %s\n", r.getAuthenticationStatusText(result)))
	sb.WriteString(fmt.Sprintf("  Response Format: %s\n", r.getResponseFormatText(result)))
	
	streamingSupport := r.getStreamingSupport(result)
	if streamingSupport != "" {
		sb.WriteString(fmt.Sprintf("  Streaming:      %s\n", streamingSupport))
	}
	
	sb.WriteString(fmt.Sprintf("  Response Time:  %dms\n", result.ResponseTime.Milliseconds()))
	sb.WriteString("\n")

	// Detailed checks
	sb.WriteString("Checks:\n")
	for _, check := range result.Checks {
		emoji := "✅"
		if !check.Passed {
			if check.Critical {
				emoji = "❌"
			} else {
				emoji = "⚠️"
			}
		}
		sb.WriteString(fmt.Sprintf("  %s %s: %s\n", emoji, check.Name, check.Message))
	}

	// Error details if present
	if result.Error != "" {
		sb.WriteString(fmt.Sprintf("\nError: %s\n", result.Error))
	}

	// Verbose output
	if r.verbose && verboseData != nil {
		sb.WriteString("\n--- Verbose Output ---\n")
		if verboseData.RequestBody != "" {
			sb.WriteString("\nRequest Body:\n")
			sb.WriteString(verboseData.RequestBody)
			sb.WriteString("\n")
		}
		if verboseData.ResponseBody != "" {
			sb.WriteString("\nResponse Body:\n")
			sb.WriteString(verboseData.ResponseBody)
			sb.WriteString("\n")
		}
	}

	_, err := r.writer.Write([]byte(sb.String()))
	return err
}


// getCompatibilityVerdict returns the main verdict message with emoji
func (r *Reporter) getCompatibilityVerdict(result *TestResult) string {
	switch result.CompatibilityLevel {
	case CompatibilityFull:
		return "✅ API is compatible with Claude Code"
	case CompatibilityPartial:
		return "⚠️ API may have compatibility issues"
	case CompatibilityNone:
		return "❌ API is NOT compatible with Claude Code"
	default:
		return "❓ Unknown compatibility status"
	}
}

// getConnectionStatus returns the connection status for JSON output
func (r *Reporter) getConnectionStatus(result *TestResult) string {
	for _, check := range result.Checks {
		if check.Name == "Connection" || check.Name == "Streaming Connection" {
			if check.Passed {
				return "connected"
			}
			return "failed"
		}
	}
	return "unknown"
}

// getConnectionStatusText returns the connection status for text output
func (r *Reporter) getConnectionStatusText(result *TestResult) string {
	for _, check := range result.Checks {
		if check.Name == "Connection" || check.Name == "Streaming Connection" {
			if check.Passed {
				return "✅ Connected"
			}
			return "❌ Failed"
		}
	}
	return "❓ Unknown"
}

// getAuthenticationStatus returns the authentication status for JSON output
func (r *Reporter) getAuthenticationStatus(result *TestResult) string {
	for _, check := range result.Checks {
		if check.Name == "Authentication" || check.Name == "Streaming Authentication" {
			if check.Passed {
				return "authenticated"
			}
			return "failed"
		}
	}
	return "unknown"
}

// getAuthenticationStatusText returns the authentication status for text output
func (r *Reporter) getAuthenticationStatusText(result *TestResult) string {
	for _, check := range result.Checks {
		if check.Name == "Authentication" || check.Name == "Streaming Authentication" {
			if check.Passed {
				return "✅ Authenticated"
			}
			return "❌ Failed"
		}
	}
	return "❓ Unknown"
}

// getResponseFormatValid returns whether the response format is valid
func (r *Reporter) getResponseFormatValid(result *TestResult) bool {
	for _, check := range result.Checks {
		if check.Name == "Response Format" {
			return check.Passed
		}
	}
	return false
}

// getResponseFormatText returns the response format status for text output
func (r *Reporter) getResponseFormatText(result *TestResult) string {
	for _, check := range result.Checks {
		if check.Name == "Response Format" {
			if check.Passed {
				return "✅ Valid"
			}
			return "❌ Invalid"
		}
	}
	return "❓ Not tested"
}

// getStreamingSupport returns the streaming support status
func (r *Reporter) getStreamingSupport(result *TestResult) string {
	hasStreamingCheck := false
	sseValid := false
	completionSignal := false

	for _, check := range result.Checks {
		switch check.Name {
		case "SSE Format":
			hasStreamingCheck = true
			sseValid = check.Passed
		case "Completion Signal":
			completionSignal = check.Passed
		}
	}

	if !hasStreamingCheck {
		return ""
	}

	if sseValid && completionSignal {
		return "✅ Full support"
	} else if sseValid {
		return "⚠️ Partial (no completion signal)"
	}
	return "❌ Not supported"
}

// ReportError outputs an error with its category
func (r *Reporter) ReportError(err error, category string) error {
	if r.jsonOutput {
		output := map[string]string{
			"error":    err.Error(),
			"category": category,
		}
		return r.writeJSON(output)
	}

	emoji := "❌"
	_, writeErr := fmt.Fprintf(r.writer, "%s Error [%s]: %s\n", emoji, category, err.Error())
	return writeErr
}

// IsJSONOutput returns whether JSON output is enabled
func (r *Reporter) IsJSONOutput() bool {
	return r.jsonOutput
}

// IsVerbose returns whether verbose output is enabled
func (r *Reporter) IsVerbose() bool {
	return r.verbose
}
