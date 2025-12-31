package tui

import (
	"fmt"
	"strings"
	"testing"

	"apimgr/config"

	tea "github.com/charmbracelet/bubbletea"
)

// TestInitEditForm tests the initEditForm method
// Requirements: 6.1, 6.2
func TestInitEditForm(t *testing.T) {
	tests := []struct {
		name           string
		configs        []config.APIConfig
		cursor         int
		expectViewEdit bool
		expectFormData FormData
	}{
		{
			name: "edit first config",
			configs: []config.APIConfig{
				{
					Alias:     "test-config",
					APIKey:    "sk-test-key",
					AuthToken: "auth-token",
					BaseURL:   "https://api.example.com",
					Model:     "claude-sonnet-4-20250514",
					Models:    []string{"model1", "model2"},
				},
			},
			cursor:         0,
			expectViewEdit: true,
			expectFormData: FormData{
				Alias:     "test-config",
				APIKey:    "sk-test-key",
				AuthToken: "auth-token",
				BaseURL:   "https://api.example.com",
				Model:     "claude-sonnet-4-20250514",
				Models:    "model1, model2",
			},
		},
		{
			name: "edit second config",
			configs: []config.APIConfig{
				{Alias: "config1", APIKey: "key1"},
				{Alias: "config2", APIKey: "key2", BaseURL: "https://api2.example.com"},
			},
			cursor:         1,
			expectViewEdit: true,
			expectFormData: FormData{
				Alias:   "config2",
				APIKey:  "key2",
				BaseURL: "https://api2.example.com",
			},
		},
		{
			name:           "cursor out of bounds negative",
			configs:        []config.APIConfig{{Alias: "test", APIKey: "key"}},
			cursor:         -1,
			expectViewEdit: false,
		},
		{
			name:           "cursor out of bounds positive",
			configs:        []config.APIConfig{{Alias: "test", APIKey: "key"}},
			cursor:         5,
			expectViewEdit: false,
		},
		{
			name:           "empty configs",
			configs:        []config.APIConfig{},
			cursor:         0,
			expectViewEdit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs:   tt.configs,
				cursor:    tt.cursor,
				viewState: ViewMain,
			}

			m.initEditForm()

			if tt.expectViewEdit {
				if m.viewState != ViewEdit {
					t.Errorf("initEditForm() viewState = %v, want ViewEdit", m.viewState)
				}
				if len(m.formInputs) != FormFieldCount {
					t.Errorf("initEditForm() formInputs count = %d, want %d", len(m.formInputs), FormFieldCount)
				}
				if m.formFocus != 0 {
					t.Errorf("initEditForm() formFocus = %d, want 0", m.formFocus)
				}
				if m.errorMsg != "" {
					t.Errorf("initEditForm() errorMsg = %q, want empty", m.errorMsg)
				}

				// Verify form data is pre-filled correctly
				formData := GetFormData(m.formInputs)
				if formData.Alias != tt.expectFormData.Alias {
					t.Errorf("initEditForm() Alias = %q, want %q", formData.Alias, tt.expectFormData.Alias)
				}
				if formData.APIKey != tt.expectFormData.APIKey {
					t.Errorf("initEditForm() APIKey = %q, want %q", formData.APIKey, tt.expectFormData.APIKey)
				}
				if formData.AuthToken != tt.expectFormData.AuthToken {
					t.Errorf("initEditForm() AuthToken = %q, want %q", formData.AuthToken, tt.expectFormData.AuthToken)
				}
				if formData.BaseURL != tt.expectFormData.BaseURL {
					t.Errorf("initEditForm() BaseURL = %q, want %q", formData.BaseURL, tt.expectFormData.BaseURL)
				}
				if formData.Model != tt.expectFormData.Model {
					t.Errorf("initEditForm() Model = %q, want %q", formData.Model, tt.expectFormData.Model)
				}
				if formData.Models != tt.expectFormData.Models {
					t.Errorf("initEditForm() Models = %q, want %q", formData.Models, tt.expectFormData.Models)
				}
			} else {
				if m.viewState == ViewEdit {
					t.Error("initEditForm() should not change to ViewEdit for invalid cursor")
				}
			}
		})
	}
}

// TestHandleMainViewKeysEdit tests the 'e' key handling in main view
// Requirements: 6.1
func TestHandleMainViewKeysEdit(t *testing.T) {
	tests := []struct {
		name           string
		configs        []config.APIConfig
		cursor         int
		expectViewEdit bool
	}{
		{
			name: "press e with valid cursor",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "sk-test-key"},
			},
			cursor:         0,
			expectViewEdit: true,
		},
		{
			name:           "press e with empty configs",
			configs:        []config.APIConfig{},
			cursor:         0,
			expectViewEdit: false,
		},
		{
			name: "press e with cursor out of bounds",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "sk-test-key"},
			},
			cursor:         5,
			expectViewEdit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs:   tt.configs,
				cursor:    tt.cursor,
				viewState: ViewMain,
			}

			// Simulate pressing 'e' key
			newModel, _ := m.handleMainViewKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
			updatedModel := newModel.(Model)

			if tt.expectViewEdit {
				if updatedModel.viewState != ViewEdit {
					t.Errorf("handleMainViewKeys('e') viewState = %v, want ViewEdit", updatedModel.viewState)
				}
			} else {
				if updatedModel.viewState == ViewEdit {
					t.Error("handleMainViewKeys('e') should not change to ViewEdit for invalid state")
				}
			}
		})
	}
}

