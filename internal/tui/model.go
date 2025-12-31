// Package tui provides a terminal user interface for apimgr
package tui

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"apimgr/config"
	"apimgr/internal/compatibility"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewState represents the current view state
type ViewState int

const (
	ViewMain          ViewState = iota // Main list view
	ViewDetail                         // Detail view
	ViewAdd                            // Add config form
	ViewEdit                           // Edit config form
	ViewDelete                         // Delete confirmation dialog
	ViewHelp                           // Help panel
	ViewModelSelect                    // Model selection list
	ViewPingTesting                    // Ping test in progress
	ViewPingResult                     // Ping test result
	ViewCompatTesting                  // Compatibility test in progress
	ViewCompatResult                   // Compatibility test result
)

// Model is the core state model for TUI
type Model struct {
	configs       []config.APIConfig // Config list
	activeAlias   string             // Current active config alias
	cursor        int                // Current cursor position
	selected      int                // Currently selected config index
	viewState     ViewState          // Current view state
	configManager *config.Manager    // Config manager

	// Form related
	formInputs []textinput.Model // Form input fields
	formFocus  int               // Currently focused input field

	// Messages and errors
	message  string // Status message
	errorMsg string // Error message

	// Window size
	width  int
	height int

	// Scroll state - Requirements: 11.1, 11.2, 11.3
	scrollOffset      int // Scroll offset for main list view
	modelScrollOffset int // Scroll offset for model selection list

	// Test state
	testing    bool        // Whether testing is in progress
	testResult *TestResult // Test result

	// Compatibility test state
	compatResult *CompatTestResult // Compatibility test result

	// Model selection state
	modelCursor int      // Cursor position in model selection list
	modelList   []string // Available models for current config

	// Help view scroll state
	helpScrollOffset int // Scroll offset for help view
}

// CompatTestResult holds compatibility test result data
type CompatTestResult struct {
	Success            bool
	CompatibilityLevel string // "full", "partial", "none"
	Checks             []CompatCheck
	ResponseTime       string
	Error              string
}

// CompatCheck represents a single compatibility check result
type CompatCheck struct {
	Name     string
	Passed   bool
	Message  string
	Critical bool
}

// TestResult holds test result data
type TestResult struct {
	Success  bool
	Message  string
	Duration string
}

// NewModel creates a new TUI model
func NewModel(cm *config.Manager) Model {
	return Model{
		configs:           []config.APIConfig{},
		cursor:            0,
		selected:          -1,
		viewState:         ViewMain,
		configManager:     cm,
		formInputs:        []textinput.Model{},
		formFocus:         0,
		width:             80,
		height:            24,
		scrollOffset:      0,
		modelScrollOffset: 0,
	}
}

