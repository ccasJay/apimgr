package compatibility

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: api-compatibility-test, Property 9: Result reporting based on check outcomes**
// **Validates: Requirements 5.1, 5.2, 5.3, 5.4**
//
// *For any* test result:
// - If all checks pass → compatibility level is "full" and exit code is 0
// - If any critical check fails → compatibility level is "none" and exit code is 1
// - If only non-critical checks fail → compatibility level is "partial" and exit code is 2
func TestProperty9_ResultReportingBasedOnCheckOutcomes(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for a single CheckResult
	checkResultGen := gopter.CombineGens(
		gen.Bool(),                                                  // Passed
		gen.Bool(),                                                  // Critical
		gen.AnyString(),                                             // Name
		gen.AnyString(),                                             // Message
	).Map(func(values []interface{}) CheckResult {
		return CheckResult{
			Passed:   values[0].(bool),
			Critical: values[1].(bool),
			Name:     values[2].(string),
			Message:  values[3].(string),
		}
	})

	// Generator for a slice of CheckResults (1 to 10 checks)
	checksGen := gen.SliceOfN(10, checkResultGen).SuchThat(func(checks []CheckResult) bool {
		return len(checks) > 0
	})

	// Property: If all checks pass, compatibility level is "full" and exit code is 0
	properties.Property("all checks pass implies full compatibility and exit code 0", prop.ForAll(
		func(checks []CheckResult) bool {
			// Make all checks pass
			for i := range checks {
				checks[i].Passed = true
			}

			level, exitCode := DetermineCompatibilityLevel(checks)
			return level == CompatibilityFull && exitCode == ExitCodeSuccess
		},
		checksGen,
	))

	// Property: If any critical check fails, compatibility level is "none" and exit code is 1
	properties.Property("critical failure implies no compatibility and exit code 1", prop.ForAll(
		func(checks []CheckResult, failIndex int) bool {
			if len(checks) == 0 {
				return true // Skip empty slices
			}

			// Make all checks pass first
			for i := range checks {
				checks[i].Passed = true
			}

			// Make one check fail critically
			idx := failIndex % len(checks)
			checks[idx].Passed = false
			checks[idx].Critical = true

			level, exitCode := DetermineCompatibilityLevel(checks)
			return level == CompatibilityNone && exitCode == ExitCodeFailure
		},
		checksGen,
		gen.IntRange(0, 100),
	))

	// Property: If only non-critical checks fail, compatibility level is "partial" and exit code is 2
	properties.Property("only non-critical failures implies partial compatibility and exit code 2", prop.ForAll(
		func(checks []CheckResult, failIndex int) bool {
			if len(checks) == 0 {
				return true // Skip empty slices
			}

			// Make all checks pass first
			for i := range checks {
				checks[i].Passed = true
				checks[i].Critical = false // Ensure no critical checks
			}

			// Make one check fail non-critically
			idx := failIndex % len(checks)
			checks[idx].Passed = false
			checks[idx].Critical = false

			level, exitCode := DetermineCompatibilityLevel(checks)
			return level == CompatibilityPartial && exitCode == ExitCodeWarning
		},
		checksGen,
		gen.IntRange(0, 100),
	))

	// Property: Empty checks returns "none" and exit code 1
	properties.Property("empty checks implies no compatibility and exit code 1", prop.ForAll(
		func(_ int) bool {
			level, exitCode := DetermineCompatibilityLevel([]CheckResult{})
			return level == CompatibilityNone && exitCode == ExitCodeFailure
		},
		gen.Int(),
	))

	// Property: Mixed failures with at least one critical failure returns "none" and exit code 1
	properties.Property("mixed failures with critical implies no compatibility", prop.ForAll(
		func(checks []CheckResult, criticalIdx int, nonCriticalIdx int) bool {
			if len(checks) < 2 {
				return true // Need at least 2 checks for mixed failures
			}

			// Make all checks pass first
			for i := range checks {
				checks[i].Passed = true
			}

			// Make one check fail critically
			critIdx := criticalIdx % len(checks)
			checks[critIdx].Passed = false
			checks[critIdx].Critical = true

			// Make another check fail non-critically (different index)
			nonCritIdx := nonCriticalIdx % len(checks)
			if nonCritIdx == critIdx {
				nonCritIdx = (nonCritIdx + 1) % len(checks)
			}
			checks[nonCritIdx].Passed = false
			checks[nonCritIdx].Critical = false

			level, exitCode := DetermineCompatibilityLevel(checks)
			return level == CompatibilityNone && exitCode == ExitCodeFailure
		},
		checksGen,
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t)
}