// TestHandleFormViewKeysEsc tests the Esc key handling in form view
// Requirements: 6.5
func TestHandleFormViewKeysEsc(t *testing.T) {
	m := Model{
		viewState:  ViewEdit,
		formInputs: FormInputs(),
		formFocus:  2,
		errorMsg:   "some error",
	}

	// Simulate pressing Esc key
	newModel, _ := m.handleFormViewKeys(tea.KeyMsg{Type: tea.KeyEsc})
	updatedModel := newModel.(Model)

	if updatedModel.viewState != ViewMain {
		t.Errorf("handleFormViewKeys(Esc) viewState = %v, want ViewMain", updatedModel.viewState)
	}
	if updatedModel.errorMsg != "" {
		t.Errorf("handleFormViewKeys(Esc) errorMsg = %q, want empty", updatedModel.errorMsg)
	}
	if len(updatedModel.formInputs) != 0 {
		t.Errorf("handleFormViewKeys(Esc) formInputs should be cleared")
	}
	if updatedModel.formFocus != 0 {
		t.Errorf("handleFormViewKeys(Esc) formFocus = %d, want 0", updatedModel.formFocus)
	}
}

// TestRenderFormViewFullEdit tests the RenderFormViewFull method for edit mode
// Requirements: 6.2
func TestRenderFormViewFullEdit(t *testing.T) {
	m := Model{
		viewState:  ViewEdit,
		formInputs: FormInputs(),
		formFocus:  0,
	}

	output := m.RenderFormViewFull()

	// Check that the title is "编辑配置"
	if !strings.Contains(output, "编辑配置") {
		t.Error("RenderFormViewFull() should contain '编辑配置' for ViewEdit")
	}
}

// TestConfigUpdatedMsgHandling tests the ConfigUpdatedMsg handling in Update
// Requirements: 6.3
func TestConfigUpdatedMsgHandling(t *testing.T) {
	tests := []struct {
		name          string
		msg           ConfigUpdatedMsg
		expectMessage string
		expectError   bool
	}{
		{
			name: "successful update",
			msg: ConfigUpdatedMsg{
				Alias: "test-config",
				Err:   nil,
			},
			expectMessage: "配置已更新: test-config",
			expectError:   false,
		},
		{
			name: "failed update",
			msg: ConfigUpdatedMsg{
				Alias: "test-config",
				Err:   &testError{msg: "update failed"},
			},
			expectMessage: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				viewState:  ViewEdit,
				formInputs: FormInputs(),
			}

			newModel, _ := m.Update(tt.msg)
			updatedModel := newModel.(Model)

			if tt.expectError {
				if updatedModel.errorMsg == "" {
					t.Error("Update(ConfigUpdatedMsg) should set errorMsg on error")
				}
			} else {
				if updatedModel.message != tt.expectMessage {
					t.Errorf("Update(ConfigUpdatedMsg) message = %q, want %q", updatedModel.message, tt.expectMessage)
				}
				if updatedModel.viewState != ViewMain {
					t.Errorf("Update(ConfigUpdatedMsg) viewState = %v, want ViewMain", updatedModel.viewState)
				}
				if len(updatedModel.formInputs) != 0 {
					t.Error("Update(ConfigUpdatedMsg) should clear formInputs on success")
				}
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}


// TestHandleDetailViewKeysEdit tests the 'e' key handling in detail view
// Requirements: 6.1
func TestHandleDetailViewKeysEdit(t *testing.T) {
	tests := []struct {
		name           string
		configs        []config.APIConfig
		selected       int
		expectViewEdit bool
	}{
		{
			name: "press e with valid selected",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "sk-test-key"},
			},
			selected:       0,
			expectViewEdit: true,
		},
		{
			name:           "press e with invalid selected negative",
			configs:        []config.APIConfig{{Alias: "test", APIKey: "key"}},
			selected:       -1,
			expectViewEdit: false,
		},
		{
			name:           "press e with invalid selected out of bounds",
			configs:        []config.APIConfig{{Alias: "test", APIKey: "key"}},
			selected:       5,
			expectViewEdit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs:   tt.configs,
				selected:  tt.selected,
				viewState: ViewDetail,
			}

			// Simulate pressing 'e' key
			newModel, _ := m.handleDetailViewKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
			updatedModel := newModel.(Model)

			if tt.expectViewEdit {
				if updatedModel.viewState != ViewEdit {
					t.Errorf("handleDetailViewKeys('e') viewState = %v, want ViewEdit", updatedModel.viewState)
				}
				// Verify cursor is set to selected
				if updatedModel.cursor != tt.selected {
					t.Errorf("handleDetailViewKeys('e') cursor = %d, want %d", updatedModel.cursor, tt.selected)
				}
			} else {
				if updatedModel.viewState == ViewEdit {
					t.Error("handleDetailViewKeys('e') should not change to ViewEdit for invalid state")
				}
			}
		})
	}
}


