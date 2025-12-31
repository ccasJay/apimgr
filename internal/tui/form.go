// Package tui provides a terminal user interface for apimgr
package tui

import (
	"errors"
	"strings"

	"apimgr/internal/utils"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// FormField represents the index of each form field
const (
	FormFieldAlias = iota
	FormFieldAPIKey
	FormFieldAuthToken
	FormFieldBaseURL
	FormFieldModel
	FormFieldModels
	FormFieldCount // Total number of fields
)

// FormData represents the data collected from the form
// Requirements: 5.3, 5.4, 6.3, 6.4
type FormData struct {
	Alias     string
	APIKey    string
	AuthToken string
	BaseURL   string
	Model     string
	Models    string // Comma-separated list of models
}

// Validate validates the form data
// Requirements: 5.4, 6.4
func (f *FormData) Validate() error {
	// Alias cannot be empty
	if strings.TrimSpace(f.Alias) == "" {
		return errors.New("alias 不能为空")
	}

	// At least one authentication method is required
	if strings.TrimSpace(f.APIKey) == "" && strings.TrimSpace(f.AuthToken) == "" {
		return errors.New("API key 和 auth token 不能同时为空")
	}

	// Validate URL format if provided
	if strings.TrimSpace(f.BaseURL) != "" && !utils.ValidateURL(f.BaseURL) {
		return errors.New("无效的 URL 格式")
	}

	return nil
}

// ParseModels parses the comma-separated models string into a slice
func (f *FormData) ParseModels() []string {
	if strings.TrimSpace(f.Models) == "" {
		return []string{}
	}

	parts := strings.Split(f.Models, ",")
	models := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			models = append(models, trimmed)
		}
	}
	return models
}

// Form styles
var (
	formLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(14)

	formInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	formFocusedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	formErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	formHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
)

// FormInputs creates and initializes form input fields
// Requirements: 5.2, 6.2
func FormInputs() []textinput.Model {
	inputs := make([]textinput.Model, FormFieldCount)

	// Alias input
	inputs[FormFieldAlias] = textinput.New()
	inputs[FormFieldAlias].Placeholder = "配置别名"
	inputs[FormFieldAlias].CharLimit = 64
	inputs[FormFieldAlias].Width = 40
	inputs[FormFieldAlias].Prompt = ""

	// API Key input
	inputs[FormFieldAPIKey] = textinput.New()
	inputs[FormFieldAPIKey].Placeholder = "API 密钥"
	inputs[FormFieldAPIKey].CharLimit = 256
	inputs[FormFieldAPIKey].Width = 40
	inputs[FormFieldAPIKey].EchoMode = textinput.EchoPassword
	inputs[FormFieldAPIKey].EchoCharacter = '•'
	inputs[FormFieldAPIKey].Prompt = ""

	// Auth Token input
	inputs[FormFieldAuthToken] = textinput.New()
	inputs[FormFieldAuthToken].Placeholder = "认证令牌"
	inputs[FormFieldAuthToken].CharLimit = 256
	inputs[FormFieldAuthToken].Width = 40
	inputs[FormFieldAuthToken].EchoMode = textinput.EchoPassword
	inputs[FormFieldAuthToken].EchoCharacter = '•'
	inputs[FormFieldAuthToken].Prompt = ""

	// Base URL input
	inputs[FormFieldBaseURL] = textinput.New()
	inputs[FormFieldBaseURL].Placeholder = "https://api.example.com"
	inputs[FormFieldBaseURL].CharLimit = 256
	inputs[FormFieldBaseURL].Width = 40
	inputs[FormFieldBaseURL].Prompt = ""

	// Model input
	inputs[FormFieldModel] = textinput.New()
	inputs[FormFieldModel].Placeholder = "claude-sonnet-4-20250514"
	inputs[FormFieldModel].CharLimit = 128
	inputs[FormFieldModel].Width = 40
	inputs[FormFieldModel].Prompt = ""

	// Models list input
	inputs[FormFieldModels] = textinput.New()
	inputs[FormFieldModels].Placeholder = "model1, model2, model3"
	inputs[FormFieldModels].CharLimit = 512
	inputs[FormFieldModels].Width = 40
	inputs[FormFieldModels].Prompt = ""

	// Focus the first input
	inputs[FormFieldAlias].Focus()

	return inputs
}

