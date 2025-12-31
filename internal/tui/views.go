package tui

import (
	"fmt"
	"strings"

	"apimgr/config"

	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	activeSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Background(lipgloss.Color("57")).
				Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))
)

// RenderMainView renders the main list view
// Requirements: 11.1, 11.2, 11.3
func (m Model) RenderMainView() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("API 配置管理器"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", m.getEffectiveWidth(40))))
	b.WriteString("\n\n")

	// Config list with scrolling
	if len(m.configs) == 0 {
		b.WriteString(dimStyle.Render("暂无配置，按 'a' 添加新配置"))
		b.WriteString("\n")
	} else {
		visibleHeight := m.getVisibleListHeight()
		startIdx := m.scrollOffset
		endIdx := startIdx + visibleHeight
		if endIdx > len(m.configs) {
			endIdx = len(m.configs)
		}

		// Show scroll indicator at top if scrolled down
		if startIdx > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ 还有 %d 项...", startIdx)))
			b.WriteString("\n")
		}

		// Render visible configs
		for i := startIdx; i < endIdx; i++ {
			cfg := m.configs[i]
			line := m.renderConfigLine(i, cfg)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Show scroll indicator at bottom if more items below
		if endIdx < len(m.configs) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ 还有 %d 项...", len(m.configs)-endIdx)))
			b.WriteString("\n")
		}
	}

	// Add some spacing before status bar
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", m.getEffectiveWidth(40))))
	b.WriteString("\n")

	// Status bar
	b.WriteString(m.RenderStatusBar())

	return b.String()
}

// getEffectiveWidth returns the effective width for rendering, with a minimum and maximum
// Requirements: 11.2
func (m Model) getEffectiveWidth(defaultWidth int) int {
	if m.width <= 0 {
		return defaultWidth
	}
	// Use window width but cap at a reasonable maximum for readability
	maxWidth := 80
	if m.width < maxWidth {
		return m.width - 2 // Leave some margin
	}
	return maxWidth
}

// renderConfigLine renders a single config line in the list
func (m Model) renderConfigLine(index int, cfg config.APIConfig) string {
	isSelected := index == m.cursor
	isActive := cfg.Alias == m.activeAlias

	// Build cursor indicator
	cursor := "  "
	if isSelected {
		cursor = "> "
	}

	// Build active indicator
	activeMarker := "  "
	if isActive {
		activeMarker = "* "
	}

	// Build the main line content
	alias := cfg.Alias
	
	// Add model info if available
	modelInfo := ""
	if cfg.Model != "" {
		modelInfo = fmt.Sprintf(" [%s]", cfg.Model)
	}

	// Add base URL info (truncated if too long)
	urlInfo := ""
	if cfg.BaseURL != "" {
		url := cfg.BaseURL
		if len(url) > 30 {
			url = url[:27] + "..."
		}
		urlInfo = fmt.Sprintf(" (%s)", url)
	}

	// Combine all parts
	content := fmt.Sprintf("%s%s%s%s%s", cursor, activeMarker, alias, modelInfo, urlInfo)

	// Apply appropriate style based on selection and active state
	if isSelected && isActive {
		return activeSelectedStyle.Render(content)
	} else if isSelected {
		return selectedStyle.Render(content)
	} else if isActive {
		return activeStyle.Render(content)
	}
	return normalStyle.Render(content)
}

// Detail view styles
var (
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Width(12)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	detailActiveTagStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Background(lipgloss.Color("22")).
				Bold(true).
				Padding(0, 1)

	detailSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	detailMaskedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))
)