// TestInitModelSelect tests the initModelSelect method
// Requirements: 12.1, 12.2
func TestInitModelSelect(t *testing.T) {
	tests := []struct {
		name              string
		cfg               config.APIConfig
		expectModelCursor int
		expectModelList   []string
	}{
		{
			name: "init with active model in list",
			cfg: config.APIConfig{
				Alias:  "test-config",
				Model:  "model2",
				Models: []string{"model1", "model2", "model3"},
			},
			expectModelCursor: 1, // model2 is at index 1
			expectModelList:   []string{"model1", "model2", "model3"},
		},
		{
			name: "init with active model at start",
			cfg: config.APIConfig{
				Alias:  "test-config",
				Model:  "model1",
				Models: []string{"model1", "model2"},
			},
			expectModelCursor: 0,
			expectModelList:   []string{"model1", "model2"},
		},
		{
			name: "init with active model at end",
			cfg: config.APIConfig{
				Alias:  "test-config",
				Model:  "model3",
				Models: []string{"model1", "model2", "model3"},
			},
			expectModelCursor: 2,
			expectModelList:   []string{"model1", "model2", "model3"},
		},
		{
			name: "init with active model not in list",
			cfg: config.APIConfig{
				Alias:  "test-config",
				Model:  "unknown-model",
				Models: []string{"model1", "model2"},
			},
			expectModelCursor: 0, // defaults to 0 when not found
			expectModelList:   []string{"model1", "model2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				viewState: ViewMain,
			}

			m.initModelSelect(tt.cfg)

			if m.viewState != ViewModelSelect {
				t.Errorf("initModelSelect() viewState = %v, want ViewModelSelect", m.viewState)
			}
			if m.modelCursor != tt.expectModelCursor {
				t.Errorf("initModelSelect() modelCursor = %d, want %d", m.modelCursor, tt.expectModelCursor)
			}
			if len(m.modelList) != len(tt.expectModelList) {
				t.Errorf("initModelSelect() modelList length = %d, want %d", len(m.modelList), len(tt.expectModelList))
			}
			for i, model := range m.modelList {
				if model != tt.expectModelList[i] {
					t.Errorf("initModelSelect() modelList[%d] = %q, want %q", i, model, tt.expectModelList[i])
				}
			}
			if m.message != "" {
				t.Errorf("initModelSelect() message = %q, want empty", m.message)
			}
			if m.errorMsg != "" {
				t.Errorf("initModelSelect() errorMsg = %q, want empty", m.errorMsg)
			}
		})
	}
}

// TestHandleModelSelectViewKeys tests keyboard handling in model selection view
// Requirements: 12.1, 12.2, 12.3
func TestHandleModelSelectViewKeys(t *testing.T) {
	tests := []struct {
		name              string
		modelList         []string
		modelCursor       int
		key               string
		expectModelCursor int
		expectViewState   ViewState
	}{
		{
			name:              "move down with j",
			modelList:         []string{"model1", "model2", "model3"},
			modelCursor:       0,
			key:               "j",
			expectModelCursor: 1,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "move down with down arrow",
			modelList:         []string{"model1", "model2", "model3"},
			modelCursor:       0,
			key:               "down",
			expectModelCursor: 1,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "move up with k",
			modelList:         []string{"model1", "model2", "model3"},
			modelCursor:       2,
			key:               "k",
			expectModelCursor: 1,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "move up with up arrow",
			modelList:         []string{"model1", "model2", "model3"},
			modelCursor:       2,
			key:               "up",
			expectModelCursor: 1,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "cannot move down past end",
			modelList:         []string{"model1", "model2"},
			modelCursor:       1,
			key:               "j",
			expectModelCursor: 1,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "cannot move up past start",
			modelList:         []string{"model1", "model2"},
			modelCursor:       0,
			key:               "k",
			expectModelCursor: 0,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "jump to top with g",
			modelList:         []string{"model1", "model2", "model3"},
			modelCursor:       2,
			key:               "g",
			expectModelCursor: 0,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "jump to bottom with G",
			modelList:         []string{"model1", "model2", "model3"},
			modelCursor:       0,
			key:               "G",
			expectModelCursor: 2,
			expectViewState:   ViewModelSelect,
		},
		{
			name:              "cancel with esc",
			modelList:         []string{"model1", "model2"},
			modelCursor:       1,
			key:               "esc",
			expectModelCursor: 0, // reset
			expectViewState:   ViewMain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				viewState:   ViewModelSelect,
				modelList:   tt.modelList,
				modelCursor: tt.modelCursor,
			}

			var keyMsg tea.KeyMsg
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else if tt.key == "down" {
				keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			} else if tt.key == "up" {
				keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			newModel, _ := m.handleModelSelectViewKeys(keyMsg)
			updatedModel := newModel.(Model)

			if updatedModel.modelCursor != tt.expectModelCursor {
				t.Errorf("handleModelSelectViewKeys(%q) modelCursor = %d, want %d", tt.key, updatedModel.modelCursor, tt.expectModelCursor)
			}
			if updatedModel.viewState != tt.expectViewState {
				t.Errorf("handleModelSelectViewKeys(%q) viewState = %v, want %v", tt.key, updatedModel.viewState, tt.expectViewState)
			}
		})
	}
}

