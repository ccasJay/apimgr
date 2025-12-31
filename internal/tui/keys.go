package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard shortcuts
type KeyMap struct {
	Up           key.Binding // k - move up
	Down         key.Binding // j - move down
	Top          key.Binding // g - jump to top
	Bottom       key.Binding // G - jump to bottom
	Select       key.Binding // Enter - select
	SwitchLocal  key.Binding // s - switch local (Claude Code only)
	SwitchGlobal key.Binding // S - switch global active config
	Add          key.Binding // a - add config
	Edit         key.Binding // e - edit config
	Delete       key.Binding // d - delete config
	Ping         key.Binding // p - ping test
	Test         key.Binding // t - compatibility test
	Model        key.Binding // m - switch model
	Help         key.Binding // ? - help
	Quit         key.Binding // q - quit
	Cancel       key.Binding // Esc - cancel
	Confirm      key.Binding // Enter - confirm (in form)
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "向上"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "向下"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "跳到顶部"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "跳到底部"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "选择"),
		),
		SwitchLocal: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "本地切换"),
		),
		SwitchGlobal: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "全局切换"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "添加配置"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "编辑配置"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "删除配置"),
		),
		Ping: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "连接测试"),
		),
		Test: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "兼容性测试"),
		),
		Model: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "切换模型"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "帮助"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "退出"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "取消"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "确认"),
		),
	}
}

// ShortHelp returns short help text
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Select, k.SwitchLocal, k.SwitchGlobal, k.Quit}
}

// FullHelp returns full help text
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom},
		{k.Select, k.SwitchLocal, k.SwitchGlobal, k.Add},
		{k.Edit, k.Delete, k.Ping, k.Test},
		{k.Model, k.Help, k.Quit, k.Cancel},
	}
}
