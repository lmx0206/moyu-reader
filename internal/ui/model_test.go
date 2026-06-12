package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"moyureader/internal/store"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// newReaderModel returns a Model already in the reader screen for a sample book.
func newReaderModel(t *testing.T) *Model {
	t.Helper()
	m := &Model{
		st:     store.New(t.TempDir()),
		lib:    store.NewLibrary(),
		width:  40,
		height: 12,
		screen: screenReader,
		reader: NewReaderView(sampleBook(), store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12),
		bookID: "x",
	}
	return m
}

func TestModelBossKeyToggles(t *testing.T) {
	m := newReaderModel(t)
	if m.bossActive {
		t.Fatal("boss should start inactive")
	}
	nm, _ := m.Update(keyRunes("`"))
	m = nm.(*Model)
	if !m.bossActive {
		t.Fatal("backtick should activate boss screen")
	}
	// 老板键激活时，任意键恢复
	nm, _ = m.Update(keyRunes("x"))
	m = nm.(*Model)
	if m.bossActive {
		t.Fatal("any key should deactivate boss screen")
	}
}

func TestModelToggleModeRoutesToReader(t *testing.T) {
	m := newReaderModel(t)
	nm, _ := m.Update(keyRunes("m"))
	m = nm.(*Model)
	if m.reader.Prefs().Mode != "inline" {
		t.Fatalf("m should toggle reader mode, got %q", m.reader.Prefs().Mode)
	}
}

func TestModelViewBossHidesNovel(t *testing.T) {
	m := newReaderModel(t)
	m.bossActive = true
	out := m.View()
	if out == "" {
		t.Fatal("boss view should not be empty")
	}
}

func TestModelWindowSizeUpdatesReader(t *testing.T) {
	m := newReaderModel(t)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = nm.(*Model)
	if m.width != 80 || m.height != 24 {
		t.Fatalf("window size not applied: %dx%d", m.width, m.height)
	}
}