// TestHandleMainViewKeysModel tests the 'm' key handling in main view
// Requirements: 12.1, 12.4
func TestHandleMainViewKeysModel(t *testing.T) {
	tests := []struct {
		name              string
		configs           []config.APIConfig
		cursor            int
		expectViewState   ViewState
		expectErrorMsg    string
	}{
		{
			name: "press m with multiple models",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "key", Models: []string{"model1", "model2"}},
			},
			cursor:          0,
			expectViewState: ViewModelSelect,
			expectErrorMsg:  "",
		},
		{
			name: "press m with single model",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "key", Models: []string{"model1"}},
			},
			cursor:          0,
			expectViewState: ViewMain,
			expectErrorMsg:  "此配置没有定义多个模型可供切换",
		},
		{
			name: "press m with no models",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "key", Models: []string{}},
			},
			cursor:          0,
			expectViewState: ViewMain,
			expectErrorMsg:  "此配置没有定义多个模型可供切换",
		},
		{
			name:            "press m with empty configs",
			configs:         []config.APIConfig{},
			cursor:          0,
			expectViewState: ViewMain,
			expectErrorMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs:   tt.configs,
				cursor:    tt.cursor,
				viewState: ViewMain,
			}

			newModel, _ := m.handleMainViewKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
			updatedModel := newModel.(Model)

			if updatedModel.viewState != tt.expectViewState {
				t.Errorf("handleMainViewKeys('m') viewState = %v, want %v", updatedModel.viewState, tt.expectViewState)
			}
			if updatedModel.errorMsg != tt.expectErrorMsg {
				t.Errorf("handleMainViewKeys('m') errorMsg = %q, want %q", updatedModel.errorMsg, tt.expectErrorMsg)
			}
		})
	}
}

// TestModelSwitchedMsgHandling tests the ModelSwitchedMsg handling in Update
// Requirements: 12.3
func TestModelSwitchedMsgHandling(t *testing.T) {
	tests := []struct {
		name          string
		msg           ModelSwitchedMsg
		expectMessage string
		expectError   bool
	}{
		{
			name: "successful switch",
			msg: ModelSwitchedMsg{
				Alias: "test-config",
				Model: "claude-sonnet-4-20250514",
				Err:   nil,
			},
			expectMessage: "模型已切换到: claude-sonnet-4-20250514",
			expectError:   false,
		},
		{
			name: "failed switch",
			msg: ModelSwitchedMsg{
				Alias: "test-config",
				Model: "invalid-model",
				Err:   &testError{msg: "model not found"},
			},
			expectMessage: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				viewState: ViewModelSelect,
			}

			newModel, _ := m.Update(tt.msg)
			updatedModel := newModel.(Model)

			if tt.expectError {
				if updatedModel.errorMsg == "" {
					t.Error("Update(ModelSwitchedMsg) should set errorMsg on error")
				}
			} else {
				if updatedModel.message != tt.expectMessage {
					t.Errorf("Update(ModelSwitchedMsg) message = %q, want %q", updatedModel.message, tt.expectMessage)
				}
				if updatedModel.viewState != ViewMain {
					t.Errorf("Update(ModelSwitchedMsg) viewState = %v, want ViewMain", updatedModel.viewState)
				}
			}
		})
	}
}

