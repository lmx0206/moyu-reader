package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"moyureader/internal/store"
)

// Run launches the full-screen TUI. If openID is non-empty, it opens that book
// directly; otherwise it starts on the shelf.
func Run(st *store.Store, lib *store.Library, openID string) error {
	m := NewModel(st, lib, openID)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
