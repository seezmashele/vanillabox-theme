package ui

import "charm.land/bubbles/v2/key"

// keyMap is every binding in the program. Bindings are enabled and disabled per
// screen in model.updateBindings, which means the help line at the bottom always
// describes exactly what the current screen accepts.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Toggle  key.Binding
	Confirm key.Binding
	Install key.Binding
	Back    key.Binding
	Restart key.Binding
	Quit    key.Binding

	// ForceQuit is always live, including mid-install. Bubble Tea does not quit
	// on ctrl+c by itself in raw mode, so without this the install screen would
	// trap the user.
	ForceQuit key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			// Bubble Tea normalises the space bar to "space"; a binding on " "
			// silently never matches.
			key.WithKeys("space"),
			key.WithHelp("space", "choose"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "continue"),
		),
		Install: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "install"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "start over"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
		),
	}
}

// ShortHelp implements help.KeyMap. Disabled bindings are skipped by the help
// bubble, so this can list everything.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Toggle,
		k.Confirm, k.Install, k.Restart, k.Back, k.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Toggle},
		{k.Confirm, k.Install, k.Restart},
		{k.Back, k.Quit},
	}
}