// TestRenderModelSelectView tests the RenderModelSelectView method
// Requirements: 12.1, 12.2
func TestRenderModelSelectView(t *testing.T) {
	m := Model{
		configs: []config.APIConfig{
			{Alias: "test-config", Model: "model2", Models: []string{"model1", "model2", "model3"}},
		},
		cursor:            0,
		viewState:         ViewModelSelect,
		modelList:         []string{"model1", "model2", "model3"},
		modelCursor:       1,
		height:            30, // Set sufficient height to show all models
		width:             80,
		modelScrollOffset: 0,
	}

	output := m.RenderModelSelectView()

	// Check that the title is present
	if !strings.Contains(output, "选择模型") {
		t.Error("RenderModelSelectView() should contain '选择模型'")
	}

	// Check that config info is shown
	if !strings.Contains(output, "test-config") {
		t.Error("RenderModelSelectView() should contain config alias")
	}

	// Check that models are listed
	if !strings.Contains(output, "model1") {
		t.Error("RenderModelSelectView() should contain 'model1'")
	}
	if !strings.Contains(output, "model2") {
		t.Error("RenderModelSelectView() should contain 'model2'")
	}
	if !strings.Contains(output, "model3") {
		t.Error("RenderModelSelectView() should contain 'model3'")
	}

	// Check that help text is present
	if !strings.Contains(output, "Enter") {
		t.Error("RenderModelSelectView() should contain help text with 'Enter'")
	}
	if !strings.Contains(output, "Esc") {
		t.Error("RenderModelSelectView() should contain help text with 'Esc'")
	}
}


// TestHandleMainViewKeysPing tests the 'p' key handling in main view
// Requirements: 8.1
func TestHandleMainViewKeysPing(t *testing.T) {
	tests := []struct {
		name            string
		configs         []config.APIConfig
		cursor          int
		expectViewState ViewState
		expectTesting   bool
	}{
		{
			name: "press p with valid cursor",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "sk-test-key", BaseURL: "https://api.example.com"},
			},
			cursor:          0,
			expectViewState: ViewPingTesting,
			expectTesting:   true,
		},
		{
			name:            "press p with empty configs",
			configs:         []config.APIConfig{},
			cursor:          0,
			expectViewState: ViewMain,
			expectTesting:   false,
		},
		{
			name: "press p with cursor out of bounds",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "sk-test-key"},
			},
			cursor:          5,
			expectViewState: ViewMain,
			expectTesting:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs:   tt.configs,
				cursor:    tt.cursor,
				viewState: ViewMain,
			}

			// Simulate pressing 'p' key
			newModel, cmd := m.handleMainViewKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
			updatedModel := newModel.(Model)

			if updatedModel.viewState != tt.expectViewState {
				t.Errorf("handleMainViewKeys('p') viewState = %v, want %v", updatedModel.viewState, tt.expectViewState)
			}
			if updatedModel.testing != tt.expectTesting {
				t.Errorf("handleMainViewKeys('p') testing = %v, want %v", updatedModel.testing, tt.expectTesting)
			}
			if tt.expectTesting && cmd == nil {
				t.Error("handleMainViewKeys('p') should return a command when testing")
			}
		})
	}
}

// TestHandleDetailViewKeysPing tests the 'p' key handling in detail view
// Requirements: 8.1
func TestHandleDetailViewKeysPing(t *testing.T) {
	tests := []struct {
		name            string
		configs         []config.APIConfig
		selected        int
		expectViewState ViewState
		expectTesting   bool
	}{
		{
			name: "press p with valid selected",
			configs: []config.APIConfig{
				{Alias: "test-config", APIKey: "sk-test-key", BaseURL: "https://api.example.com"},
			},
			selected:        0,
			expectViewState: ViewPingTesting,
			expectTesting:   true,
		},
		{
			name:            "press p with invalid selected negative",
			configs:         []config.APIConfig{{Alias: "test", APIKey: "key"}},
			selected:        -1,
			expectViewState: ViewDetail,
			expectTesting:   false,
		},
		{
			name:            "press p with invalid selected out of bounds",
			configs:         []config.APIConfig{{Alias: "test", APIKey: "key"}},
			selected:        5,
			expectViewState: ViewDetail,
			expectTesting:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs:   tt.configs,
				selected:  tt.selected,
				viewState: ViewDetail,
			}

			// Simulate pressing 'p' key
			newModel, cmd := m.handleDetailViewKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
			updatedModel := newModel.(Model)

			if updatedModel.viewState != tt.expectViewState {
				t.Errorf("handleDetailViewKeys('p') viewState = %v, want %v", updatedModel.viewState, tt.expectViewState)
			}
			if updatedModel.testing != tt.expectTesting {
				t.Errorf("handleDetailViewKeys('p') testing = %v, want %v", updatedModel.testing, tt.expectTesting)
			}
			if tt.expectTesting && cmd == nil {
				t.Error("handleDetailViewKeys('p') should return a command when testing")
			}
		})
	}
}