// GetFormData extracts FormData from form inputs
func GetFormData(inputs []textinput.Model) FormData {
	return FormData{
		Alias:     inputs[FormFieldAlias].Value(),
		APIKey:    inputs[FormFieldAPIKey].Value(),
		AuthToken: inputs[FormFieldAuthToken].Value(),
		BaseURL:   inputs[FormFieldBaseURL].Value(),
		Model:     inputs[FormFieldModel].Value(),
		Models:    inputs[FormFieldModels].Value(),
	}
}

// SetFormData populates form inputs with existing data
// Requirements: 6.2
func SetFormData(inputs []textinput.Model, data FormData) {
	inputs[FormFieldAlias].SetValue(data.Alias)
	inputs[FormFieldAPIKey].SetValue(data.APIKey)
	inputs[FormFieldAuthToken].SetValue(data.AuthToken)
	inputs[FormFieldBaseURL].SetValue(data.BaseURL)
	inputs[FormFieldModel].SetValue(data.Model)
	inputs[FormFieldModels].SetValue(data.Models)
}

// FormLabels returns the labels for each form field
func FormLabels() []string {
	return []string{
		"Alias:",
		"API Key:",
		"Auth Token:",
		"Base URL:",
		"Model:",
		"Models:",
	}
}

// FormHints returns the hint text for each form field
func FormHints() []string {
	return []string{
		"配置的唯一标识符",
		"API 密钥 (与 Auth Token 二选一)",
		"认证令牌 (与 API Key 二选一)",
		"API 基础 URL (可选)",
		"当前使用的模型 (可选)",
		"支持的模型列表，逗号分隔 (可选)",
	}
}

// RenderForm renders the form view with inputs
// Requirements: 5.2, 6.2
func RenderForm(inputs []textinput.Model, focusIndex int, title string, errorMsg string) string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", 50)))
	b.WriteString("\n\n")

	labels := FormLabels()
	hints := FormHints()

	// Render each input field
	for i, input := range inputs {
		// Label
		label := labels[i]
		if i == focusIndex {
			b.WriteString(formFocusedStyle.Render(label))
		} else {
			b.WriteString(formLabelStyle.Render(label))
		}
		b.WriteString(" ")

		// Input field
		b.WriteString(input.View())
		b.WriteString("\n")

		// Hint (only show for focused field)
		if i == focusIndex {
			b.WriteString(formLabelStyle.Render(""))
			b.WriteString(" ")
			b.WriteString(formHintStyle.Render(hints[i]))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Error message
	if errorMsg != "" {
		b.WriteString("\n")
		b.WriteString(formErrorStyle.Render("✗ " + errorMsg))
		b.WriteString("\n")
	}

	// Footer with available actions
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", 50)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Tab/↓: 下一项 │ Shift+Tab/↑: 上一项 │ Enter: 确认 │ Esc: 取消"))

	return b.String()
}

// NextFormField moves focus to the next form field
func NextFormField(inputs []textinput.Model, currentFocus int) int {
	inputs[currentFocus].Blur()
	nextFocus := (currentFocus + 1) % len(inputs)
	inputs[nextFocus].Focus()
	return nextFocus
}

// PrevFormField moves focus to the previous form field
func PrevFormField(inputs []textinput.Model, currentFocus int) int {
	inputs[currentFocus].Blur()
	prevFocus := currentFocus - 1
	if prevFocus < 0 {
		prevFocus = len(inputs) - 1
	}
	inputs[prevFocus].Focus()
	return prevFocus
}