// RenderDetailView renders the detail view
// Requirements: 3.1, 3.2, 3.3, 3.4, 11.2
func (m Model) RenderDetailView() string {
	var b strings.Builder

	if m.selected < 0 || m.selected >= len(m.configs) {
		return dimStyle.Render("未选择配置，按 Enter 选择一个配置查看详情")
	}

	cfg := m.configs[m.selected]
	effectiveWidth := m.getEffectiveWidth(40)

	// Title with active status indicator
	b.WriteString(titleStyle.Render("配置详情"))
	if cfg.Alias == m.activeAlias {
		b.WriteString("  ")
		b.WriteString(detailActiveTagStyle.Render("★ 活跃"))
	}
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n\n")

	// Basic information section
	b.WriteString(detailSectionStyle.Render("基本信息"))
	b.WriteString("\n")

	// Alias
	b.WriteString(detailLabelStyle.Render("Alias:"))
	b.WriteString(detailValueStyle.Render(m.truncateText(cfg.Alias, effectiveWidth-14)))
	b.WriteString("\n")

	// Provider (if set)
	if cfg.Provider != "" {
		b.WriteString(detailLabelStyle.Render("Provider:"))
		b.WriteString(detailValueStyle.Render(m.truncateText(cfg.Provider, effectiveWidth-14)))
		b.WriteString("\n")
	}

	// Base URL
	b.WriteString(detailLabelStyle.Render("Base URL:"))
	if cfg.BaseURL != "" {
		b.WriteString(detailValueStyle.Render(m.truncateText(cfg.BaseURL, effectiveWidth-14)))
	} else {
		b.WriteString(dimStyle.Render("(默认)"))
	}
	b.WriteString("\n")

	b.WriteString("\n")

	// Model information section
	b.WriteString(detailSectionStyle.Render("模型配置"))
	b.WriteString("\n")

	// Current active model
	b.WriteString(detailLabelStyle.Render("当前模型:"))
	if cfg.Model != "" {
		b.WriteString(detailValueStyle.Render(m.truncateText(cfg.Model, effectiveWidth-14)))
	} else {
		b.WriteString(dimStyle.Render("(未设置)"))
	}
	b.WriteString("\n")

	// Supported models list
	b.WriteString(detailLabelStyle.Render("模型列表:"))
	if len(cfg.Models) > 0 {
		modelsStr := strings.Join(cfg.Models, ", ")
		b.WriteString(detailValueStyle.Render(m.truncateText(modelsStr, effectiveWidth-14)))
	} else {
		b.WriteString(dimStyle.Render("(无)"))
	}
	b.WriteString("\n")

	b.WriteString("\n")

	// Authentication section (masked sensitive info)
	b.WriteString(detailSectionStyle.Render("认证信息"))
	b.WriteString("\n")

	// API Key (masked)
	b.WriteString(detailLabelStyle.Render("API Key:"))
	if cfg.APIKey != "" {
		b.WriteString(detailMaskedStyle.Render(maskString(cfg.APIKey)))
	} else {
		b.WriteString(dimStyle.Render("(未设置)"))
	}
	b.WriteString("\n")

	// Auth Token (masked)
	b.WriteString(detailLabelStyle.Render("Auth Token:"))
	if cfg.AuthToken != "" {
		b.WriteString(detailMaskedStyle.Render(maskString(cfg.AuthToken)))
	} else {
		b.WriteString(dimStyle.Render("(未设置)"))
	}
	b.WriteString("\n")

	// Footer with available actions
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("s: 本地切换 │ S: 全局切换 │ e: 编辑 │ d: 删除 │ p: 测试 │ Esc: 返回"))

	return b.String()
}

// truncateText truncates text to fit within maxWidth, adding ellipsis if needed
// Requirements: 11.2
func (m Model) truncateText(text string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}
	if len(text) <= maxWidth {
		return text
	}
	return text[:maxWidth-3] + "..."
}

// RenderFormView renders the form view (add/edit)
func (m Model) RenderFormView() string {
	var b strings.Builder

	title := "添加配置"
	if m.viewState == ViewEdit {
		title = "编辑配置"
	}

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Form inputs will be rendered here
	b.WriteString("表单输入区域\n")

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter: 确认 | Esc: 取消"))

	return b.String()
}