// Init initializes the model and returns initial commands
func (m Model) Init() tea.Cmd {
	return loadConfigs(m.configManager)
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust scroll offset if needed after window resize - Requirements: 11.1
		m.adjustScrollOffset()
		return m, nil

	case ConfigsLoadedMsg:
		m.configs = msg.Configs
		m.activeAlias = msg.ActiveAlias
		// Adjust cursor if it's out of bounds after reload (e.g., after deletion)
		if len(m.configs) > 0 && m.cursor >= len(m.configs) {
			m.cursor = len(m.configs) - 1
		}
		// Also adjust selected if needed
		if m.selected >= len(m.configs) {
			m.selected = -1
		}
		return m, nil

	case ConfigSwitchedMsg:
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		} else {
			if msg.IsLocal {
				m.message = "已本地切换到: " + msg.Alias + " (仅当前终端会话)"
			} else {
				m.activeAlias = msg.Alias
				m.message = "已全局切换到: " + msg.Alias
			}
		}
		return m, nil

	case ConfigAddedMsg:
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		} else {
			m.message = "配置已添加: " + msg.Config.Alias
			m.viewState = ViewMain
			m.formInputs = []textinput.Model{}
			m.formFocus = 0
			// Reload configs
			return m, loadConfigs(m.configManager)
		}
		return m, nil

	case ConfigUpdatedMsg:
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		} else {
			m.message = "配置已更新: " + msg.Alias
			m.viewState = ViewMain
			m.formInputs = []textinput.Model{}
			m.formFocus = 0
			// Reload configs
			return m, loadConfigs(m.configManager)
		}
		return m, nil

	case ConfigDeletedMsg:
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		} else {
			m.message = "配置已删除: " + msg.Alias
			// If deleted config was active, clear active alias
			if m.activeAlias == msg.Alias {
				m.activeAlias = ""
			}
			// Reload configs
			return m, loadConfigs(m.configManager)
		}
		m.viewState = ViewMain
		return m, nil

	case ModelSwitchedMsg:
		if msg.Err != nil {
			m.errorMsg = msg.Err.Error()
		} else {
			m.message = "模型已切换到: " + msg.Model
			m.viewState = ViewMain
			// Reload configs to reflect the change
			return m, loadConfigs(m.configManager)
		}
		m.viewState = ViewMain
		return m, nil

	case PingResultMsg:
		m.testing = false
		if msg.Err != nil {
			m.testResult = &TestResult{
				Success:  false,
				Message:  msg.Err.Error(),
				Duration: "",
			}
		} else {
			m.testResult = &TestResult{
				Success:  msg.Success,
				Message:  "连接成功",
				Duration: msg.Duration.String(),
			}
		}
		m.viewState = ViewPingResult
		return m, nil

	case CompatResultMsg:
		m.testing = false
		if msg.Err != nil {
			m.compatResult = &CompatTestResult{
				Success:            false,
				CompatibilityLevel: compatibility.CompatibilityNone,
				Error:              msg.Err.Error(),
			}
		} else if msg.Result != nil {
			// Convert compatibility.TestResult to CompatTestResult
			checks := make([]CompatCheck, len(msg.Result.Checks))
			for i, c := range msg.Result.Checks {
				checks[i] = CompatCheck{
					Name:     c.Name,
					Passed:   c.Passed,
					Message:  c.Message,
					Critical: c.Critical,
				}
			}
			m.compatResult = &CompatTestResult{
				Success:            msg.Result.Success,
				CompatibilityLevel: msg.Result.CompatibilityLevel,
				Checks:             checks,
				ResponseTime:       msg.Result.ResponseTime.String(),
				Error:              msg.Result.Error,
			}
		}
		m.viewState = ViewCompatResult
		return m, nil

	case errMsg:
		m.errorMsg = string(msg)
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.viewState {
	case ViewMain:
		return m.handleMainViewKeys(msg)
	case ViewDetail:
		return m.handleDetailViewKeys(msg)
	case ViewAdd, ViewEdit:
		return m.handleFormViewKeys(msg)
	case ViewDelete:
		return m.handleDeleteViewKeys(msg)
	case ViewHelp:
		return m.handleHelpViewKeys(msg)
	case ViewModelSelect:
		return m.handleModelSelectViewKeys(msg)
	case ViewPingTesting:
		// During testing, only allow quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	case ViewCompatTesting:
		// During compatibility testing, only allow quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	case ViewPingResult:
		return m.handlePingResultViewKeys(msg)
	case ViewCompatResult:
		return m.handleCompatResultViewKeys(msg)
	default:
		return m, nil
	}
}

