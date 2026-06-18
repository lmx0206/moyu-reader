package ui

import (
	"strings"
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
	b := sampleBook()
	m := &Model{
		st:     store.New(t.TempDir()),
		lib:    store.NewLibrary(),
		width:  40,
		height: 12,
		screen: screenReader,
		book:   b,
		reader: NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12),
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
	// 普通键不退出
	nm, _ = m.Update(keyRunes("x"))
	m = nm.(*Model)
	if !m.bossActive {
		t.Fatal("ordinary key must NOT exit boss screen")
	}
	// 老板键退出
	nm, _ = m.Update(keyRunes("`"))
	m = nm.(*Model)
	if m.bossActive {
		t.Fatal("backtick again should exit boss screen")
	}
}

func TestModelSTogglesNav(t *testing.T) {
	m := newReaderModel(t)
	start := m.reader.Nav()
	nm, _ := m.Update(keyRunes("s"))
	m = nm.(*Model)
	if m.reader.Nav() == start {
		t.Fatalf("s should toggle nav mode, still %q", m.reader.Nav())
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

func TestModelGOpensTOC(t *testing.T) {
	m := newReaderModel(t)
	nm, _ := m.Update(keyRunes("g"))
	m = nm.(*Model)
	if m.screen != screenTOC || m.toc == nil {
		t.Fatalf("g should open TOC, screen=%v toc=%v", m.screen, m.toc)
	}
}

func TestModelTOCEnterJumps(t *testing.T) {
	m := newReaderModel(t)
	nm, _ := m.Update(keyRunes("g"))
	m = nm.(*Model)
	m.toc.MoveDown() // select chapter 1
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(*Model)
	if m.screen != screenReader {
		t.Fatalf("enter should return to reader, got %v", m.screen)
	}
	if m.reader.Progress().Chapter != 1 {
		t.Fatalf("should have jumped to chapter 1, got %+v", m.reader.Progress())
	}
}

func TestModelHelpKeyOpensAndCloses(t *testing.T) {
	m := newReaderModel(t)
	nm, _ := m.Update(keyRunes("?"))
	m = nm.(*Model)
	if m.screen != screenHelp {
		t.Fatalf("? should open help, got %v", m.screen)
	}
	if !contains(m.View(), "KEYBINDINGS") {
		t.Fatalf("help view should list keybindings:\n%s", m.View())
	}
	// any key closes back to reader
	nm, _ = m.Update(keyRunes("x"))
	m = nm.(*Model)
	if m.screen != screenReader {
		t.Fatalf("any key should close help back to reader, got %v", m.screen)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func TestModelAddAnnotation(t *testing.T) {
	m := newReaderModel(t)
	m.lib.Books = append(m.lib.Books, store.BookEntry{ID: "x"}) // bookID is "x"
	nm, _ := m.Update(keyRunes("a"))
	m = nm.(*Model)
	if m.screen != screenAnnotate {
		t.Fatalf("a should open annotate screen, got %v", m.screen)
	}
	nm, _ = m.Update(keyRunes("hi"))
	m = nm.(*Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(*Model)
	e := m.lib.FindByID("x")
	if e == nil || len(e.Annotations) != 1 || e.Annotations[0].Note != "hi" {
		t.Fatalf("annotation not saved: %+v", e)
	}
}

func TestModelAnnotationListJumpAndDelete(t *testing.T) {
	m := newReaderModel(t)
	m.lib.Books = append(m.lib.Books, store.BookEntry{
		ID:          "x",
		Annotations: []store.Annotation{{Chapter: 1, Para: 0, Note: "n"}},
	})
	nm, _ := m.Update(keyRunes("l"))
	m = nm.(*Model)
	if m.screen != screenAnnotList || m.annot == nil {
		t.Fatalf("l should open annotation list")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(*Model)
	if m.reader.Progress().Chapter != 1 {
		t.Fatalf("enter should jump to ch1, got %d", m.reader.Progress().Chapter)
	}
	nm, _ = m.Update(keyRunes("l"))
	m = nm.(*Model)
	nm, _ = m.Update(keyRunes("d"))
	m = nm.(*Model)
	if e := m.lib.FindByID("x"); e == nil || len(e.Annotations) != 0 {
		t.Fatalf("d should delete annotation: %+v", e)
	}
}