// RenderDeleteConfirm renders the delete confirmation dialog
// Requirements: 7.1, 7.2, 11.2
func (m Model) RenderDeleteConfirm() string {
	var b strings.Builder
	effectiveWidth := m.getEffectiveWidth(40)

	b.WriteString(titleStyle.Render("确认删除"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n\n")

	if m.cursor >= 0 && m.cursor < len(m.configs) {
		cfg := m.configs[m.cursor]
		
		// Warning message
		b.WriteString(errorStyle.Render("⚠ 警告: 此操作不可撤销！"))
		b.WriteString("\n\n")
		
		// Config info to be deleted
		b.WriteString(normalStyle.Render("即将删除配置: "))
		b.WriteString(selectedStyle.Render(cfg.Alias))
		b.WriteString("\n\n")
		
		// Show if this is the active config
		if cfg.Alias == m.activeAlias {
			b.WriteString(errorStyle.Render("注意: 这是当前活跃的配置！"))
			b.WriteString("\n\n")
		}
		
		// Show config details
		if cfg.BaseURL != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("Base URL: %s", m.truncateText(cfg.BaseURL, effectiveWidth-12))))
			b.WriteString("\n")
		}
		if cfg.Model != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("Model: %s", m.truncateText(cfg.Model, effectiveWidth-8))))
			b.WriteString("\n")
		}
	} else {
		b.WriteString(errorStyle.Render("错误: 未选择有效的配置"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("y: 确认删除 │ n/Esc: 取消"))

	return b.String()
}

// RenderHelpView renders the help panel with scrolling support
// Requirements: 10.2, 10.3, 10.4, 11.2
func (m Model) RenderHelpView() string {
	var b strings.Builder
	effectiveWidth := m.getEffectiveWidth(50)

	// Title
	b.WriteString(titleStyle.Render("快捷键帮助"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n")

	// Build all help content lines
	helpLines := m.buildHelpLines()

	// Calculate visible range with scrolling
	visibleHeight := m.getVisibleHelpHeight()
	startIdx := m.helpScrollOffset
	endIdx := startIdx + visibleHeight
	if endIdx > len(helpLines) {
		endIdx = len(helpLines)
	}

	// Show scroll indicator at top if scrolled down
	if startIdx > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ 还有 %d 行...", startIdx)))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// Render visible lines
	for i := startIdx; i < endIdx; i++ {
		b.WriteString(helpLines[i])
	}

	// Show scroll indicator at bottom if more content below
	if endIdx < len(helpLines) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ 还有 %d 行...", len(helpLines)-endIdx)))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: 上下滚动 │ q/Esc: 返回"))

	return b.String()
}

// buildHelpLines builds all help content lines for scrolling
func (m Model) buildHelpLines() []string {
	var lines []string

	// Navigation section
	lines = append(lines, detailSectionStyle.Render("导航")+"\n")
	lines = append(lines, renderHelpLine("j / ↓", "向下移动光标"))
	lines = append(lines, renderHelpLine("k / ↑", "向上移动光标"))
	lines = append(lines, renderHelpLine("g", "跳转到列表顶部"))
	lines = append(lines, renderHelpLine("G", "跳转到列表底部"))
	lines = append(lines, renderHelpLine("Enter", "选择/查看配置详情"))
	lines = append(lines, "\n")

	// Config management section
	lines = append(lines, detailSectionStyle.Render("配置管理")+"\n")
	lines = append(lines, renderHelpLine("s", "本地切换 (仅当前终端)"))
	lines = append(lines, renderHelpLine("S", "全局切换 (设为活跃配置)"))
	lines = append(lines, renderHelpLine("a", "添加新配置"))
	lines = append(lines, renderHelpLine("e", "编辑当前配置"))
	lines = append(lines, renderHelpLine("d", "删除当前配置"))
	lines = append(lines, "\n")

	// Model management section
	lines = append(lines, detailSectionStyle.Render("模型管理")+"\n")
	lines = append(lines, renderHelpLine("m", "切换模型"))
	lines = append(lines, "\n")

	// Testing section
	lines = append(lines, detailSectionStyle.Render("测试")+"\n")
	lines = append(lines, renderHelpLine("p", "连接测试 (Ping)"))
	lines = append(lines, renderHelpLine("t", "API 兼容性测试"))
	lines = append(lines, "\n")

	// General section
	lines = append(lines, detailSectionStyle.Render("通用")+"\n")
	lines = append(lines, renderHelpLine("?", "显示此帮助面板"))
	lines = append(lines, renderHelpLine("Esc", "返回/取消"))
	lines = append(lines, renderHelpLine("q", "退出程序"))
	lines = append(lines, "\n")

	return lines
}

// renderHelpLine renders a single help line with key and description
func renderHelpLine(key, desc string) string {
	keyStyled := helpKeyStyle.Render(fmt.Sprintf("  %-10s", key))
	descStyled := normalStyle.Render(desc)
	return fmt.Sprintf("%s %s\n", keyStyled, descStyled)
}

// RenderStatusBar renders the bottom status bar
func (m Model) RenderStatusBar() string {
	var b strings.Builder

	// Error message (displayed prominently)
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("✗ 错误: " + m.errorMsg))
		b.WriteString("\n")
	}

	// Status message (success/info messages)
	if m.message != "" {
		b.WriteString(messageStyle.Render("✓ " + m.message))
		b.WriteString("\n")
	}

	// Add spacing if there were messages
	if m.errorMsg != "" || m.message != "" {
		b.WriteString("\n")
	}

	// Shortcut hints - formatted nicely
	keys := DefaultKeyMap()
	shortHelp := keys.ShortHelp()
	hints := make([]string, 0, len(shortHelp))
	for _, k := range shortHelp {
		keyStr := helpKeyStyle.Render(k.Help().Key)
		descStr := helpStyle.Render(k.Help().Desc)
		hints = append(hints, fmt.Sprintf("%s %s", keyStr, descStr))
	}
	b.WriteString(strings.Join(hints, helpStyle.Render(" │ ")))

	return b.String()
}

