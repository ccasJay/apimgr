// Package compatibility provides API compatibility testing functionality
// for validating that API configurations work correctly with Claude Code.
package compatibility

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SSEEvent represents a single Server-Sent Event
type SSEEvent struct {
	Event string // The event type (from "event:" line)
	Data  string // The data payload (from "data:" line)
	ID    string // Optional event ID (from "id:" line)
	Retry int    // Optional retry interval (from "retry:" line)
}

// SSEValidationResult represents the result of SSE stream validation
type SSEValidationResult struct {
	Valid              bool     `json:"valid"`
	EventCount         int      `json:"eventCount"`
	HasCompletionSignal bool    `json:"hasCompletionSignal"`
	CompletionType     string   `json:"completionType,omitempty"` // "done" for OpenAI, "message_stop" for Anthropic
	MalformedLines     []string `json:"malformedLines,omitempty"`
	Errors             []string `json:"errors,omitempty"`
}

// SSEParser parses Server-Sent Events from a stream
type SSEParser struct {
	reader *bufio.Reader
}

// NewSSEParser creates a new SSE parser from an io.Reader
func NewSSEParser(r io.Reader) *SSEParser {
	return &SSEParser{
		reader: bufio.NewReader(r),
	}
}

// ParseEvent reads and parses the next SSE event from the stream
// Returns nil when the stream ends
func (p *SSEParser) ParseEvent() (*SSEEvent, error) {
	event := &SSEEvent{}
	hasData := false

	for {
		line, err := p.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if hasData {
					return event, nil
				}
				return nil, io.EOF
			}
			return nil, fmt.Errorf("error reading SSE stream: %w", err)
		}

		// Trim the newline
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		// Empty line signals end of event
		if line == "" {
			if hasData {
				return event, nil
			}
			continue
		}

		// Parse the line
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimPrefix(data, " ") // Optional space after colon
			if event.Data != "" {
				event.Data += "\n" + data
			} else {
				event.Data = data
			}
			hasData = true
		} else if strings.HasPrefix(line, "event:") {
			event.Event = strings.TrimPrefix(line, "event:")
			event.Event = strings.TrimPrefix(event.Event, " ")
			hasData = true
		} else if strings.HasPrefix(line, "id:") {
			event.ID = strings.TrimPrefix(line, "id:")
			event.ID = strings.TrimPrefix(event.ID, " ")
		} else if strings.HasPrefix(line, "retry:") {
			// Parse retry as integer (ignore errors)
			retryStr := strings.TrimPrefix(line, "retry:")
			retryStr = strings.TrimPrefix(retryStr, " ")
			fmt.Sscanf(retryStr, "%d", &event.Retry)
		} else if strings.HasPrefix(line, ":") {
			// Comment line, ignore
			continue
		}
		// Lines without a recognized prefix are ignored per SSE spec
	}
}

// ParseAll reads all events from the stream
func (p *SSEParser) ParseAll() ([]*SSEEvent, error) {
	var events []*SSEEvent

	for {
		event, err := p.ParseEvent()
		if err != nil {
			if err == io.EOF {
				break
			}
			return events, err
		}
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}


// SSEValidator defines the interface for validating SSE streams
type SSEValidator interface {
	// ValidateStream validates an SSE stream and returns the validation result
	ValidateStream(reader io.Reader) (*SSEValidationResult, error)
	// IsCompletionSignal checks if an event represents a completion signal
	IsCompletionSignal(event *SSEEvent) bool
	// IsValidEventFormat checks if an event has valid format for the provider
	IsValidEventFormat(event *SSEEvent) bool
}

// AnthropicSSEValidator validates SSE streams from Anthropic API
type AnthropicSSEValidator struct{}

// NewAnthropicSSEValidator creates a new Anthropic SSE validator
func NewAnthropicSSEValidator() *AnthropicSSEValidator {
	return &AnthropicSSEValidator{}
}

// IsCompletionSignal checks if an event is the Anthropic completion signal (message_stop)
func (v *AnthropicSSEValidator) IsCompletionSignal(event *SSEEvent) bool {
	if event == nil {
		return false
	}

	// Check event type
	if event.Event == "message_stop" {
		return true
	}

	// Also check data for message_stop type
	if event.Data != "" {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(event.Data), &data); err == nil {
			if dataType, ok := data["type"].(string); ok && dataType == "message_stop" {
				return true
			}
		}
	}

	return false
}

