package compatibility

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: api-compatibility-test, Property 5: SSE format validation**
// **Validates: Requirements 2.2, 2.3**
//
// *For any* SSE stream, the validator SHALL correctly identify whether each event line
// is properly formatted (starts with `data:` or `event:` prefix), and SHALL detect
// the completion signal (`[DONE]` for OpenAI, `message_stop` for Anthropic).
func TestProperty5_SSEFormatValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for non-empty strings (for content)
	nonEmptyStringGen := gen.AlphaString().Map(func(s string) string {
		if len(s) == 0 {
			return "content"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// Generator for valid Anthropic SSE stream with message_stop completion
	anthropicStreamGen := func(content string, eventCount int) string {
		var sb strings.Builder

		// message_start event
		sb.WriteString("event: message_start\n")
		sb.WriteString(`data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant"}}`)
		sb.WriteString("\n\n")

		// content_block_delta events
		for i := 0; i < eventCount; i++ {
			sb.WriteString("event: content_block_delta\n")
			data := map[string]interface{}{
				"type": "content_block_delta",
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": content,
				},
			}
			jsonData, _ := json.Marshal(data)
			sb.WriteString("data: ")
			sb.WriteString(string(jsonData))
			sb.WriteString("\n\n")
		}

		// message_stop event (completion signal)
		sb.WriteString("event: message_stop\n")
		sb.WriteString(`data: {"type":"message_stop"}`)
		sb.WriteString("\n\n")

		return sb.String()
	}

	// Generator for valid OpenAI SSE stream with [DONE] completion
	openAIStreamGen := func(content string, eventCount int) string {
		var sb strings.Builder

		// Delta events
		for i := 0; i < eventCount; i++ {
			data := map[string]interface{}{
				"id":     "chatcmpl-123",
				"object": "chat.completion.chunk",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": content,
						},
					},
				},
			}
			jsonData, _ := json.Marshal(data)
			sb.WriteString("data: ")
			sb.WriteString(string(jsonData))
			sb.WriteString("\n\n")
		}

		// [DONE] completion signal
		sb.WriteString("data: [DONE]\n\n")

		return sb.String()
	}

	// Property: Valid Anthropic SSE stream with message_stop is detected as valid with completion signal
	properties.Property("valid anthropic stream with message_stop is valid", prop.ForAll(
		func(content string, eventCount int) bool {
			// Ensure at least 1 event
			if eventCount < 1 {
				eventCount = 1
			}
			if eventCount > 5 {
				eventCount = 5
			}

			stream := anthropicStreamGen(content, eventCount)
			reader := strings.NewReader(stream)

			validator := NewAnthropicSSEValidator()
			result, err := validator.ValidateStream(reader)
			if err != nil {
				return false
			}

			// Should be valid with completion signal
			return result.Valid &&
				result.HasCompletionSignal &&
				result.CompletionType == "message_stop" &&
				result.EventCount > 0
		},
		nonEmptyStringGen,
		gen.IntRange(1, 5),
	))

	// Property: Valid OpenAI SSE stream with [DONE] is detected as valid with completion signal
	properties.Property("valid openai stream with done is valid", prop.ForAll(
		func(content string, eventCount int) bool {
			// Ensure at least 1 event
			if eventCount < 1 {
				eventCount = 1
			}
			if eventCount > 5 {
				eventCount = 5
			}

			stream := openAIStreamGen(content, eventCount)
			reader := strings.NewReader(stream)

			validator := NewOpenAISSEValidator()
			result, err := validator.ValidateStream(reader)
			if err != nil {
				return false
			}

			// Should be valid with completion signal
			return result.Valid &&
				result.HasCompletionSignal &&
				result.CompletionType == "done" &&
				result.EventCount > 0
		},
		nonEmptyStringGen,
		gen.IntRange(1, 5),
	))

	// Property: Anthropic stream without message_stop is detected as missing completion signal
	properties.Property("anthropic stream without message_stop has no completion signal", prop.ForAll(
		func(content string) bool {
			// Create stream without message_stop
			var sb strings.Builder
			sb.WriteString("event: message_start\n")
			sb.WriteString(`data: {"type":"message_start","message":{"id":"msg_123"}}`)
			sb.WriteString("\n\n")
			sb.WriteString("event: content_block_delta\n")
			data := map[string]interface{}{
				"type": "content_block_delta",
				"delta": map[string]interface{}{
					"type": "text_delta",
					"text": content,
				},
			}
			jsonData, _ := json.Marshal(data)
			sb.WriteString("data: ")
			sb.WriteString(string(jsonData))
			sb.WriteString("\n\n")
			// No message_stop event

			reader := strings.NewReader(sb.String())
			validator := NewAnthropicSSEValidator()
			result, err := validator.ValidateStream(reader)
			if err != nil {
				return false
			}

			// Should not have completion signal
			return !result.HasCompletionSignal && !result.Valid
		},
		nonEmptyStringGen,
	))

	// Property: OpenAI stream without [DONE] is detected as missing completion signal
	properties.Property("openai stream without done has no completion signal", prop.ForAll(
		func(content string) bool {
			// Create stream without [DONE]
			var sb strings.Builder
			data := map[string]interface{}{
				"id":     "chatcmpl-123",
				"object": "chat.completion.chunk",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": content,
						},
					},
				},
			}
			jsonData, _ := json.Marshal(data)
			sb.WriteString("data: ")
			sb.WriteString(string(jsonData))
			sb.WriteString("\n\n")
			// No [DONE] event

			reader := strings.NewReader(sb.String())
			validator := NewOpenAISSEValidator()
			result, err := validator.ValidateStream(reader)
			if err != nil {
				return false
			}

			// Should not have completion signal
			return !result.HasCompletionSignal && !result.Valid
		},
		nonEmptyStringGen,
	))

	// Property: SSE events with data: prefix are correctly parsed
	properties.Property("sse events with data prefix are parsed", prop.ForAll(
		func(content string) bool {
			stream := "data: " + content + "\n\n"
			reader := strings.NewReader(stream)

			parser := NewSSEParser(reader)
			events, err := parser.ParseAll()
			if err != nil {
				return false
			}

			return len(events) == 1 && events[0].Data == content
		},
		nonEmptyStringGen,
	))

	// Property: SSE events with event: prefix are correctly parsed
	properties.Property("sse events with event prefix are parsed", prop.ForAll(
		func(eventType string) bool {
			stream := "event: " + eventType + "\ndata: {}\n\n"
			reader := strings.NewReader(stream)

			parser := NewSSEParser(reader)
			events, err := parser.ParseAll()
			if err != nil {
				return false
			}

			return len(events) == 1 && events[0].Event == eventType
		},
		nonEmptyStringGen,
	))

	// Property: NewSSEValidator returns correct validator type for each provider
	properties.Property("NewSSEValidator returns correct validator type", prop.ForAll(
		func(providerType string) bool {
			validator := NewSSEValidator(providerType)

			switch providerType {
			case "anthropic":
				_, ok := validator.(*AnthropicSSEValidator)
				return ok
			case "openai":
				_, ok := validator.(*OpenAISSEValidator)
				return ok
			default:
				// Unknown providers default to OpenAI
				_, ok := validator.(*OpenAISSEValidator)
				return ok
			}
		},
		gen.OneConstOf("anthropic", "openai", "unknown", "custom"),
	))

	// Property: IsCompletionSignal correctly identifies Anthropic message_stop
	properties.Property("anthropic completion signal is detected", prop.ForAll(
		func(_ int) bool {
			validator := NewAnthropicSSEValidator()

			// Test with event type
			event1 := &SSEEvent{Event: "message_stop", Data: `{"type":"message_stop"}`}
			if !validator.IsCompletionSignal(event1) {
				return false
			}

			// Test with data type only
			event2 := &SSEEvent{Data: `{"type":"message_stop"}`}
			if !validator.IsCompletionSignal(event2) {
				return false
			}

			// Test non-completion event
			event3 := &SSEEvent{Event: "content_block_delta", Data: `{"type":"content_block_delta"}`}
			if validator.IsCompletionSignal(event3) {
				return false
			}

			return true
		},
		gen.Int(),
	))

	// Property: IsCompletionSignal correctly identifies OpenAI [DONE]
	properties.Property("openai completion signal is detected", prop.ForAll(
		func(_ int) bool {
			validator := NewOpenAISSEValidator()

			// Test [DONE]
			event1 := &SSEEvent{Data: "[DONE]"}
			if !validator.IsCompletionSignal(event1) {
				return false
			}

			// Test [DONE] with whitespace
			event2 := &SSEEvent{Data: " [DONE] "}
			if !validator.IsCompletionSignal(event2) {
				return false
			}

			// Test non-completion event
			event3 := &SSEEvent{Data: `{"id":"chatcmpl-123"}`}
			if validator.IsCompletionSignal(event3) {
				return false
			}

			return true
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}