// maskString masks sensitive string, showing only first and last few characters
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// RenderModelSelectView renders the model selection view
// Requirements: 12.1, 12.2, 11.3
func (m Model) RenderModelSelectView() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("选择模型"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", m.getEffectiveWidth(40))))
	b.WriteString("\n\n")

	// Show current config info
	if m.cursor >= 0 && m.cursor < len(m.configs) {
		cfg := m.configs[m.cursor]
		b.WriteString(dimStyle.Render(fmt.Sprintf("配置: %s", cfg.Alias)))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(fmt.Sprintf("当前模型: %s", cfg.Model)))
		b.WriteString("\n\n")
	}

	// Model list with scrolling
	if len(m.modelList) == 0 {
		b.WriteString(dimStyle.Render("没有可用的模型"))
		b.WriteString("\n")
	} else {
		// Get current active model for marking
		var activeModel string
		if m.cursor >= 0 && m.cursor < len(m.configs) {
			activeModel = m.configs[m.cursor].Model
		}

		visibleHeight := m.getVisibleModelListHeight()
		startIdx := m.modelScrollOffset
		endIdx := startIdx + visibleHeight
		if endIdx > len(m.modelList) {
			endIdx = len(m.modelList)
		}

		// Show scroll indicator at top if scrolled down
		if startIdx > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ 还有 %d 项...", startIdx)))
			b.WriteString("\n")
		}

		// Render visible models
		for i := startIdx; i < endIdx; i++ {
			model := m.modelList[i]
			line := m.renderModelLine(i, model, activeModel)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Show scroll indicator at bottom if more items below
		if endIdx < len(m.modelList) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ 还有 %d 项...", len(m.modelList)-endIdx)))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", m.getEffectiveWidth(40))))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: 上下移动 │ Enter: 确认选择 │ Esc: 取消"))

	return b.String()
}