// TestPingResultMsgHandling tests the PingResultMsg handling in Update
// Requirements: 8.3, 8.4
func TestPingResultMsgHandling(t *testing.T) {
	tests := []struct {
		name              string
		msg               PingResultMsg
		expectSuccess     bool
		expectViewState   ViewState
		expectTestResult  bool
	}{
		{
			name: "successful ping",
			msg: PingResultMsg{
				Success:  true,
				Duration: 100 * 1000000, // 100ms in nanoseconds
				Err:      nil,
			},
			expectSuccess:    true,
			expectViewState:  ViewPingResult,
			expectTestResult: true,
		},
		{
			name: "failed ping with error",
			msg: PingResultMsg{
				Success:  false,
				Duration: 0,
				Err:      &testError{msg: "connection refused"},
			},
			expectSuccess:    false,
			expectViewState:  ViewPingResult,
			expectTestResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				viewState: ViewPingTesting,
				testing:   true,
			}

			newModel, _ := m.Update(tt.msg)
			updatedModel := newModel.(Model)

			if updatedModel.viewState != tt.expectViewState {
				t.Errorf("Update(PingResultMsg) viewState = %v, want %v", updatedModel.viewState, tt.expectViewState)
			}
			if updatedModel.testing != false {
				t.Error("Update(PingResultMsg) should set testing to false")
			}
			if tt.expectTestResult && updatedModel.testResult == nil {
				t.Error("Update(PingResultMsg) should set testResult")
			}
			if updatedModel.testResult != nil && updatedModel.testResult.Success != tt.expectSuccess {
				t.Errorf("Update(PingResultMsg) testResult.Success = %v, want %v", updatedModel.testResult.Success, tt.expectSuccess)
			}
		})
	}
}

// TestHandlePingResultViewKeys tests keyboard handling in ping result view
// Requirements: 8.3
func TestHandlePingResultViewKeys(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		expectViewState ViewState
		expectRetry     bool
	}{
		{
			name:            "return with esc",
			key:             "esc",
			expectViewState: ViewMain,
			expectRetry:     false,
		},
		{
			name:            "return with enter",
			key:             "enter",
			expectViewState: ViewMain,
			expectRetry:     false,
		},
		{
			name:            "return with q",
			key:             "q",
			expectViewState: ViewMain,
			expectRetry:     false,
		},
		{
			name:            "retry with r",
			key:             "r",
			expectViewState: ViewPingTesting,
			expectRetry:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				viewState: ViewPingResult,
				configs: []config.APIConfig{
					{Alias: "test-config", APIKey: "key", BaseURL: "https://api.example.com"},
				},
				cursor: 0,
				testResult: &TestResult{
					Success: true,
					Message: "连接成功",
				},
			}

			var keyMsg tea.KeyMsg
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			} else if tt.key == "enter" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
			} else {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			newModel, cmd := m.handlePingResultViewKeys(keyMsg)
			updatedModel := newModel.(Model)

			if updatedModel.viewState != tt.expectViewState {
				t.Errorf("handlePingResultViewKeys(%q) viewState = %v, want %v", tt.key, updatedModel.viewState, tt.expectViewState)
			}
			if tt.expectRetry {
				if cmd == nil {
					t.Error("handlePingResultViewKeys('r') should return a command for retry")
				}
				if !updatedModel.testing {
					t.Error("handlePingResultViewKeys('r') should set testing to true")
				}
			} else {
				if updatedModel.testResult != nil {
					t.Error("handlePingResultViewKeys() should clear testResult when returning")
				}
			}
		})
	}
}

// TestRenderPingTestingView tests the RenderPingTestingView method
// Requirements: 8.2
func TestRenderPingTestingView(t *testing.T) {
	m := Model{
		configs: []config.APIConfig{
			{Alias: "test-config", BaseURL: "https://api.example.com"},
		},
		cursor:    0,
		viewState: ViewPingTesting,
		testing:   true,
	}

	output := m.RenderPingTestingView()

	// Check that the title is present
	if !strings.Contains(output, "连接测试") {
		t.Error("RenderPingTestingView() should contain '连接测试'")
	}

	// Check that config info is shown
	if !strings.Contains(output, "test-config") {
		t.Error("RenderPingTestingView() should contain config alias")
	}

	// Check that URL is shown
	if !strings.Contains(output, "https://api.example.com") {
		t.Error("RenderPingTestingView() should contain URL")
	}

	// Check that testing indicator is shown
	if !strings.Contains(output, "正在测试") {
		t.Error("RenderPingTestingView() should contain testing indicator")
	}
}

// TestRenderPingResultView tests the RenderPingResultView method
// Requirements: 8.3, 8.4
func TestRenderPingResultView(t *testing.T) {
	tests := []struct {
		name           string
		testResult     *TestResult
		expectSuccess  bool
		expectContains []string
	}{
		{
			name: "successful result",
			testResult: &TestResult{
				Success:  true,
				Message:  "连接成功",
				Duration: "100ms",
			},
			expectSuccess:  true,
			expectContains: []string{"连接测试结果", "连接成功", "100ms"},
		},
		{
			name: "failed result",
			testResult: &TestResult{
				Success:  false,
				Message:  "连接被拒绝",
				Duration: "50ms",
			},
			expectSuccess:  false,
			expectContains: []string{"连接测试结果", "连接失败", "连接被拒绝"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs: []config.APIConfig{
					{Alias: "test-config", BaseURL: "https://api.example.com"},
				},
				cursor:     0,
				viewState:  ViewPingResult,
				testResult: tt.testResult,
			}

			output := m.RenderPingResultView()

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("RenderPingResultView() should contain %q", expected)
				}
			}

			// Check help text
			if !strings.Contains(output, "r:") || !strings.Contains(output, "重试") {
				t.Error("RenderPingResultView() should contain retry help text")
			}
		})
	}
}