// IsValidEventFormat checks if an event has valid Anthropic SSE format
func (v *AnthropicSSEValidator) IsValidEventFormat(event *SSEEvent) bool {
	if event == nil {
		return false
	}

	// Anthropic events should have either event type or data
	if event.Event == "" && event.Data == "" {
		return false
	}

	// If there's data, it should be valid JSON
	if event.Data != "" {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
			return false
		}
	}

	return true
}

// ValidateStream validates an Anthropic SSE stream
func (v *AnthropicSSEValidator) ValidateStream(reader io.Reader) (*SSEValidationResult, error) {
	result := &SSEValidationResult{
		Valid:              true,
		MalformedLines:     []string{},
		Errors:             []string{},
	}

	parser := NewSSEParser(reader)
	events, err := parser.ParseAll()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", err))
		return result, nil
	}

	result.EventCount = len(events)

	for _, event := range events {
		if !v.IsValidEventFormat(event) {
			result.MalformedLines = append(result.MalformedLines, 
				fmt.Sprintf("event=%s, data=%s", event.Event, truncateString(event.Data, 50)))
		}

		if v.IsCompletionSignal(event) {
			result.HasCompletionSignal = true
			result.CompletionType = "message_stop"
		}
	}

	// Stream is valid if we have events and a completion signal
	result.Valid = result.EventCount > 0 && result.HasCompletionSignal && len(result.MalformedLines) == 0

	return result, nil
}

// OpenAISSEValidator validates SSE streams from OpenAI API
type OpenAISSEValidator struct{}

// NewOpenAISSEValidator creates a new OpenAI SSE validator
func NewOpenAISSEValidator() *OpenAISSEValidator {
	return &OpenAISSEValidator{}
}

// IsCompletionSignal checks if an event is the OpenAI completion signal ([DONE])
func (v *OpenAISSEValidator) IsCompletionSignal(event *SSEEvent) bool {
	if event == nil {
		return false
	}

	// OpenAI uses [DONE] as the completion signal
	return strings.TrimSpace(event.Data) == "[DONE]"
}

// IsValidEventFormat checks if an event has valid OpenAI SSE format
func (v *OpenAISSEValidator) IsValidEventFormat(event *SSEEvent) bool {
	if event == nil {
		return false
	}

	// OpenAI events should have data
	if event.Data == "" {
		return false
	}

	// [DONE] is a special valid format
	if strings.TrimSpace(event.Data) == "[DONE]" {
		return true
	}

	// Otherwise, data should be valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
		return false
	}

	return true
}

// ValidateStream validates an OpenAI SSE stream
func (v *OpenAISSEValidator) ValidateStream(reader io.Reader) (*SSEValidationResult, error) {
	result := &SSEValidationResult{
		Valid:              true,
		MalformedLines:     []string{},
		Errors:             []string{},
	}

	parser := NewSSEParser(reader)
	events, err := parser.ParseAll()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", err))
		return result, nil
	}

	result.EventCount = len(events)

	for _, event := range events {
		if !v.IsValidEventFormat(event) {
			result.MalformedLines = append(result.MalformedLines,
				fmt.Sprintf("data=%s", truncateString(event.Data, 50)))
		}

		if v.IsCompletionSignal(event) {
			result.HasCompletionSignal = true
			result.CompletionType = "done"
		}
	}

	// Stream is valid if we have events and a completion signal
	result.Valid = result.EventCount > 0 && result.HasCompletionSignal && len(result.MalformedLines) == 0

	return result, nil
}

// NewSSEValidator creates a new SSE validator based on the provider type
func NewSSEValidator(providerType string) SSEValidator {
	switch providerType {
	case "anthropic":
		return NewAnthropicSSEValidator()
	case "openai":
		return NewOpenAISSEValidator()
	default:
		// Default to OpenAI-compatible format
		return NewOpenAISSEValidator()
	}
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
