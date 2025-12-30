package compatibility

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: api-compatibility-test, Property 6: Output format completeness**
// **Validates: Requirements 3.1, 3.2**
//
// *For any* test result, the output (text or JSON) SHALL contain all required
// diagnostic fields: connection status, authentication status, response format
// validity, and compatibility level.
func TestProperty6_OutputFormatCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for CheckResult
	checkResultGen := gopter.CombineGens(
		gen.AnyString(),
		gen.Bool(),
		gen.AnyString(),
		gen.Bool(),
	).Map(func(values []interface{}) CheckResult {
		return CheckResult{
			Name:     values[0].(string),
			Passed:   values[1].(bool),
			Message:  values[2].(string),
			Critical: values[3].(bool),
		}
	})

	// Generator for standard check names that we need to verify
	standardChecksGen := gopter.CombineGens(
		gen.Bool(), // Connection passed
		gen.Bool(), // Authentication passed
		gen.Bool(), // Response Format passed
	).Map(func(values []interface{}) []CheckResult {
		return []CheckResult{
			{Name: "Connection", Passed: values[0].(bool), Message: "Connection check", Critical: true},
			{Name: "Authentication", Passed: values[1].(bool), Message: "Auth check", Critical: true},
			{Name: "Response Format", Passed: values[2].(bool), Message: "Format check", Critical: true},
		}
	})

	// Generator for additional random checks
	additionalChecksGen := gen.SliceOfN(5, checkResultGen)

	// Generator for compatibility level
	compatLevelGen := gen.OneConstOf(CompatibilityFull, CompatibilityPartial, CompatibilityNone)

	// Generator for response time
	responseTimeGen := gen.Int64Range(0, 10000).Map(func(ms int64) time.Duration {
		return time.Duration(ms) * time.Millisecond
	})

	// Generator for optional error string
	errorGen := gen.OneGenOf(
		gen.Const(""),
		gen.AnyString(),
	)


	// Property: JSON output contains all required diagnostic fields
	properties.Property("JSON output contains all required diagnostic fields", prop.ForAll(
		func(standardChecks []CheckResult, additionalChecks []CheckResult, compatLevel string, responseTime time.Duration, errStr string) bool {
			// Combine checks
			allChecks := append(standardChecks, additionalChecks...)

			result := &TestResult{
				Success:            compatLevel == CompatibilityFull,
				CompatibilityLevel: compatLevel,
				Checks:             allChecks,
				ResponseTime:       responseTime,
				Error:              errStr,
			}

			var buf bytes.Buffer
			reporter := NewReporter(&buf, WithJSONOutput(true))
			err := reporter.Report(result)
			if err != nil {
				return false
			}

			// Parse JSON output
			var output DiagnosticOutput
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				return false
			}

			// Verify all required fields are present
			hasConnectionStatus := output.ConnectionStatus != ""
			hasAuthStatus := output.AuthenticationStatus != ""
			hasCompatLevel := output.CompatibilityLevel != ""
			// ResponseFormatValid is a bool, so it's always present

			return hasConnectionStatus && hasAuthStatus && hasCompatLevel
		},
		standardChecksGen,
		additionalChecksGen,
		compatLevelGen,
		responseTimeGen,
		errorGen,
	))

	// Property: Text output contains all required diagnostic information
	properties.Property("text output contains all required diagnostic information", prop.ForAll(
		func(standardChecks []CheckResult, additionalChecks []CheckResult, compatLevel string, responseTime time.Duration, errStr string) bool {
			// Combine checks
			allChecks := append(standardChecks, additionalChecks...)

			result := &TestResult{
				Success:            compatLevel == CompatibilityFull,
				CompatibilityLevel: compatLevel,
				Checks:             allChecks,
				ResponseTime:       responseTime,
				Error:              errStr,
			}

			var buf bytes.Buffer
			reporter := NewReporter(&buf, WithJSONOutput(false))
			err := reporter.Report(result)
			if err != nil {
				return false
			}

			output := buf.String()

			// Verify text output contains required sections
			hasConnectionInfo := strings.Contains(output, "Connection:")
			hasAuthInfo := strings.Contains(output, "Authentication:")
			hasFormatInfo := strings.Contains(output, "Response Format:")
			// Check for compatibility verdict - all verdicts contain "compat" (compatible, compatibility)
			hasCompatVerdict := strings.Contains(strings.ToLower(output), "compat")

			return hasConnectionInfo && hasAuthInfo && hasFormatInfo && hasCompatVerdict
		},
		standardChecksGen,
		additionalChecksGen,
		compatLevelGen,
		responseTimeGen,
		errorGen,
	))

	// Property: JSON output compatibility level matches input
	properties.Property("JSON output compatibility level matches input", prop.ForAll(
		func(standardChecks []CheckResult, compatLevel string, responseTime time.Duration) bool {
			result := &TestResult{
				Success:            compatLevel == CompatibilityFull,
				CompatibilityLevel: compatLevel,
				Checks:             standardChecks,
				ResponseTime:       responseTime,
			}

			var buf bytes.Buffer
			reporter := NewReporter(&buf, WithJSONOutput(true))
			err := reporter.Report(result)
			if err != nil {
				return false
			}

			var output DiagnosticOutput
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				return false
			}

			return output.CompatibilityLevel == compatLevel
		},
		standardChecksGen,
		compatLevelGen,
		responseTimeGen,
	))

	// Property: JSON output response time matches input
	properties.Property("JSON output response time matches input", prop.ForAll(
		func(standardChecks []CheckResult, responseTime time.Duration) bool {
			result := &TestResult{
				Success:            true,
				CompatibilityLevel: CompatibilityFull,
				Checks:             standardChecks,
				ResponseTime:       responseTime,
			}

			var buf bytes.Buffer
			reporter := NewReporter(&buf, WithJSONOutput(true))
			err := reporter.Report(result)
			if err != nil {
				return false
			}

			var output DiagnosticOutput
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				return false
			}

			return output.ResponseTimeMs == responseTime.Milliseconds()
		},
		standardChecksGen,
		responseTimeGen,
	))

	properties.TestingRun(t)
}


