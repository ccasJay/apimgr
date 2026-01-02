package tui

import (
	"time"

	"apimgr/config/models"
	"apimgr/internal/compatibility"
)

// ConfigsLoadedMsg is sent when configs are loaded
type ConfigsLoadedMsg struct {
	Configs     []models.APIConfig
	ActiveAlias string
}

// ConfigSwitchedMsg is sent when active config is switched
type ConfigSwitchedMsg struct {
	Alias   string
	IsLocal bool // true for local switch, false for global switch
	Err     error
}

// ConfigAddedMsg is sent when a config is added
type ConfigAddedMsg struct {
	Config models.APIConfig
	Err    error
}

// ConfigUpdatedMsg is sent when a config is updated
type ConfigUpdatedMsg struct {
	Alias string
	Err   error
}

// ConfigDeletedMsg is sent when a config is deleted
type ConfigDeletedMsg struct {
	Alias string
	Err   error
}

// PingResultMsg is sent when ping test completes
type PingResultMsg struct {
	Success  bool
	Duration time.Duration
	Err      error
}

// CompatResultMsg is sent when compatibility test completes
type CompatResultMsg struct {
	Result *compatibility.TestResult
	Err    error
}

// ModelSwitchedMsg is sent when model is switched
type ModelSwitchedMsg struct {
	Alias string
	Model string
	Err   error
}
