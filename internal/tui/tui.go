package tui

import (
	"fmt"
	"os"

	"apimgr/config"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the TUI interface
func Run() error {
	// Check if we're running in a terminal
	if !isTerminal() {
		return fmt.Errorf("apimgr TUI requires a terminal. Use subcommands for non-interactive mode")
	}

	configManager, err := config.NewConfigManager()
	if err != nil {
		return err
	}

	m := NewModel(configManager)
	
	// Create program with options that work better across different terminals
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
	}
	
	// Add input/output options for better compatibility
	if os.Getenv("TERM") != "" {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	
	p := tea.NewProgram(m, opts...)

	_, err = p.Run()
	return err
}

// isTerminal checks if stdin is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