// TestReporterTextOutput tests basic text output functionality
func TestReporterTextOutput(t *testing.T) {
	result := &TestResult{
		Success:            true,
		CompatibilityLevel: CompatibilityFull,
		Checks: []CheckResult{
			{Name: "Connection", Passed: true, Message: "Connected successfully", Critical: true},
			{Name: "Authentication", Passed: true, Message: "Auth successful", Critical: true},
			{Name: "Response Format", Passed: true, Message: "Format valid", Critical: true},
		},
		ResponseTime: 150 * time.Millisecond,
	}

	var buf bytes.Buffer
	reporter := NewReporter(&buf)
	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	output := buf.String()

	// Check for success verdict
	if !strings.Contains(output, "✅ API is compatible with Claude Code") {
		t.Error("Expected success verdict in output")
	}

	// Check for summary section
	if !strings.Contains(output, "Summary:") {
		t.Error("Expected Summary section in output")
	}

	// Check for checks section
	if !strings.Contains(output, "Checks:") {
		t.Error("Expected Checks section in output")
	}
}

// TestReporterJSONOutput tests basic JSON output functionality
func TestReporterJSONOutput(t *testing.T) {
	result := &TestResult{
		Success:            false,
		CompatibilityLevel: CompatibilityNone,
		Checks: []CheckResult{
			{Name: "Connection", Passed: true, Message: "Connected", Critical: true},
			{Name: "Authentication", Passed: false, Message: "Auth failed", Critical: true},
		},
		ResponseTime: 200 * time.Millisecond,
		Error:        "Authentication failed",
	}

	var buf bytes.Buffer
	reporter := NewReporter(&buf, WithJSONOutput(true))
	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	var output DiagnosticOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if output.CompatibilityLevel != CompatibilityNone {
		t.Errorf("Expected compatibility level 'none', got '%s'", output.CompatibilityLevel)
	}

	if output.ConnectionStatus != "connected" {
		t.Errorf("Expected connection status 'connected', got '%s'", output.ConnectionStatus)
	}

	if output.AuthenticationStatus != "failed" {
		t.Errorf("Expected authentication status 'failed', got '%s'", output.AuthenticationStatus)
	}
}