// TestHandlePingTestingViewKeys tests keyboard handling during ping testing
func TestHandlePingTestingViewKeys(t *testing.T) {
	m := Model{
		viewState: ViewPingTesting,
		testing:   true,
	}

	// Test that regular keys don't do anything during testing
	newModel, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	updatedModel := newModel.(Model)

	if updatedModel.viewState != ViewPingTesting {
		t.Error("handleKeyMsg() should not change view state during testing")
	}
}


// TestWindowSizeMsg tests the WindowSizeMsg handling in Update
// Requirements: 11.1
func TestWindowSizeMsg(t *testing.T) {
	tests := []struct {
		name         string
		initialWidth int
		initialHeight int
		newWidth     int
		newHeight    int
	}{
		{
			name:          "resize to larger",
			initialWidth:  80,
			initialHeight: 24,
			newWidth:      120,
			newHeight:     40,
		},
		{
			name:          "resize to smaller",
			initialWidth:  120,
			initialHeight: 40,
			newWidth:      80,
			newHeight:     24,
		},
		{
			name:          "resize width only",
			initialWidth:  80,
			initialHeight: 24,
			newWidth:      120,
			newHeight:     24,
		},
		{
			name:          "resize height only",
			initialWidth:  80,
			initialHeight: 24,
			newWidth:      80,
			newHeight:     40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				width:  tt.initialWidth,
				height: tt.initialHeight,
			}

			newModel, _ := m.Update(tea.WindowSizeMsg{
				Width:  tt.newWidth,
				Height: tt.newHeight,
			})
			updatedModel := newModel.(Model)

			if updatedModel.width != tt.newWidth {
				t.Errorf("Update(WindowSizeMsg) width = %d, want %d", updatedModel.width, tt.newWidth)
			}
			if updatedModel.height != tt.newHeight {
				t.Errorf("Update(WindowSizeMsg) height = %d, want %d", updatedModel.height, tt.newHeight)
			}
		})
	}
}

// TestGetVisibleListHeight tests the getVisibleListHeight method
// Requirements: 11.1, 11.3
func TestGetVisibleListHeight(t *testing.T) {
	tests := []struct {
		name           string
		height         int
		expectMinHeight int
	}{
		{
			name:            "normal height",
			height:          24,
			expectMinHeight: 1,
		},
		{
			name:            "large height",
			height:          50,
			expectMinHeight: 1,
		},
		{
			name:            "small height",
			height:          10,
			expectMinHeight: 1,
		},
		{
			name:            "very small height",
			height:          5,
			expectMinHeight: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				height: tt.height,
			}

			visibleHeight := m.getVisibleListHeight()

			if visibleHeight < tt.expectMinHeight {
				t.Errorf("getVisibleListHeight() = %d, want at least %d", visibleHeight, tt.expectMinHeight)
			}
		})
	}
}

// TestAdjustScrollOffset tests the adjustScrollOffset method
// Requirements: 11.3
func TestAdjustScrollOffset(t *testing.T) {
	tests := []struct {
		name               string
		configs            []config.APIConfig
		cursor             int
		scrollOffset       int
		height             int
		expectScrollOffset int
	}{
		{
			name:               "cursor visible no scroll needed",
			configs:            makeConfigs(5),
			cursor:             2,
			scrollOffset:       0,
			height:             30,
			expectScrollOffset: 0,
		},
		{
			name:               "cursor below visible area",
			configs:            makeConfigs(20),
			cursor:             15,
			scrollOffset:       0,
			height:             15,
			expectScrollOffset: 8, // cursor should be visible at bottom
		},
		{
			name:               "cursor above visible area",
			configs:            makeConfigs(20),
			cursor:             2,
			scrollOffset:       10,
			height:             15,
			expectScrollOffset: 2, // scroll up to show cursor
		},
		{
			name:               "empty configs",
			configs:            []config.APIConfig{},
			cursor:             0,
			scrollOffset:       0,
			height:             24,
			expectScrollOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				configs:      tt.configs,
				cursor:       tt.cursor,
				scrollOffset: tt.scrollOffset,
				height:       tt.height,
			}

			m.adjustScrollOffset()

			// Verify cursor is within visible range
			visibleHeight := m.getVisibleListHeight()
			if len(tt.configs) > 0 {
				if m.cursor < m.scrollOffset {
					t.Errorf("adjustScrollOffset() cursor %d is above scrollOffset %d", m.cursor, m.scrollOffset)
				}
				if m.cursor >= m.scrollOffset+visibleHeight && m.cursor < len(tt.configs) {
					t.Errorf("adjustScrollOffset() cursor %d is below visible area (scrollOffset=%d, visibleHeight=%d)", m.cursor, m.scrollOffset, visibleHeight)
				}
			}
		})
	}
}

