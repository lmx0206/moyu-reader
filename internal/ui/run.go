package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"moyureader/internal/store"
)

// Run launches the full-screen TUI. If openID is non-empty, it opens that book
// directly; otherwise it starts on the shelf.
func Run(st *store.Store, lib *store.Library, openID string) error {
	m := NewModel(st, lib, openID)
	// Mouse capture is enabled only while in the REPL reading mode (so the wheel
	// can scroll the fake-shell scrollback) and disabled on the way out, so it
	// never interferes with normal terminal text selection / copy elsewhere.
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