// handleMainViewKeys handles keyboard input in main view
func (m Model) handleMainViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		m.moveDown()
		// Clear messages on navigation
		m.message = ""
		m.errorMsg = ""
		return m, nil

	case "k", "up":
		m.moveUp()
		// Clear messages on navigation
		m.message = ""
		m.errorMsg = ""
		return m, nil

	case "g":
		m.moveToTop()
		// Clear messages on navigation
		m.message = ""
		m.errorMsg = ""
		return m, nil

	case "G":
		m.moveToBottom()
		// Clear messages on navigation
		m.message = ""
		m.errorMsg = ""
		return m, nil

	case "enter":
		if len(m.configs) > 0 {
			m.selected = m.cursor
			m.viewState = ViewDetail
		}
		return m, nil

	case "s":
		// Switch local (Claude Code only) - sync to Claude Code settings without changing global active
		if len(m.configs) > 0 && m.cursor >= 0 && m.cursor < len(m.configs) {
			cfg := m.configs[m.cursor]
			// Clear previous messages
			m.message = ""
			m.errorMsg = ""
			return m, switchLocalConfig(m.configManager, &cfg)
		}
		return m, nil

	case "S":
		// Switch global active config - Requirements: 4.1, 4.2, 4.3, 4.4
		if len(m.configs) > 0 && m.cursor >= 0 && m.cursor < len(m.configs) {
			alias := m.configs[m.cursor].Alias
			// Clear previous messages
			m.message = ""
			m.errorMsg = ""
			return m, switchGlobalConfig(m.configManager, alias)
		}
		return m, nil

	case "a":
		// Add new config - Requirements: 5.1
		m.initAddForm()
		return m, nil

	case "e":
		// Edit selected config - Requirements: 6.1
		if len(m.configs) > 0 && m.cursor >= 0 && m.cursor < len(m.configs) {
			m.initEditForm()
		}
		return m, nil

	case "d":
		// Delete selected config - Requirements: 7.1
		if len(m.configs) > 0 && m.cursor >= 0 && m.cursor < len(m.configs) {
			m.viewState = ViewDelete
			m.message = ""
			m.errorMsg = ""
		}
		return m, nil

	case "?":
		m.viewState = ViewHelp
		m.helpScrollOffset = 0 // Reset scroll when opening help
		return m, nil

	case "m":
		// Switch model - Requirements: 12.1, 12.2, 12.4
		if len(m.configs) > 0 && m.cursor >= 0 && m.cursor < len(m.configs) {
			cfg := m.configs[m.cursor]
			if len(cfg.Models) <= 1 {
				// No multiple models to switch - Requirements: 12.4
				m.errorMsg = "此配置没有定义多个模型可供切换"
				return m, nil
			}
			m.initModelSelect(cfg)
		}
		return m, nil

	case "p":
		// Ping test - Requirements: 8.1, 8.2, 8.3, 8.4
		if len(m.configs) > 0 && m.cursor >= 0 && m.cursor < len(m.configs) {
			cfg := m.configs[m.cursor]
			m.testing = true
			m.viewState = ViewPingTesting
			m.message = ""
			m.errorMsg = ""
			return m, pingConfig(&cfg)
		}
		return m, nil

	case "t":
		// Compatibility test - Requirements: 9.1, 9.2, 9.3, 9.4
		if len(m.configs) > 0 && m.cursor >= 0 && m.cursor < len(m.configs) {
			cfg := m.configs[m.cursor]
			m.testing = true
			m.viewState = ViewCompatTesting
			m.message = ""
			m.errorMsg = ""
			m.compatResult = nil
			return m, runCompatibilityTest(&cfg)
		}
		return m, nil
	}

	return m, nil
}

// handleDetailViewKeys handles keyboard input in detail view
func (m Model) handleDetailViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.viewState = ViewMain
		return m, nil

	case "s":
		// Switch local (Claude Code only) from detail view
		if m.selected >= 0 && m.selected < len(m.configs) {
			cfg := m.configs[m.selected]
			// Clear previous messages
			m.message = ""
			m.errorMsg = ""
			return m, switchLocalConfig(m.configManager, &cfg)
		}
		return m, nil

	case "S":
		// Switch global active config from detail view - Requirements: 4.1, 4.2, 4.3, 4.4
		if m.selected >= 0 && m.selected < len(m.configs) {
			alias := m.configs[m.selected].Alias
			// Clear previous messages
			m.message = ""
			m.errorMsg = ""
			return m, switchGlobalConfig(m.configManager, alias)
		}
		return m, nil

	case "e":
		// Edit selected config from detail view - Requirements: 6.1
		if m.selected >= 0 && m.selected < len(m.configs) {
			// Set cursor to selected for initEditForm to work correctly
			m.cursor = m.selected
			m.initEditForm()
		}
		return m, nil

	case "d":
		// Delete selected config from detail view - Requirements: 7.1
		if m.selected >= 0 && m.selected < len(m.configs) {
			// Set cursor to selected for delete to work correctly
			m.cursor = m.selected
			m.viewState = ViewDelete
			m.message = ""
			m.errorMsg = ""
		}
		return m, nil

	case "?":
		m.viewState = ViewHelp
		m.helpScrollOffset = 0 // Reset scroll when opening help
		return m, nil

	case "m":
		// Switch model from detail view - Requirements: 12.1, 12.2, 12.4
		if m.selected >= 0 && m.selected < len(m.configs) {
			cfg := m.configs[m.selected]
			if len(cfg.Models) <= 1 {
				// No multiple models to switch - Requirements: 12.4
				m.errorMsg = "此配置没有定义多个模型可供切换"
				return m, nil
			}
			// Set cursor to selected for initModelSelect to work correctly
			m.cursor = m.selected
			m.initModelSelect(cfg)
		}
		return m, nil

	case "p":
		// Ping test from detail view - Requirements: 8.1, 8.2, 8.3, 8.4
		if m.selected >= 0 && m.selected < len(m.configs) {
			cfg := m.configs[m.selected]
			m.testing = true
			m.viewState = ViewPingTesting
			m.message = ""
			m.errorMsg = ""
			return m, pingConfig(&cfg)
		}
		return m, nil

	case "t":
		// Compatibility test from detail view - Requirements: 9.1, 9.2, 9.3, 9.4
		if m.selected >= 0 && m.selected < len(m.configs) {
			cfg := m.configs[m.selected]
			m.testing = true
			m.viewState = ViewCompatTesting
			m.message = ""
			m.errorMsg = ""
			m.compatResult = nil
			return m, runCompatibilityTest(&cfg)
		}
		return m, nil
	}

	return m, nil
}

