package ui

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit        key.Binding
	Filter      key.Binding
	Tab         key.Binding
	ShiftTab    key.Binding
	Enter       key.Binding
	Back        key.Binding
	Snapshot    key.Binding
	Upgrade     key.Binding
	Remove      key.Binding
	Diff        key.Binding
	Export      key.Binding
	Rescan      key.Binding
	Edit        key.Binding
	Help        key.Binding
	PkgHelp     key.Binding
	Deps        key.Binding
	SizeFilter  key.Binding
	MultiSelect key.Binding
	Install     key.Binding
	Theme       key.Binding
	Space       key.Binding
	PreRelease  key.Binding
	Up          key.Binding
	Down        key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	HalfPageUp  key.Binding
	HalfPageDn  key.Binding
	Home        key.Binding
	End         key.Binding
}

var Keys = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "source"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev source"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "detail"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Snapshot: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "snap"),
	),
	Upgrade: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "upgrade"),
	),
	Remove: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "remove"),
	),
	Diff: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "diff"),
	),
	Export: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "export"),
	),
	Rescan: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "rescan"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit description"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	PkgHelp: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "help/usage"),
	),
	Deps: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "dependencies"),
	),
	SizeFilter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter"),
	),
	MultiSelect: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "select"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "search/install"),
	),
	Theme: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "theme"),
	),
	Space: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle"),
	),
	PreRelease: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pre-release"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdn", "page down"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "½ page up"),
	),
	HalfPageDn: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	),
	Home: key.NewBinding(
		key.WithKeys("home", "g"),
		key.WithHelp("home", "top"),
	),
	End: key.NewBinding(
		key.WithKeys("end", "G"),
		key.WithHelp("end", "bottom"),
	),
}
