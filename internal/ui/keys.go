package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Open       key.Binding
	Back       key.Binding
	Quit       key.Binding
	Refresh    key.Binding
	Toggle     key.Binding
	Up         key.Binding
	Down       key.Binding
	HalfUp     key.Binding
	HalfDown   key.Binding
	Settings   key.Binding
	Add        key.Binding
	Delete     key.Binding
	Filter     key.Binding
	Help       key.Binding
	Comments   key.Binding
	ExpandBody key.Binding
}

var keys = keyMap{
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open"),
	),
	Comments: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "comments link"),
	),
	ExpandBody: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "expand post body"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "toggle fold"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	HalfUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("c-u", "half page up"),
	),
	HalfDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("c-d", "half page down"),
	),
	Settings: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "settings"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d", "x"),
		key.WithHelp("d", "delete"),
	),
	Filter: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "filter source"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}