// moveUp moves cursor up
// Requirements: 2.2, 11.3
func (m *Model) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.adjustScrollOffset()
	}
}

// moveDown moves cursor down
// Requirements: 2.1, 11.3
func (m *Model) moveDown() {
	if len(m.configs) > 0 && m.cursor < len(m.configs)-1 {
		m.cursor++
		m.adjustScrollOffset()
	}
}

// moveToTop moves cursor to top
// Requirements: 2.3, 11.3
func (m *Model) moveToTop() {
	m.cursor = 0
	m.scrollOffset = 0
}

// moveToBottom moves cursor to bottom
// Requirements: 2.4, 11.3
func (m *Model) moveToBottom() {
	if len(m.configs) > 0 {
		m.cursor = len(m.configs) - 1
		m.adjustScrollOffset()
	}
}

// getVisibleListHeight returns the number of lines available for the config list
// Requirements: 11.1, 11.3
func (m *Model) getVisibleListHeight() int {
	// Account for:
	// - Title line (1)
	// - Separator line (1)
	// - Empty line after separator (1)
	// - Empty line before bottom separator (1)
	// - Bottom separator (1)
	// - Status bar (at least 2 lines for shortcuts + potential messages)
	headerLines := 3
	footerLines := 4
	
	available := m.height - headerLines - footerLines
	if available < 1 {
		available = 1
	}
	return available
}

// adjustScrollOffset adjusts the scroll offset to keep cursor visible
// Requirements: 11.3
func (m *Model) adjustScrollOffset() {
	visibleHeight := m.getVisibleListHeight()
	
	// If cursor is above visible area, scroll up
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	
	// If cursor is below visible area, scroll down
	if m.cursor >= m.scrollOffset+visibleHeight {
		m.scrollOffset = m.cursor - visibleHeight + 1
	}
	
	// Ensure scroll offset is not negative
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	
	// Ensure we don't scroll past the end
	maxOffset := len(m.configs) - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
}

// View renders the UI
func (m Model) View() string {
	switch m.viewState {
	case ViewHelp:
		return m.RenderHelpView()
	case ViewDetail:
		return m.RenderDetailView()
	case ViewAdd, ViewEdit:
		return m.RenderFormViewFull()
	case ViewDelete:
		return m.RenderDeleteConfirm()
	case ViewModelSelect:
		return m.RenderModelSelectView()
	case ViewPingTesting:
		return m.RenderPingTestingView()
	case ViewPingResult:
		return m.RenderPingResultView()
	case ViewCompatTesting:
		return m.RenderCompatTestingView()
	case ViewCompatResult:
		return m.RenderCompatResultView()
	default:
		return m.RenderMainView()
	}
}

// loadConfigs creates a command to load configs
func loadConfigs(cm *config.Manager) tea.Cmd {
	return func() tea.Msg {
		configs, err := cm.List()
		if err != nil {
			return errMsg(err.Error())
		}

		activeName, _ := cm.GetActiveName()

		return ConfigsLoadedMsg{
			Configs:     configs,
			ActiveAlias: activeName,
		}
	}
}