// renderModelLine renders a single model line in the selection list
// Requirements: 12.2
func (m Model) renderModelLine(index int, model string, activeModel string) string {
	isSelected := index == m.modelCursor
	isActive := model == activeModel

	// Build cursor indicator
	cursor := "  "
	if isSelected {
		cursor = "> "
	}

	// Build active indicator
	activeMarker := "  "
	if isActive {
		activeMarker = "* "
	}

	// Combine all parts
	content := fmt.Sprintf("%s%s%s", cursor, activeMarker, model)

	// Apply appropriate style based on selection and active state
	if isSelected && isActive {
		return activeSelectedStyle.Render(content)
	} else if isSelected {
		return selectedStyle.Render(content)
	} else if isActive {
		return activeStyle.Render(content)
	}
	return normalStyle.Render(content)
}

// RenderPingTestingView renders the ping testing in progress view
// Requirements: 8.2, 11.2
func (m Model) RenderPingTestingView() string {
	var b strings.Builder
	effectiveWidth := m.getEffectiveWidth(40)

	// Title
	b.WriteString(titleStyle.Render("连接测试"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n\n")

	// Show which config is being tested
	if m.cursor >= 0 && m.cursor < len(m.configs) {
		cfg := m.configs[m.cursor]
		b.WriteString(dimStyle.Render(fmt.Sprintf("配置: %s", cfg.Alias)))
		b.WriteString("\n")
		if cfg.BaseURL != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("URL: %s", m.truncateText(cfg.BaseURL, effectiveWidth-6))))
		} else {
			b.WriteString(dimStyle.Render("URL: https://api.anthropic.com (默认)"))
		}
		b.WriteString("\n\n")
	}

	// Testing indicator
	b.WriteString(messageStyle.Render("⏳ 正在测试连接..."))
	b.WriteString("\n")

	return b.String()
}

// RenderPingResultView renders the ping test result view
// Requirements: 8.3, 8.4, 11.2
func (m Model) RenderPingResultView() string {
	var b strings.Builder
	effectiveWidth := m.getEffectiveWidth(40)

	// Title
	b.WriteString(titleStyle.Render("连接测试结果"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n\n")

	// Show which config was tested
	if m.cursor >= 0 && m.cursor < len(m.configs) {
		cfg := m.configs[m.cursor]
		b.WriteString(dimStyle.Render(fmt.Sprintf("配置: %s", cfg.Alias)))
		b.WriteString("\n")
		if cfg.BaseURL != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("URL: %s", m.truncateText(cfg.BaseURL, effectiveWidth-6))))
		} else {
			b.WriteString(dimStyle.Render("URL: https://api.anthropic.com (默认)"))
		}
		b.WriteString("\n\n")
	}

	// Show result
	if m.testResult != nil {
		if m.testResult.Success {
			b.WriteString(messageStyle.Render("✅ 连接成功!"))
			b.WriteString("\n\n")
			if m.testResult.Duration != "" {
				b.WriteString(normalStyle.Render(fmt.Sprintf("响应时间: %s", m.testResult.Duration)))
				b.WriteString("\n")
			}
		} else {
			b.WriteString(errorStyle.Render("❌ 连接失败"))
			b.WriteString("\n\n")
			if m.testResult.Message != "" {
				b.WriteString(errorStyle.Render(fmt.Sprintf("错误: %s", m.truncateText(m.testResult.Message, effectiveWidth-6))))
				b.WriteString("\n")
			}
			if m.testResult.Duration != "" {
				b.WriteString(dimStyle.Render(fmt.Sprintf("耗时: %s", m.testResult.Duration)))
				b.WriteString("\n")
			}
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("r: 重试 │ Enter/Esc: 返回"))

	return b.String()
}

// RenderCompatTestingView renders the compatibility testing in progress view
// Requirements: 9.2, 11.2
func (m Model) RenderCompatTestingView() string {
	var b strings.Builder
	effectiveWidth := m.getEffectiveWidth(50)

	// Title
	b.WriteString(titleStyle.Render("API 兼容性测试"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n\n")

	// Show which config is being tested
	if m.cursor >= 0 && m.cursor < len(m.configs) {
		cfg := m.configs[m.cursor]
		b.WriteString(dimStyle.Render(fmt.Sprintf("配置: %s", cfg.Alias)))
		b.WriteString("\n")
		if cfg.BaseURL != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("URL: %s", m.truncateText(cfg.BaseURL, effectiveWidth-6))))
		} else {
			b.WriteString(dimStyle.Render("URL: https://api.anthropic.com (默认)"))
		}
		b.WriteString("\n")
		if cfg.Model != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("模型: %s", cfg.Model)))
		}
		b.WriteString("\n\n")
	}

	// Testing indicator
	b.WriteString(messageStyle.Render("⏳ 正在执行兼容性测试..."))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("测试内容包括:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • 连接测试"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • 认证验证"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • 响应格式检查"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  • 流式响应测试"))
	b.WriteString("\n")

	return b.String()
}

