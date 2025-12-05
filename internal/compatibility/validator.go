// Package compatibility provides API compatibility testing functionality
// for validating that API configurations work correctly with Claude Code.
package compatibility

import (
	"encoding/json"
	"io"
)

// ResponseValidator defines the interface for validating provider-specific API responses
type ResponseValidator interface {
	// ValidateBasicResponse validates a non-streaming response body
	ValidateBasicResponse(body []byte) (*ValidationResult, error)
	// ValidateStreamingResponse validates a streaming response from a reader
	ValidateStreamingResponse(reader io.Reader) (*ValidationResult, error)
}

// AnthropicResponse represents the expected Anthropic Messages API response structure
type AnthropicResponse struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Role       string                   `json:"role"`
	Content    []AnthropicContentBlock  `json:"content"`
	Model      string                   `json:"model"`
	StopReason string                   `json:"stop_reason"`
	Usage      *AnthropicUsage          `json:"usage"`
}

// AnthropicContentBlock represents a content block in Anthropic response
type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage represents token usage in Anthropic response
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicValidator validates responses from the Anthropic Messages API
type AnthropicValidator struct{}

// NewAnthropicValidator creates a new AnthropicValidator
func NewAnthropicValidator() *AnthropicValidator {
	return &AnthropicValidator{}
}


// ValidateBasicResponse validates a non-streaming Anthropic response
func (v *AnthropicValidator) ValidateBasicResponse(body []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:         true,
		MissingFields: []string{},
	}

	// Try to parse as JSON first
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		result.Valid = false
		result.MissingFields = append(result.MissingFields, "valid JSON structure")
		return result, nil
	}

	// Check for content array (required for Anthropic)
	if content, ok := rawResponse["content"]; ok {
		if contentArr, isArray := content.([]interface{}); isArray && len(contentArr) > 0 {
			result.HasContent = true
			// Validate content block structure
			for i, block := range contentArr {
				if blockMap, isMap := block.(map[string]interface{}); isMap {
					if _, hasType := blockMap["type"]; !hasType {
						result.MissingFields = append(result.MissingFields, 
							"content["+string(rune('0'+i))+"].type")
					}
				}
			}
		} else {
			result.MissingFields = append(result.MissingFields, "content (non-empty array)")
		}
	} else {
		result.MissingFields = append(result.MissingFields, "content")
	}

	// Check for model field
	if model, ok := rawResponse["model"]; ok {
		if modelStr, isString := model.(string); isString && modelStr != "" {
			result.HasModel = true
		} else {
			result.MissingFields = append(result.MissingFields, "model (non-empty string)")
		}
	} else {
		result.MissingFields = append(result.MissingFields, "model")
	}

	// Check for usage field
	if usage, ok := rawResponse["usage"]; ok {
		if usageMap, isMap := usage.(map[string]interface{}); isMap {
			result.HasUsage = true
			// Validate usage structure
			if _, hasInput := usageMap["input_tokens"]; !hasInput {
				result.MissingFields = append(result.MissingFields, "usage.input_tokens")
			}
			if _, hasOutput := usageMap["output_tokens"]; !hasOutput {
				result.MissingFields = append(result.MissingFields, "usage.output_tokens")
			}
		} else {
			result.MissingFields = append(result.MissingFields, "usage (object)")
		}
	} else {
		result.MissingFields = append(result.MissingFields, "usage")
	}

	// Determine overall validity
	result.Valid = result.HasContent && result.HasModel && len(result.MissingFields) == 0

	return result, nil
}

// ValidateStreamingResponse validates a streaming Anthropic response
func (v *AnthropicValidator) ValidateStreamingResponse(reader io.Reader) (*ValidationResult, error) {
	// Streaming validation will be implemented in the SSE validator task
	// For now, return a basic result
	return &ValidationResult{
		Valid:      true,
		HasContent: true,
	}, nil
}