// errMsg is an error message type
type errMsg string

// switchLocalConfig creates a command to switch config locally (Claude Code only)
func switchLocalConfig(cm *config.Manager, cfg *config.APIConfig) tea.Cmd {
	return func() tea.Msg {
		err := cm.SyncClaudeSettingsOnly(cfg)
		if err != nil {
			return ConfigSwitchedMsg{
				Alias:   cfg.Alias,
				IsLocal: true,
				Err:     err,
			}
		}

		return ConfigSwitchedMsg{
			Alias:   cfg.Alias,
			IsLocal: true,
			Err:     nil,
		}
	}
}

// switchGlobalConfig creates a command to switch the global active configuration
// Requirements: 4.1, 4.2, 4.3, 4.4
func switchGlobalConfig(cm *config.Manager, alias string) tea.Cmd {
	return func() tea.Msg {
		err := cm.SetActive(alias)
		if err != nil {
			return ConfigSwitchedMsg{
				Alias:   alias,
				IsLocal: false,
				Err:     err,
			}
		}

		// Generate active script after successful switch
		if genErr := cm.GenerateActiveScript(); genErr != nil {
			// Log the error but don't fail the switch
			return ConfigSwitchedMsg{
				Alias:   alias,
				IsLocal: false,
				Err:     nil,
			}
		}

		return ConfigSwitchedMsg{
			Alias:   alias,
			IsLocal: false,
			Err:     nil,
		}
	}
}

// handleFormViewKeys handles keyboard input in form view (add/edit)
// Requirements: 5.2, 5.5, 6.2, 6.5
func (m Model) handleFormViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		// Cancel form and return to previous view
		m.viewState = ViewMain
		m.errorMsg = ""
		m.formInputs = []textinput.Model{}
		m.formFocus = 0
		return m, nil

	case "tab", "down":
		// Move to next field
		m.formFocus = NextFormField(m.formInputs, m.formFocus)
		return m, nil

	case "shift+tab", "up":
		// Move to previous field
		m.formFocus = PrevFormField(m.formInputs, m.formFocus)
		return m, nil

	case "enter":
		// Submit form
		formData := GetFormData(m.formInputs)
		if err := formData.Validate(); err != nil {
			m.errorMsg = err.Error()
			return m, nil
		}

		// Clear error and submit
		m.errorMsg = ""
		if m.viewState == ViewAdd {
			return m, m.submitAddForm(formData)
		}
		return m, m.submitEditForm(formData)

	default:
		// Pass key to focused input
		if m.formFocus >= 0 && m.formFocus < len(m.formInputs) {
			var cmd tea.Cmd
			m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

// initAddForm initializes the form for adding a new config
// Requirements: 5.1, 5.2
func (m *Model) initAddForm() {
	m.formInputs = FormInputs()
	m.formFocus = 0
	m.viewState = ViewAdd
	m.errorMsg = ""
}

// initEditForm initializes the form for editing an existing config
// Requirements: 6.1, 6.2
func (m *Model) initEditForm() {
	if m.cursor < 0 || m.cursor >= len(m.configs) {
		return
	}

	cfg := m.configs[m.cursor]
	m.formInputs = FormInputs()
	m.formFocus = 0
	m.viewState = ViewEdit
	m.errorMsg = ""

	// Pre-fill form with existing config data
	formData := FormData{
		Alias:     cfg.Alias,
		APIKey:    cfg.APIKey,
		AuthToken: cfg.AuthToken,
		BaseURL:   cfg.BaseURL,
		Model:     cfg.Model,
		Models:    strings.Join(cfg.Models, ", "),
	}
	SetFormData(m.formInputs, formData)
}

// submitAddForm creates a command to add a new config
// Requirements: 5.3
func (m *Model) submitAddForm(data FormData) tea.Cmd {
	return func() tea.Msg {
		newConfig := config.APIConfig{
			Alias:     strings.TrimSpace(data.Alias),
			APIKey:    strings.TrimSpace(data.APIKey),
			AuthToken: strings.TrimSpace(data.AuthToken),
			BaseURL:   strings.TrimSpace(data.BaseURL),
			Model:     strings.TrimSpace(data.Model),
			Models:    data.ParseModels(),
		}

		err := m.configManager.Add(newConfig)
		return ConfigAddedMsg{
			Config: newConfig,
			Err:    err,
		}
	}
}

// submitEditForm creates a command to update an existing config
// Requirements: 6.3
func (m *Model) submitEditForm(data FormData) tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.configs) {
		return nil
	}

	originalAlias := m.configs[m.cursor].Alias

	return func() tea.Msg {
		// Build updates map
		updates := map[string]string{
			"api_key":    strings.TrimSpace(data.APIKey),
			"auth_token": strings.TrimSpace(data.AuthToken),
			"base_url":   strings.TrimSpace(data.BaseURL),
			"model":      strings.TrimSpace(data.Model),
		}

		err := m.configManager.UpdatePartial(originalAlias, updates)
		return ConfigUpdatedMsg{
			Alias: originalAlias,
			Err:   err,
		}
	}
}

