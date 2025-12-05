package compatibility

import (
	"encoding/json"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: api-compatibility-test, Property 2: Response validation with error reporting**
// **Validates: Requirements 1.2, 1.3**
//
// *For any* API response body, the validator SHALL correctly identify whether the response
// structure is valid for the expected provider format, and for invalid responses, SHALL
// report all missing or malformed fields.
func TestProperty2_ResponseValidationWithErrorReporting(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for non-empty strings
	nonEmptyStringGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "default-value"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// Generator for positive integers
	positiveIntGen := gen.IntRange(1, 10000)

	// Property: Valid Anthropic responses are correctly identified as valid
	properties.Property("valid anthropic response is identified as valid", prop.ForAll(
		func(id string, model string, text string, inputTokens int, outputTokens int) bool {
			response := map[string]interface{}{
				"id":          id,
				"type":        "message",
				"role":        "assistant",
				"model":       model,
				"stop_reason": "end_turn",
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": text,
					},
				},
				"usage": map[string]interface{}{
					"input_tokens":  inputTokens,
					"output_tokens": outputTokens,
				},
			}

			body, err := json.Marshal(response)
			if err != nil {
				return false
			}

			validator := NewAnthropicValidator()
			result, err := validator.ValidateBasicResponse(body)
			if err != nil {
				return false
			}

			return result.Valid && result.HasContent && result.HasModel && result.HasUsage &&
				len(result.MissingFields) == 0
		},
		nonEmptyStringGen, // id
		nonEmptyStringGen, // model
		nonEmptyStringGen, // text
		positiveIntGen,    // inputTokens
		positiveIntGen,    // outputTokens
	))


	// Property: Anthropic responses missing content are identified as invalid with correct error
	properties.Property("anthropic response missing content reports missing field", prop.ForAll(
		func(id string, model string, inputTokens int, outputTokens int) bool {
			response := map[string]interface{}{
				"id":          id,
				"type":        "message",
				"role":        "assistant",
				"model":       model,
				"stop_reason": "end_turn",
				// content is missing
				"usage": map[string]interface{}{
					"input_tokens":  inputTokens,
					"output_tokens": outputTokens,
				},
			}

			body, err := json.Marshal(response)
			if err != nil {
				return false
			}

			validator := NewAnthropicValidator()
			result, err := validator.ValidateBasicResponse(body)
			if err != nil {
				return false
			}

			// Should be invalid and report missing content
			if result.Valid {
				return false
			}
			if result.HasContent {
				return false
			}

			// Check that "content" is in missing fields
			hasMissingContent := false
			for _, field := range result.MissingFields {
				if field == "content" {
					hasMissingContent = true
					break
				}
			}
			return hasMissingContent
		},
		nonEmptyStringGen, // id
		nonEmptyStringGen, // model
		positiveIntGen,    // inputTokens
		positiveIntGen,    // outputTokens
	))

	// Property: Anthropic responses missing model are identified as invalid with correct error
	properties.Property("anthropic response missing model reports missing field", prop.ForAll(
		func(id string, text string, inputTokens int, outputTokens int) bool {
			response := map[string]interface{}{
				"id":          id,
				"type":        "message",
				"role":        "assistant",
				"stop_reason": "end_turn",
				// model is missing
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": text,
					},
				},
				"usage": map[string]interface{}{
					"input_tokens":  inputTokens,
					"output_tokens": outputTokens,
				},
			}

			body, err := json.Marshal(response)
			if err != nil {
				return false
			}

			validator := NewAnthropicValidator()
			result, err := validator.ValidateBasicResponse(body)
			if err != nil {
				return false
			}

			// Should be invalid and report missing model
			if result.Valid {
				return false
			}
			if result.HasModel {
				return false
			}

			// Check that "model" is in missing fields
			hasMissingModel := false
			for _, field := range result.MissingFields {
				if field == "model" {
					hasMissingModel = true
					break
				}
			}
			return hasMissingModel
		},
		nonEmptyStringGen, // id
		nonEmptyStringGen, // text
		positiveIntGen,    // inputTokens
		positiveIntGen,    // outputTokens
	))


	// Property: Valid OpenAI responses are correctly identified as valid
	properties.Property("valid openai response is identified as valid", prop.ForAll(
		func(id string, model string, content string, promptTokens int, completionTokens int) bool {
			response := map[string]interface{}{
				"id":     id,
				"object": "chat.completion",
				"model":  model,
				"choices": []interface{}{
					map[string]interface{}{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": content,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     promptTokens,
					"completion_tokens": completionTokens,
					"total_tokens":      promptTokens + completionTokens,
				},
			}

			body, err := json.Marshal(response)
			if err != nil {
				return false
			}

			validator := NewOpenAIValidator()
			result, err := validator.ValidateBasicResponse(body)
			if err != nil {
				return false
			}

			return result.Valid && result.HasChoices && result.HasContent && result.HasModel &&
				result.HasUsage && len(result.MissingFields) == 0
		},
		nonEmptyStringGen, // id
		nonEmptyStringGen, // model
		nonEmptyStringGen, // content
		positiveIntGen,    // promptTokens
		positiveIntGen,    // completionTokens
	))

	// Property: OpenAI responses missing choices are identified as invalid with correct error
	properties.Property("openai response missing choices reports missing field", prop.ForAll(
		func(id string, model string, promptTokens int, completionTokens int) bool {
			response := map[string]interface{}{
				"id":     id,
				"object": "chat.completion",
				"model":  model,
				// choices is missing
				"usage": map[string]interface{}{
					"prompt_tokens":     promptTokens,
					"completion_tokens": completionTokens,
					"total_tokens":      promptTokens + completionTokens,
				},
			}

			body, err := json.Marshal(response)
			if err != nil {
				return false
			}

			validator := NewOpenAIValidator()
			result, err := validator.ValidateBasicResponse(body)
			if err != nil {
				return false
			}

			// Should be invalid and report missing choices
			if result.Valid {
				return false
			}
			if result.HasChoices {
				return false
			}

			// Check that "choices" is in missing fields
			hasMissingChoices := false
			for _, field := range result.MissingFields {
				if field == "choices" {
					hasMissingChoices = true
					break
				}
			}
			return hasMissingChoices
		},
		nonEmptyStringGen, // id
		nonEmptyStringGen, // model
		positiveIntGen,    // promptTokens
		positiveIntGen,    // completionTokens
	))


	// Property: OpenAI responses missing message.content are identified as invalid with correct error
	properties.Property("openai response missing message content reports missing field", prop.ForAll(
		func(id string, model string, promptTokens int, completionTokens int) bool {
			response := map[string]interface{}{
				"id":     id,
				"object": "chat.completion",
				"model":  model,
				"choices": []interface{}{
					map[string]interface{}{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							// content is missing
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     promptTokens,
					"completion_tokens": completionTokens,
					"total_tokens":      promptTokens + completionTokens,
				},
			}

			body, err := json.Marshal(response)
			if err != nil {
				return false
			}

			validator := NewOpenAIValidator()
			result, err := validator.ValidateBasicResponse(body)
			if err != nil {
				return false
			}

			// Should be invalid and report missing message.content
			if result.Valid {
				return false
			}
			if result.HasContent {
				return false
			}

			// Check that "choices[0].message.content" is in missing fields
			hasMissingContent := false
			for _, field := range result.MissingFields {
				if field == "choices[0].message.content" {
					hasMissingContent = true
					break
				}
			}
			return hasMissingContent
		},
		nonEmptyStringGen, // id
		nonEmptyStringGen, // model
		positiveIntGen,    // promptTokens
		positiveIntGen,    // completionTokens
	))

	// Property: Invalid JSON is correctly identified as invalid
	properties.Property("invalid json is identified as invalid", prop.ForAll(
		func(providerType string) bool {
			invalidJSON := []byte("{invalid json}")

			validator := NewValidator(providerType)
			result, err := validator.ValidateBasicResponse(invalidJSON)
			if err != nil {
				return false
			}

			// Should be invalid and report missing valid JSON structure
			if result.Valid {
				return false
			}

			hasMissingJSON := false
			for _, field := range result.MissingFields {
				if field == "valid JSON structure" {
					hasMissingJSON = true
					break
				}
			}
			return hasMissingJSON
		},
		gen.OneConstOf("anthropic", "openai"),
	))

	// Property: NewValidator returns correct validator type for each provider
	properties.Property("NewValidator returns correct validator type", prop.ForAll(
		func(providerType string) bool {
			validator := NewValidator(providerType)

			switch providerType {
			case "anthropic":
				_, ok := validator.(*AnthropicValidator)
				return ok
			case "openai":
				_, ok := validator.(*OpenAIValidator)
				return ok
			default:
				// Unknown providers default to OpenAI
				_, ok := validator.(*OpenAIValidator)
				return ok
			}
		},
		gen.OneConstOf("anthropic", "openai", "unknown", "custom"),
	))

	properties.TestingRun(t)
}