// OpenAIResponse represents the expected OpenAI Chat Completions API response structure
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Choices []OpenAIChoice `json:"choices"`
	Model   string         `json:"model"`
	Usage   *OpenAIUsage   `json:"usage"`
}

// OpenAIChoice represents a choice in OpenAI response
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIMessage represents a message in OpenAI response
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIUsage represents token usage in OpenAI response
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIValidator validates responses from the OpenAI Chat Completions API
type OpenAIValidator struct{}

// NewOpenAIValidator creates a new OpenAIValidator
func NewOpenAIValidator() *OpenAIValidator {
	return &OpenAIValidator{}
}

// ValidateBasicResponse validates a non-streaming OpenAI response
func (v *OpenAIValidator) ValidateBasicResponse(body []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:         true,
		MissingFields: []string{},
	}

	// Try to parse as JSON first
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		result.Valid = false
		result.MissingFields = append(result.MissingFields, "valid JSON structure")
		return result, nil
	}

	// Check for choices array (required for OpenAI)
	if choices, ok := rawResponse["choices"]; ok {
		if choicesArr, isArray := choices.([]interface{}); isArray && len(choicesArr) > 0 {
			result.HasChoices = true
			// Validate first choice structure
			if firstChoice, isMap := choicesArr[0].(map[string]interface{}); isMap {
				// Check for message.content
				if message, hasMessage := firstChoice["message"]; hasMessage {
					if msgMap, isMsgMap := message.(map[string]interface{}); isMsgMap {
						if _, hasContent := msgMap["content"]; !hasContent {
							result.MissingFields = append(result.MissingFields, "choices[0].message.content")
						} else {
							result.HasContent = true
						}
					} else {
						result.MissingFields = append(result.MissingFields, "choices[0].message (object)")
					}
				} else {
					result.MissingFields = append(result.MissingFields, "choices[0].message")
				}
			}
		} else {
			result.MissingFields = append(result.MissingFields, "choices (non-empty array)")
		}
	} else {
		result.MissingFields = append(result.MissingFields, "choices")
	}

	// Check for model field
	if model, ok := rawResponse["model"]; ok {
		if modelStr, isString := model.(string); isString && modelStr != "" {
			result.HasModel = true
		} else {
			result.MissingFields = append(result.MissingFields, "model (non-empty string)")
		}
	} else {
		result.MissingFields = append(result.MissingFields, "model")
	}

	// Check for usage field (optional but recommended)
	if usage, ok := rawResponse["usage"]; ok {
		if usageMap, isMap := usage.(map[string]interface{}); isMap {
			result.HasUsage = true
			// Validate usage structure
			if _, hasPrompt := usageMap["prompt_tokens"]; !hasPrompt {
				result.MissingFields = append(result.MissingFields, "usage.prompt_tokens")
			}
			if _, hasCompletion := usageMap["completion_tokens"]; !hasCompletion {
				result.MissingFields = append(result.MissingFields, "usage.completion_tokens")
			}
		} else {
			result.MissingFields = append(result.MissingFields, "usage (object)")
		}
	} else {
		result.MissingFields = append(result.MissingFields, "usage")
	}

	// Determine overall validity - choices with message.content and model are required
	result.Valid = result.HasChoices && result.HasContent && result.HasModel && len(result.MissingFields) == 0

	return result, nil
}

// ValidateStreamingResponse validates a streaming OpenAI response
func (v *OpenAIValidator) ValidateStreamingResponse(reader io.Reader) (*ValidationResult, error) {
	// Streaming validation will be implemented in the SSE validator task
	// For now, return a basic result
	return &ValidationResult{
		Valid:      true,
		HasChoices: true,
		HasContent: true,
	}, nil
}

// NewValidator creates a new ResponseValidator based on the provider type
func NewValidator(providerType string) ResponseValidator {
	switch providerType {
	case "anthropic":
		return NewAnthropicValidator()
	case "openai":
		return NewOpenAIValidator()
	default:
		// Default to OpenAI-compatible format for unknown providers
		return NewOpenAIValidator()
	}
}