// TestReporterVerboseOutput tests verbose mode output
func TestReporterVerboseOutput(t *testing.T) {
	result := &TestResult{
		Success:            true,
		CompatibilityLevel: CompatibilityFull,
		Checks: []CheckResult{
			{Name: "Connection", Passed: true, Message: "Connected", Critical: true},
			{Name: "Authentication", Passed: true, Message: "Auth OK", Critical: true},
			{Name: "Response Format", Passed: true, Message: "Format OK", Critical: true},
		},
		ResponseTime: 100 * time.Millisecond,
	}

	verboseData := &VerboseData{
		RequestBody:  `{"model": "test", "messages": []}`,
		ResponseBody: `{"content": "test response"}`,
	}

	var buf bytes.Buffer
	reporter := NewReporter(&buf, WithVerboseOutput(true))
	err := reporter.ReportWithVerbose(result, verboseData)
	if err != nil {
		t.Fatalf("ReportWithVerbose failed: %v", err)
	}

	output := buf.String()

	// Check for verbose section
	if !strings.Contains(output, "--- Verbose Output ---") {
		t.Error("Expected verbose output section")
	}

	// Check for request body
	if !strings.Contains(output, "Request Body:") {
		t.Error("Expected request body in verbose output")
	}

	// Check for response body
	if !strings.Contains(output, "Response Body:") {
		t.Error("Expected response body in verbose output")
	}
}

// TestReporterVerboseJSONOutput tests verbose mode with JSON output
func TestReporterVerboseJSONOutput(t *testing.T) {
	result := &TestResult{
		Success:            true,
		CompatibilityLevel: CompatibilityFull,
		Checks: []CheckResult{
			{Name: "Connection", Passed: true, Message: "Connected", Critical: true},
			{Name: "Authentication", Passed: true, Message: "Auth OK", Critical: true},
			{Name: "Response Format", Passed: true, Message: "Format OK", Critical: true},
		},
		ResponseTime: 100 * time.Millisecond,
	}

	verboseData := &VerboseData{
		RequestBody:  `{"model": "test"}`,
		ResponseBody: `{"content": "response"}`,
	}

	var buf bytes.Buffer
	reporter := NewReporter(&buf, WithJSONOutput(true), WithVerboseOutput(true))
	err := reporter.ReportWithVerbose(result, verboseData)
	if err != nil {
		t.Fatalf("ReportWithVerbose failed: %v", err)
	}

	// Parse the extended output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Check for verbose field
	verbose, ok := output["verbose"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected verbose field in JSON output")
	}

	if verbose["requestBody"] != `{"model": "test"}` {
		t.Error("Expected request body in verbose JSON output")
	}

	if verbose["responseBody"] != `{"content": "response"}` {
		t.Error("Expected response body in verbose JSON output")
	}
}

// TestReporterPartialCompatibility tests partial compatibility verdict
func TestReporterPartialCompatibility(t *testing.T) {
	result := &TestResult{
		Success:            false,
		CompatibilityLevel: CompatibilityPartial,
		Checks: []CheckResult{
			{Name: "Connection", Passed: true, Message: "Connected", Critical: true},
			{Name: "Authentication", Passed: true, Message: "Auth OK", Critical: true},
			{Name: "Response Format", Passed: true, Message: "Format OK", Critical: true},
			{Name: "Completion Signal", Passed: false, Message: "No signal", Critical: false},
		},
		ResponseTime: 100 * time.Millisecond,
	}

	var buf bytes.Buffer
	reporter := NewReporter(&buf)
	err := reporter.Report(result)
	if err != nil {
		t.Fatalf("Report failed: %v", err)
	}

	output := buf.String()

	// Check for warning verdict
	if !strings.Contains(output, "⚠️ API may have compatibility issues") {
		t.Error("Expected warning verdict in output")
	}
}

// TestReporterError tests error reporting
func TestReporterError(t *testing.T) {
	var buf bytes.Buffer
	reporter := NewReporter(&buf)

	err := reporter.ReportError(
		&testError{msg: "connection refused"},
		ErrorCategoryNetworkError,
	)
	if err != nil {
		t.Fatalf("ReportError failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "❌") {
		t.Error("Expected error emoji in output")
	}

	if !strings.Contains(output, ErrorCategoryNetworkError) {
		t.Error("Expected error category in output")
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