// Compatibility level styles
var (
	compatFullStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	compatPartialStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	compatNoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	checkPassedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))

	checkFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	checkCriticalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
)

// RenderCompatResultView renders the compatibility test result view
// Requirements: 9.3, 9.4, 11.2
func (m Model) RenderCompatResultView() string {
	var b strings.Builder
	effectiveWidth := m.getEffectiveWidth(50)

	// Title
	b.WriteString(titleStyle.Render("API 兼容性测试结果"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n\n")

	// Show which config was tested
	if m.cursor >= 0 && m.cursor < len(m.configs) {
		cfg := m.configs[m.cursor]
		b.WriteString(dimStyle.Render(fmt.Sprintf("配置: %s", cfg.Alias)))
		b.WriteString("\n")
		if cfg.BaseURL != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("URL: %s", m.truncateText(cfg.BaseURL, effectiveWidth-6))))
		} else {
			b.WriteString(dimStyle.Render("URL: https://api.anthropic.com (默认)"))
		}
		b.WriteString("\n\n")
	}

	// Show result
	if m.compatResult != nil {
		// Compatibility level
		b.WriteString(detailSectionStyle.Render("兼容性级别"))
		b.WriteString("\n")
		switch m.compatResult.CompatibilityLevel {
		case "full":
			b.WriteString(compatFullStyle.Render("✅ 完全兼容"))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("此配置与 Claude Code 完全兼容"))
		case "partial":
			b.WriteString(compatPartialStyle.Render("⚠️ 部分兼容"))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("此配置可能存在一些兼容性问题"))
		case "none":
			b.WriteString(compatNoneStyle.Render("❌ 不兼容"))
			b.WriteString("\n")
			b.WriteString(dimStyle.Render("此配置与 Claude Code 不兼容"))
		default:
			b.WriteString(dimStyle.Render("未知"))
		}
		b.WriteString("\n\n")

		// Response time
		if m.compatResult.ResponseTime != "" {
			b.WriteString(dimStyle.Render(fmt.Sprintf("响应时间: %s", m.compatResult.ResponseTime)))
			b.WriteString("\n\n")
		}

		// Detailed checks
		if len(m.compatResult.Checks) > 0 {
			b.WriteString(detailSectionStyle.Render("详细检查结果"))
			b.WriteString("\n")
			for _, check := range m.compatResult.Checks {
				var icon string
				var style lipgloss.Style
				if check.Passed {
					icon = "✓"
					style = checkPassedStyle
				} else if check.Critical {
					icon = "✗"
					style = checkCriticalStyle
				} else {
					icon = "!"
					style = checkFailedStyle
				}

				// Check name and status
				checkLine := fmt.Sprintf("  %s %s", icon, check.Name)
				b.WriteString(style.Render(checkLine))
				b.WriteString("\n")

				// Check message (indented)
				if check.Message != "" {
					msgLine := fmt.Sprintf("    %s", m.truncateText(check.Message, effectiveWidth-6))
					b.WriteString(dimStyle.Render(msgLine))
					b.WriteString("\n")
				}
			}
		}

		// Error message if any
		if m.compatResult.Error != "" {
			b.WriteString("\n")
			b.WriteString(errorStyle.Render(fmt.Sprintf("错误: %s", m.truncateText(m.compatResult.Error, effectiveWidth-6))))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render(strings.Repeat("─", effectiveWidth)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("r: 重试 │ Enter/Esc: 返回"))

	return b.String()
}
