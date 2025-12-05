package compatibility

import (
	"errors"
	"net/http"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: api-compatibility-test, Property 7: Error categorization**
// **Validates: Requirements 3.3**
//
// *For any* error response, the system SHALL categorize it into one of the defined
// categories: authentication_failure, model_not_found, rate_limit, format_incompatibility,
// or network_error.
func TestProperty7_ErrorCategorization(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Valid error categories that the system should produce
	validCategories := map[string]bool{
		ErrorCategoryAuthFailure:        true,
		ErrorCategoryModelNotFound:      true,
		ErrorCategoryRateLimit:          true,
		ErrorCategoryFormatIncompatible: true,
		ErrorCategoryNetworkError:       true,
		ErrorCategoryServerError:        true,
		ErrorCategoryEndpointNotFound:   true,
		ErrorCategoryUnknown:            true,
	}

	// Property: 401 and 403 status codes are categorized as authentication_failure
	properties.Property("401 and 403 status codes are categorized as authentication_failure", prop.ForAll(
		func(statusCode int, body string) bool {
			category := CategorizeError(statusCode, []byte(body))
			return category == ErrorCategoryAuthFailure
		},
		gen.OneConstOf(http.StatusUnauthorized, http.StatusForbidden),
		gen.AnyString(),
	))

	// Property: 404 with "model" in body is categorized as model_not_found
	properties.Property("404 with model in body is categorized as model_not_found", prop.ForAll(
		func(prefix string, suffix string) bool {
			// Generate body that contains "model" somewhere
			body := prefix + "model" + suffix
			category := CategorizeError(http.StatusNotFound, []byte(body))
			return category == ErrorCategoryModelNotFound
		},
		gen.AnyString(),
		gen.AnyString(),
	))

	// Property: 404 without "model" in body is categorized as endpoint_not_found
	properties.Property("404 without model in body is categorized as endpoint_not_found", prop.ForAll(
		func(body string) bool {
			// Filter out bodies that contain "model" (case insensitive)
			if containsModelKeyword(body) {
				return true // Skip this case, it's covered by another property
			}
			category := CategorizeError(http.StatusNotFound, []byte(body))
			return category == ErrorCategoryEndpointNotFound
		},
		gen.AnyString(),
	))

	// Property: 429 status code is categorized as rate_limit
	properties.Property("429 status code is categorized as rate_limit", prop.ForAll(
		func(body string) bool {
			category := CategorizeError(http.StatusTooManyRequests, []byte(body))
			return category == ErrorCategoryRateLimit
		},
		gen.AnyString(),
	))

	// Property: 200 status code is categorized as format_incompatibility
	properties.Property("200 status code is categorized as format_incompatibility", prop.ForAll(
		func(body string) bool {
			category := CategorizeError(http.StatusOK, []byte(body))
			return category == ErrorCategoryFormatIncompatible
		},
		gen.AnyString(),
	))

	// Property: 5xx status codes are categorized as server_error
	properties.Property("5xx status codes are categorized as server_error", prop.ForAll(
		func(statusCode int, body string) bool {
			category := CategorizeError(statusCode, []byte(body))
			return category == ErrorCategoryServerError
		},
		gen.IntRange(500, 599),
		gen.AnyString(),
	))

	// Property: Other status codes are categorized as unknown_error
	properties.Property("other status codes are categorized as unknown_error", prop.ForAll(
		func(statusCode int, body string) bool {
			// Skip status codes that have specific categories
			if statusCode == http.StatusUnauthorized ||
				statusCode == http.StatusForbidden ||
				statusCode == http.StatusNotFound ||
				statusCode == http.StatusTooManyRequests ||
				statusCode == http.StatusOK ||
				statusCode >= 500 {
				return true // Skip, covered by other properties
			}
			category := CategorizeError(statusCode, []byte(body))
			return category == ErrorCategoryUnknown
		},
		gen.IntRange(100, 599),
		gen.AnyString(),
	))

	// Property: All categorized errors return a valid category
	properties.Property("all errors return a valid category", prop.ForAll(
		func(statusCode int, body string) bool {
			category := CategorizeError(statusCode, []byte(body))
			return validCategories[category]
		},
		gen.IntRange(100, 599),
		gen.AnyString(),
	))

	// Property: CategorizeErrorWithInfo returns consistent category
	properties.Property("CategorizeErrorWithInfo returns consistent category", prop.ForAll(
		func(statusCode int, body string, errMsg string) bool {
			category := CategorizeError(statusCode, []byte(body))
			info := CategorizeErrorWithInfo(statusCode, []byte(body), errMsg)
			return info.Category == category && info.StatusCode == statusCode
		},
		gen.IntRange(100, 599),
		gen.AnyString(),
		gen.AnyString(),
	))

	// Property: CategorizeNetworkError always returns network_error category
	properties.Property("CategorizeNetworkError returns network_error category", prop.ForAll(
		func(errMsg string) bool {
			var err error
			if errMsg != "" {
				err = errors.New(errMsg)
			}
			info := CategorizeNetworkError(err)
			return info.Category == ErrorCategoryNetworkError && info.StatusCode == 0
		},
		gen.AnyString(),
	))

	// Property: GetUserMessage returns non-empty message for all valid categories
	properties.Property("GetUserMessage returns non-empty message for valid categories", prop.ForAll(
		func(category string) bool {
			msg := GetUserMessage(category)
			return msg != ""
		},
		gen.OneConstOf(
			ErrorCategoryAuthFailure,
			ErrorCategoryModelNotFound,
			ErrorCategoryRateLimit,
			ErrorCategoryFormatIncompatible,
			ErrorCategoryNetworkError,
			ErrorCategoryServerError,
			ErrorCategoryEndpointNotFound,
			ErrorCategoryUnknown,
		),
	))

	properties.TestingRun(t)
}

// containsModelKeyword checks if the string contains "model" (case insensitive)
func containsModelKeyword(s string) bool {
	lower := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			lower += string(c + 32)
		} else {
			lower += string(c)
		}
	}
	for i := 0; i <= len(lower)-5; i++ {
		if lower[i:i+5] == "model" {
			return true
		}
	}
	return false
}