// RenderFormViewFull renders the complete form view
// Requirements: 5.2, 6.2
func (m Model) RenderFormViewFull() string {
	title := "添加配置"
	if m.viewState == ViewEdit {
		title = "编辑配置"
	}
	return RenderForm(m.formInputs, m.formFocus, title, m.errorMsg)
}

// handleDeleteViewKeys handles keyboard input in delete confirmation view
// Requirements: 7.1, 7.2, 7.4
func (m Model) handleDeleteViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "y", "Y":
		// Confirm delete - Requirements: 7.3
		if m.cursor >= 0 && m.cursor < len(m.configs) {
			alias := m.configs[m.cursor].Alias
			return m, deleteConfig(m.configManager, alias)
		}
		m.viewState = ViewMain
		return m, nil

	case "n", "N", "esc":
		// Cancel delete - Requirements: 7.4
		m.viewState = ViewMain
		m.message = ""
		m.errorMsg = ""
		return m, nil
	}

	return m, nil
}

// deleteConfig creates a command to delete a configuration
// Requirements: 7.3, 7.5
func deleteConfig(cm *config.Manager, alias string) tea.Cmd {
	return func() tea.Msg {
		err := cm.Remove(alias)
		return ConfigDeletedMsg{
			Alias: alias,
			Err:   err,
		}
	}
}

// handleHelpViewKeys handles keyboard input in help view
func (m Model) handleHelpViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "q":
		// Close help and return to main view
		m.viewState = ViewMain
		m.helpScrollOffset = 0
		return m, nil

	case "j", "down":
		// Scroll down
		m.helpScrollOffset++
		m.adjustHelpScrollOffset()
		return m, nil

	case "k", "up":
		// Scroll up
		if m.helpScrollOffset > 0 {
			m.helpScrollOffset--
		}
		return m, nil

	case "g":
		// Jump to top
		m.helpScrollOffset = 0
		return m, nil

	case "G":
		// Jump to bottom
		m.helpScrollOffset = m.getHelpContentHeight() - m.getVisibleHelpHeight()
		m.adjustHelpScrollOffset()
		return m, nil
	}

	return m, nil
}

// getHelpContentHeight returns the total number of lines in help content
func (m *Model) getHelpContentHeight() int {
	// Count all help content lines:
	// Title (1) + Separator (1) + Empty (1)
	// Navigation section: header (1) + 5 items + empty (1) = 7
	// Config management section: header (1) + 5 items + empty (1) = 7
	// Model management section: header (1) + 1 item + empty (1) = 3
	// Testing section: header (1) + 2 items + empty (1) = 4
	// General section: header (1) + 4 items + empty (1) = 6
	// Footer separator (1) + help text (1) = 2
	return 3 + 7 + 7 + 3 + 4 + 6 + 2
}

// getVisibleHelpHeight returns the number of lines available for help content
func (m *Model) getVisibleHelpHeight() int {
	// Reserve space for title, separator, and footer
	headerLines := 3
	footerLines := 2

	available := m.height - headerLines - footerLines
	if available < 1 {
		available = 1
	}
	return available
}

// adjustHelpScrollOffset adjusts the help scroll offset to stay within bounds
func (m *Model) adjustHelpScrollOffset() {
	maxOffset := m.getHelpContentHeight() - m.getVisibleHelpHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.helpScrollOffset > maxOffset {
		m.helpScrollOffset = maxOffset
	}
	if m.helpScrollOffset < 0 {
		m.helpScrollOffset = 0
	}
}