// TestScrollingWithCursorMovement tests that scrolling works correctly with cursor movement
// Requirements: 11.3
func TestScrollingWithCursorMovement(t *testing.T) {
	// Create a model with many configs and small height
	configs := makeConfigs(20)
	m := Model{
		configs:      configs,
		cursor:       0,
		scrollOffset: 0,
		height:       15, // Small height to force scrolling
	}

	// Move cursor down multiple times
	for i := 0; i < 15; i++ {
		m.moveDown()
	}

	// Cursor should be at 15
	if m.cursor != 15 {
		t.Errorf("After 15 moveDown(), cursor = %d, want 15", m.cursor)
	}

	// Scroll offset should have adjusted to keep cursor visible
	visibleHeight := m.getVisibleListHeight()
	if m.cursor < m.scrollOffset || m.cursor >= m.scrollOffset+visibleHeight {
		t.Errorf("Cursor %d is not visible (scrollOffset=%d, visibleHeight=%d)", m.cursor, m.scrollOffset, visibleHeight)
	}

	// Move to bottom
	m.moveToBottom()
	if m.cursor != len(configs)-1 {
		t.Errorf("After moveToBottom(), cursor = %d, want %d", m.cursor, len(configs)-1)
	}

	// Move to top
	m.moveToTop()
	if m.cursor != 0 {
		t.Errorf("After moveToTop(), cursor = %d, want 0", m.cursor)
	}
	if m.scrollOffset != 0 {
		t.Errorf("After moveToTop(), scrollOffset = %d, want 0", m.scrollOffset)
	}
}

// TestRenderMainViewWithScrolling tests the RenderMainView with scrolling
// Requirements: 11.3
func TestRenderMainViewWithScrolling(t *testing.T) {
	// Create a model with many configs
	configs := makeConfigs(20)
	m := Model{
		configs:      configs,
		cursor:       10,
		scrollOffset: 5,
		height:       15,
		width:        80,
	}

	output := m.RenderMainView()

	// Should contain scroll indicators
	if !strings.Contains(output, "↑") {
		t.Error("RenderMainView() should contain up scroll indicator when scrolled down")
	}
	if !strings.Contains(output, "↓") {
		t.Error("RenderMainView() should contain down scroll indicator when more items below")
	}
}

// TestGetEffectiveWidth tests the getEffectiveWidth method
// Requirements: 11.2
func TestGetEffectiveWidth(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		defaultWidth int
		expectWidth  int
	}{
		{
			name:         "zero width uses default",
			width:        0,
			defaultWidth: 40,
			expectWidth:  40,
		},
		{
			name:         "negative width uses default",
			width:        -1,
			defaultWidth: 40,
			expectWidth:  40,
		},
		{
			name:         "small width",
			width:        30,
			defaultWidth: 40,
			expectWidth:  28, // width - 2 margin
		},
		{
			name:         "large width capped at 80",
			width:        120,
			defaultWidth: 40,
			expectWidth:  80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				width: tt.width,
			}

			effectiveWidth := m.getEffectiveWidth(tt.defaultWidth)

			if effectiveWidth != tt.expectWidth {
				t.Errorf("getEffectiveWidth(%d) = %d, want %d", tt.defaultWidth, effectiveWidth, tt.expectWidth)
			}
		})
	}
}

// TestTruncateText tests the truncateText method
// Requirements: 11.2
func TestTruncateText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxWidth  int
		expected  string
	}{
		{
			name:     "short text no truncation",
			text:     "hello",
			maxWidth: 10,
			expected: "hello",
		},
		{
			name:     "exact length no truncation",
			text:     "hello",
			maxWidth: 5,
			expected: "hello",
		},
		{
			name:     "long text truncated",
			text:     "hello world this is a long text",
			maxWidth: 15,
			expected: "hello world ...",
		},
		{
			name:     "very small maxWidth",
			text:     "hello",
			maxWidth: 3,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{}

			result := m.truncateText(tt.text, tt.maxWidth)

			if result != tt.expected {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.text, tt.maxWidth, result, tt.expected)
			}
		})
	}
}

// makeConfigs creates a slice of test configs
func makeConfigs(count int) []config.APIConfig {
	configs := make([]config.APIConfig, count)
	for i := 0; i < count; i++ {
		configs[i] = config.APIConfig{
			Alias:  fmt.Sprintf("config-%d", i),
			APIKey: fmt.Sprintf("key-%d", i),
		}
	}
	return configs
}
