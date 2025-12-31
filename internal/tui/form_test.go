package tui

import (
	"strings"
	"testing"
)

// TestFormDataValidate tests the FormData.Validate method
// Requirements: 5.4, 6.4
func TestFormDataValidate(t *testing.T) {
	tests := []struct {
		name    string
		data    FormData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with API key",
			data: FormData{
				Alias:  "test-config",
				APIKey: "sk-test-key",
			},
			wantErr: false,
		},
		{
			name: "valid with auth token",
			data: FormData{
				Alias:     "test-config",
				AuthToken: "auth-token-123",
			},
			wantErr: false,
		},
		{
			name: "valid with both API key and auth token",
			data: FormData{
				Alias:     "test-config",
				APIKey:    "sk-test-key",
				AuthToken: "auth-token-123",
			},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			data: FormData{
				Alias:     "test-config",
				APIKey:    "sk-test-key",
				AuthToken: "auth-token-123",
				BaseURL:   "https://api.example.com",
				Model:     "claude-sonnet-4-20250514",
				Models:    "model1, model2",
			},
			wantErr: false,
		},
		{
			name: "empty alias",
			data: FormData{
				Alias:  "",
				APIKey: "sk-test-key",
			},
			wantErr: true,
			errMsg:  "alias 不能为空",
		},
		{
			name: "whitespace only alias",
			data: FormData{
				Alias:  "   ",
				APIKey: "sk-test-key",
			},
			wantErr: true,
			errMsg:  "alias 不能为空",
		},
		{
			name: "both API key and auth token empty",
			data: FormData{
				Alias: "test-config",
			},
			wantErr: true,
			errMsg:  "API key 和 auth token 不能同时为空",
		},
		{
			name: "both API key and auth token whitespace",
			data: FormData{
				Alias:     "test-config",
				APIKey:    "   ",
				AuthToken: "   ",
			},
			wantErr: true,
			errMsg:  "API key 和 auth token 不能同时为空",
		},
		{
			name: "invalid URL format",
			data: FormData{
				Alias:   "test-config",
				APIKey:  "sk-test-key",
				BaseURL: "not-a-valid-url",
			},
			wantErr: true,
			errMsg:  "无效的 URL 格式",
		},
		{
			name: "URL without scheme",
			data: FormData{
				Alias:   "test-config",
				APIKey:  "sk-test-key",
				BaseURL: "api.example.com",
			},
			wantErr: true,
			errMsg:  "无效的 URL 格式",
		},
		{
			name: "valid http URL",
			data: FormData{
				Alias:   "test-config",
				APIKey:  "sk-test-key",
				BaseURL: "http://api.example.com",
			},
			wantErr: false,
		},
		{
			name: "valid https URL",
			data: FormData{
				Alias:   "test-config",
				APIKey:  "sk-test-key",
				BaseURL: "https://api.example.com/v1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.data.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestFormDataParseModels tests the FormData.ParseModels method
func TestFormDataParseModels(t *testing.T) {
	tests := []struct {
		name   string
		models string
		want   []string
	}{
		{
			name:   "empty string",
			models: "",
			want:   []string{},
		},
		{
			name:   "whitespace only",
			models: "   ",
			want:   []string{},
		},
		{
			name:   "single model",
			models: "claude-sonnet-4-20250514",
			want:   []string{"claude-sonnet-4-20250514"},
		},
		{
			name:   "multiple models",
			models: "model1, model2, model3",
			want:   []string{"model1", "model2", "model3"},
		},
		{
			name:   "models with extra whitespace",
			models: "  model1  ,  model2  ,  model3  ",
			want:   []string{"model1", "model2", "model3"},
		},
		{
			name:   "models with empty entries",
			models: "model1,,model2,  ,model3",
			want:   []string{"model1", "model2", "model3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := FormData{Models: tt.models}
			got := data.ParseModels()

			if len(got) != len(tt.want) {
				t.Errorf("ParseModels() got %d models, want %d", len(got), len(tt.want))
				return
			}

			for i, model := range got {
				if model != tt.want[i] {
					t.Errorf("ParseModels()[%d] = %v, want %v", i, model, tt.want[i])
				}
			}
		})
	}
}

// TestFormInputs tests the FormInputs function
func TestFormInputs(t *testing.T) {
	inputs := FormInputs()

	// Check that we have the correct number of inputs
	if len(inputs) != FormFieldCount {
		t.Errorf("FormInputs() returned %d inputs, want %d", len(inputs), FormFieldCount)
	}

	// Check that the first input is focused
	if !inputs[FormFieldAlias].Focused() {
		t.Error("FormInputs() first input should be focused")
	}

	// Check that other inputs are not focused
	for i := 1; i < len(inputs); i++ {
		if inputs[i].Focused() {
			t.Errorf("FormInputs() input %d should not be focused", i)
		}
	}
}

// TestGetFormData tests the GetFormData function
func TestGetFormData(t *testing.T) {
	inputs := FormInputs()

	// Set values
	inputs[FormFieldAlias].SetValue("test-alias")
	inputs[FormFieldAPIKey].SetValue("test-api-key")
	inputs[FormFieldAuthToken].SetValue("test-auth-token")
	inputs[FormFieldBaseURL].SetValue("https://api.example.com")
	inputs[FormFieldModel].SetValue("claude-sonnet-4-20250514")
	inputs[FormFieldModels].SetValue("model1, model2")

	data := GetFormData(inputs)

	if data.Alias != "test-alias" {
		t.Errorf("GetFormData().Alias = %v, want %v", data.Alias, "test-alias")
	}
	if data.APIKey != "test-api-key" {
		t.Errorf("GetFormData().APIKey = %v, want %v", data.APIKey, "test-api-key")
	}
	if data.AuthToken != "test-auth-token" {
		t.Errorf("GetFormData().AuthToken = %v, want %v", data.AuthToken, "test-auth-token")
	}
	if data.BaseURL != "https://api.example.com" {
		t.Errorf("GetFormData().BaseURL = %v, want %v", data.BaseURL, "https://api.example.com")
	}
	if data.Model != "claude-sonnet-4-20250514" {
		t.Errorf("GetFormData().Model = %v, want %v", data.Model, "claude-sonnet-4-20250514")
	}
	if data.Models != "model1, model2" {
		t.Errorf("GetFormData().Models = %v, want %v", data.Models, "model1, model2")
	}
}

// TestSetFormData tests the SetFormData function
func TestSetFormData(t *testing.T) {
	inputs := FormInputs()

	data := FormData{
		Alias:     "test-alias",
		APIKey:    "test-api-key",
		AuthToken: "test-auth-token",
		BaseURL:   "https://api.example.com",
		Model:     "claude-sonnet-4-20250514",
		Models:    "model1, model2",
	}

	SetFormData(inputs, data)

	if inputs[FormFieldAlias].Value() != "test-alias" {
		t.Errorf("SetFormData() Alias = %v, want %v", inputs[FormFieldAlias].Value(), "test-alias")
	}
	if inputs[FormFieldAPIKey].Value() != "test-api-key" {
		t.Errorf("SetFormData() APIKey = %v, want %v", inputs[FormFieldAPIKey].Value(), "test-api-key")
	}
	if inputs[FormFieldAuthToken].Value() != "test-auth-token" {
		t.Errorf("SetFormData() AuthToken = %v, want %v", inputs[FormFieldAuthToken].Value(), "test-auth-token")
	}
	if inputs[FormFieldBaseURL].Value() != "https://api.example.com" {
		t.Errorf("SetFormData() BaseURL = %v, want %v", inputs[FormFieldBaseURL].Value(), "https://api.example.com")
	}
	if inputs[FormFieldModel].Value() != "claude-sonnet-4-20250514" {
		t.Errorf("SetFormData() Model = %v, want %v", inputs[FormFieldModel].Value(), "claude-sonnet-4-20250514")
	}
	if inputs[FormFieldModels].Value() != "model1, model2" {
		t.Errorf("SetFormData() Models = %v, want %v", inputs[FormFieldModels].Value(), "model1, model2")
	}
}

// TestNextFormField tests the NextFormField function
func TestNextFormField(t *testing.T) {
	inputs := FormInputs()

	// Start at first field
	currentFocus := 0

	// Move to next field
	nextFocus := NextFormField(inputs, currentFocus)
	if nextFocus != 1 {
		t.Errorf("NextFormField() = %d, want %d", nextFocus, 1)
	}
	if inputs[0].Focused() {
		t.Error("NextFormField() should blur previous field")
	}
	if !inputs[1].Focused() {
		t.Error("NextFormField() should focus next field")
	}

	// Move from last field should wrap to first
	currentFocus = FormFieldCount - 1
	inputs[currentFocus].Focus()
	nextFocus = NextFormField(inputs, currentFocus)
	if nextFocus != 0 {
		t.Errorf("NextFormField() from last = %d, want %d", nextFocus, 0)
	}
}

// TestPrevFormField tests the PrevFormField function
func TestPrevFormField(t *testing.T) {
	inputs := FormInputs()

	// Start at second field
	currentFocus := 1
	inputs[0].Blur()
	inputs[1].Focus()

	// Move to previous field
	prevFocus := PrevFormField(inputs, currentFocus)
	if prevFocus != 0 {
		t.Errorf("PrevFormField() = %d, want %d", prevFocus, 0)
	}
	if inputs[1].Focused() {
		t.Error("PrevFormField() should blur current field")
	}
	if !inputs[0].Focused() {
		t.Error("PrevFormField() should focus previous field")
	}

	// Move from first field should wrap to last
	currentFocus = 0
	prevFocus = PrevFormField(inputs, currentFocus)
	if prevFocus != FormFieldCount-1 {
		t.Errorf("PrevFormField() from first = %d, want %d", prevFocus, FormFieldCount-1)
	}
}

// TestFormLabels tests the FormLabels function
func TestFormLabels(t *testing.T) {
	labels := FormLabels()

	if len(labels) != FormFieldCount {
		t.Errorf("FormLabels() returned %d labels, want %d", len(labels), FormFieldCount)
	}

	expectedLabels := []string{
		"Alias:",
		"API Key:",
		"Auth Token:",
		"Base URL:",
		"Model:",
		"Models:",
	}

	for i, label := range labels {
		if label != expectedLabels[i] {
			t.Errorf("FormLabels()[%d] = %v, want %v", i, label, expectedLabels[i])
		}
	}
}

// TestFormHints tests the FormHints function
func TestFormHints(t *testing.T) {
	hints := FormHints()

	if len(hints) != FormFieldCount {
		t.Errorf("FormHints() returned %d hints, want %d", len(hints), FormFieldCount)
	}

	// Just check that hints are not empty
	for i, hint := range hints {
		if hint == "" {
			t.Errorf("FormHints()[%d] should not be empty", i)
		}
	}
}


// TestRenderForm tests the RenderForm function
// Requirements: 5.2, 6.2
func TestRenderForm(t *testing.T) {
	inputs := FormInputs()

	// Test add form rendering
	t.Run("add form", func(t *testing.T) {
		output := RenderForm(inputs, 0, "添加配置", "")

		// Check title is present
		if !strings.Contains(output, "添加配置") {
			t.Error("RenderForm() should contain title '添加配置'")
		}

		// Check help text is present
		if !strings.Contains(output, "Enter: 确认") {
			t.Error("RenderForm() should contain help text")
		}
		if !strings.Contains(output, "Esc: 取消") {
			t.Error("RenderForm() should contain cancel help text")
		}
	})

	// Test edit form rendering
	t.Run("edit form", func(t *testing.T) {
		output := RenderForm(inputs, 0, "编辑配置", "")

		// Check title is present
		if !strings.Contains(output, "编辑配置") {
			t.Error("RenderForm() should contain title '编辑配置'")
		}
	})

	// Test error message rendering
	t.Run("with error", func(t *testing.T) {
		output := RenderForm(inputs, 0, "添加配置", "测试错误消息")

		// Check error message is present
		if !strings.Contains(output, "测试错误消息") {
			t.Error("RenderForm() should contain error message")
		}
	})

	// Test focused field hint
	t.Run("focused field hint", func(t *testing.T) {
		hints := FormHints()
		output := RenderForm(inputs, 0, "添加配置", "")

		// Check that the hint for the focused field is present
		if !strings.Contains(output, hints[0]) {
			t.Error("RenderForm() should contain hint for focused field")
		}
	})
}
