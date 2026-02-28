package app

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	Quit      key.Binding
	Tab       key.Binding
	Enter     key.Binding
	Back      key.Binding
	Command   key.Binding
	Help      key.Binding
	Resume    key.Binding
	Stop      key.Binding
	Kill      key.Binding
	Run       key.Binding
	Refresh   key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch view"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Command: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "command"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Resume: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "resume"),
		),
		Stop: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "stop"),
		),
		Kill: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "kill"),
		),
		Run: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "run trigger"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "refresh"),
		),
	}
}