// initModelSelect initializes the model selection view
// Requirements: 12.1, 12.2
func (m *Model) initModelSelect(cfg config.APIConfig) {
	m.modelList = cfg.Models
	m.modelCursor = 0
	// Find current active model position
	for i, model := range cfg.Models {
		if model == cfg.Model {
			m.modelCursor = i
			break
		}
	}
	m.viewState = ViewModelSelect
	m.message = ""
	m.errorMsg = ""
}

// handleModelSelectViewKeys handles keyboard input in model selection view
// Requirements: 12.1, 12.2, 12.3
func (m Model) handleModelSelectViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		// Cancel model selection and return to main view
		m.viewState = ViewMain
		m.modelList = nil
		m.modelCursor = 0
		m.modelScrollOffset = 0
		return m, nil

	case "j", "down":
		// Move cursor down in model list
		if len(m.modelList) > 0 && m.modelCursor < len(m.modelList)-1 {
			m.modelCursor++
			m.adjustModelScrollOffset()
		}
		return m, nil

	case "k", "up":
		// Move cursor up in model list
		if m.modelCursor > 0 {
			m.modelCursor--
			m.adjustModelScrollOffset()
		}
		return m, nil

	case "g":
		// Jump to top of model list
		m.modelCursor = 0
		m.modelScrollOffset = 0
		return m, nil

	case "G":
		// Jump to bottom of model list
		if len(m.modelList) > 0 {
			m.modelCursor = len(m.modelList) - 1
			m.adjustModelScrollOffset()
		}
		return m, nil

	case "enter":
		// Confirm model selection - Requirements: 12.3
		if m.cursor >= 0 && m.cursor < len(m.configs) && m.modelCursor >= 0 && m.modelCursor < len(m.modelList) {
			alias := m.configs[m.cursor].Alias
			selectedModel := m.modelList[m.modelCursor]
			m.viewState = ViewMain
			m.modelList = nil
			m.modelScrollOffset = 0
			return m, switchModel(m.configManager, alias, selectedModel)
		}
		m.viewState = ViewMain
		return m, nil
	}

	return m, nil
}

// getVisibleModelListHeight returns the number of lines available for the model list
// Requirements: 11.1, 11.3
func (m *Model) getVisibleModelListHeight() int {
	// Account for:
	// - Title line (1)
	// - Separator line (1)
	// - Empty line (1)
	// - Config info (2 lines)
	// - Empty line (1)
	// - Empty line before footer (1)
	// - Footer separator (1)
	// - Help text (1)
	headerLines := 6
	footerLines := 3
	
	available := m.height - headerLines - footerLines
	if available < 1 {
		available = 1
	}
	return available
}

// adjustModelScrollOffset adjusts the model scroll offset to keep cursor visible
// Requirements: 11.3
func (m *Model) adjustModelScrollOffset() {
	visibleHeight := m.getVisibleModelListHeight()
	
	// If cursor is above visible area, scroll up
	if m.modelCursor < m.modelScrollOffset {
		m.modelScrollOffset = m.modelCursor
	}
	
	// If cursor is below visible area, scroll down
	if m.modelCursor >= m.modelScrollOffset+visibleHeight {
		m.modelScrollOffset = m.modelCursor - visibleHeight + 1
	}
	
	// Ensure scroll offset is not negative
	if m.modelScrollOffset < 0 {
		m.modelScrollOffset = 0
	}
	
	// Ensure we don't scroll past the end
	maxOffset := len(m.modelList) - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.modelScrollOffset > maxOffset {
		m.modelScrollOffset = maxOffset
	}
}

// switchModel creates a command to switch the active model for a configuration
// Requirements: 12.3
func switchModel(cm *config.Manager, alias string, model string) tea.Cmd {
	return func() tea.Msg {
		err := cm.SwitchModel(alias, model)
		if err != nil {
			return ModelSwitchedMsg{
				Alias: alias,
				Model: model,
				Err:   err,
			}
		}

		// Regenerate active script if this is the active config
		activeName, _ := cm.GetActiveName()
		if activeName == alias {
			if genErr := cm.GenerateActiveScript(); genErr != nil {
				// Log the error but don't fail the switch
				return ModelSwitchedMsg{
					Alias: alias,
					Model: model,
					Err:   nil,
				}
			}
		}

		return ModelSwitchedMsg{
			Alias: alias,
			Model: model,
			Err:   nil,
		}
	}
}

