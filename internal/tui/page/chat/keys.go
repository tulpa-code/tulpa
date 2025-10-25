package chat

import (
	"github.com/charmbracelet/bubbles/v2/key"
)

type KeyMap struct {
	NewSession    key.Binding
	AddAttachment key.Binding
	Cancel        key.Binding
	Tab           key.Binding
	Details       key.Binding
	NextAgent     key.Binding
	PreviousAgent key.Binding
	Agent1        key.Binding
	Agent2        key.Binding
	Agent3        key.Binding
	Agent4        key.Binding
	Agent5        key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		NewSession: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new session"),
		),
		AddAttachment: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "add attachment"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc", "alt+esc"),
			key.WithHelp("esc", "cancel"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "change focus"),
		),
		Details: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "toggle details"),
		),
		NextAgent: key.NewBinding(
			key.WithKeys("ctrl+tab"),
			key.WithHelp("ctrl+tab", "next agent"),
		),
		PreviousAgent: key.NewBinding(
			key.WithKeys("ctrl+shift+tab"),
			key.WithHelp("ctrl+shift+tab", "previous agent"),
		),
		Agent1: key.NewBinding(
			key.WithKeys("ctrl+1"),
			key.WithHelp("ctrl+1", "agent 1"),
		),
		Agent2: key.NewBinding(
			key.WithKeys("ctrl+2"),
			key.WithHelp("ctrl+2", "agent 2"),
		),
		Agent3: key.NewBinding(
			key.WithKeys("ctrl+3"),
			key.WithHelp("ctrl+3", "agent 3"),
		),
		Agent4: key.NewBinding(
			key.WithKeys("ctrl+4"),
			key.WithHelp("ctrl+4", "agent 4"),
		),
		Agent5: key.NewBinding(
			key.WithKeys("ctrl+5"),
			key.WithHelp("ctrl+5", "agent 5"),
		),
	}
}