// pingConfig creates a command to perform a ping test on a configuration
// Requirements: 8.1, 8.2, 8.3, 8.4
func pingConfig(cfg *config.APIConfig) tea.Cmd {
	return func() tea.Msg {
		return performPingTest(cfg)
	}
}

// performPingTest performs the actual ping test
// Requirements: 8.1, 8.2, 8.3, 8.4
func performPingTest(cfg *config.APIConfig) PingResultMsg {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          10,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// Create request
	req, err := http.NewRequest("HEAD", baseURL, nil)
	if err != nil {
		return PingResultMsg{
			Success:  false,
			Duration: 0,
			Err:      fmt.Errorf("创建请求失败: %v", err),
		}
	}

	// Add auth headers
	if cfg.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.AuthToken))
	} else if cfg.APIKey != "" {
		req.Header.Set("x-api-key", cfg.APIKey)
		req.Header.Set("API-Key", cfg.APIKey)
	}

	// Perform request
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		// Categorize errors
		var errMsg string
		errStr := err.Error()

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			errMsg = "请求超时 (超过10秒)"
		} else if strings.Contains(errStr, "connection refused") {
			errMsg = "连接被拒绝 (服务器未监听此端口)"
		} else if strings.Contains(errStr, "network is unreachable") {
			errMsg = "网络不可达"
		} else if strings.Contains(errStr, "EOF") {
			errMsg = "连接意外关闭"
		} else if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "NXDOMAIN") {
			errMsg = "DNS 解析失败 (域名不存在)"
		} else {
			errMsg = fmt.Sprintf("连接失败: %v", err)
		}

		return PingResultMsg{
			Success:  false,
			Duration: duration,
			Err:      fmt.Errorf("%s", errMsg),
		}
	}
	defer resp.Body.Close()

	// Check response status
	isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 500

	return PingResultMsg{
		Success:  isSuccess,
		Duration: duration,
		Err:      nil,
	}
}

// handlePingResultViewKeys handles keyboard input in ping result view
// Requirements: 8.3, 8.4
func (m Model) handlePingResultViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "enter", "q":
		// Return to main view
		m.viewState = ViewMain
		m.testResult = nil
		m.testing = false
		return m, nil

	case "r":
		// Retry ping test
		if m.cursor >= 0 && m.cursor < len(m.configs) {
			cfg := m.configs[m.cursor]
			m.testing = true
			m.viewState = ViewPingTesting
			m.testResult = nil
			return m, pingConfig(&cfg)
		}
		return m, nil
	}

	return m, nil
}

// runCompatibilityTest creates a command to perform a compatibility test on a configuration
// Requirements: 9.1, 9.2, 9.3, 9.4
func runCompatibilityTest(cfg *config.APIConfig) tea.Cmd {
	return func() tea.Msg {
		tester, err := compatibility.NewTester(cfg)
		if err != nil {
			return CompatResultMsg{
				Result: nil,
				Err:    fmt.Errorf("创建测试器失败: %v", err),
			}
		}

		// Run full test including streaming
		result, err := tester.RunFullTest(true)
		if err != nil {
			return CompatResultMsg{
				Result: result,
				Err:    fmt.Errorf("测试执行失败: %v", err),
			}
		}

		return CompatResultMsg{
			Result: result,
			Err:    nil,
		}
	}
}

// handleCompatResultViewKeys handles keyboard input in compatibility result view
// Requirements: 9.3, 9.4
func (m Model) handleCompatResultViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc", "enter", "q":
		// Return to main view
		m.viewState = ViewMain
		m.compatResult = nil
		m.testing = false
		return m, nil

	case "r":
		// Retry compatibility test
		if m.cursor >= 0 && m.cursor < len(m.configs) {
			cfg := m.configs[m.cursor]
			m.testing = true
			m.viewState = ViewCompatTesting
			m.compatResult = nil
			return m, runCompatibilityTest(&cfg)
		}
		return m, nil
	}

	return m, nil
}
