package ui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"moyureader/internal/store"
)

// cmdMsgType runs a command and returns the concrete type of the message it
// produces, for comparing against known bubbletea commands (whose message
// types are unexported and so cannot be named directly).
func cmdMsgType(cmd tea.Cmd) reflect.Type {
	if cmd == nil {
		return nil
	}
	return reflect.TypeOf(cmd())
}

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

func TestModelCOpensStatsFromReader(t *testing.T) {
	m := newReaderModel(t)
	m.lib.Books = append(m.lib.Books, store.BookEntry{ID: "x"})
	nm, _ := m.Update(keyRunes("c"))
	m = nm.(*Model)
	if m.screen != screenStats {
		t.Fatalf("c should open stats, got %v", m.screen)
	}
	if !contains(m.View(), "Cover") {
		t.Fatalf("stats view should show coverage report:\n%s", m.View())
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(*Model)
	if m.screen != screenReader {
		t.Fatalf("esc should return to reader, got %v", m.screen)
	}
}

func TestModelRecordsCharsReadOnPageDown(t *testing.T) {
	m := newReaderModel(t)                              // sampleBook, bookID "x"
	m.book = sampleBook()                               // ensure book set
	m.lib.Books = append(m.lib.Books, store.BookEntry{ID: "x"})
	m.openBookForTest()                                 // sets TotalChars
	nm, _ := m.Update(keyRunes("f"))                    // page down -> saveProgress
	m = nm.(*Model)
	e := m.lib.FindByID("x")
	if e.TotalChars == 0 {
		t.Fatalf("TotalChars should be set on open")
	}
	if e.CharsRead <= 0 || e.FurthestPara == 0 {
		t.Fatalf("paging down should advance high-water: %+v", e)
	}
}

func TestModelMCyclesIntoRepl(t *testing.T) {
	m := newReaderModel(t) // starts shell
	nm, _ := m.Update(keyRunes("m"))
	m = nm.(*Model)
	if m.reader.Prefs().Mode != "inline" || m.repl != nil {
		t.Fatalf("first m should go shell->inline, got mode=%q repl=%v", m.reader.Prefs().Mode, m.repl)
	}
	nm, _ = m.Update(keyRunes("m"))
	m = nm.(*Model)
	if m.repl == nil {
		t.Fatalf("second m should enter repl")
	}
	// typing inside repl feeds the buffer, not the page
	nm, _ = m.Update(keyRunes("toc"))
	m = nm.(*Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(*Model)
	if !contains(m.View(), "第二章") {
		t.Fatalf("repl 'toc' should list chapters:\n%s", m.View())
	}
	// m again leaves repl back to shell reading
	nm, _ = m.Update(keyRunes("m"))
	m = nm.(*Model)
	if m.repl != nil || m.reader.Prefs().Mode != "shell" {
		t.Fatalf("m in repl should return to shell, repl=%v mode=%q", m.repl, m.reader.Prefs().Mode)
	}
}

func TestModelReplBossAndEsc(t *testing.T) {
	m := newReaderModel(t)
	m.repl = NewReplView(m.book, store.Progress{}, store.Prefs{Style: "log"}, 40, 12)
	// backtick still triggers boss even in repl
	nm, _ := m.Update(keyRunes("`"))
	m = nm.(*Model)
	if !m.bossActive {
		t.Fatalf("backtick should trigger boss in repl")
	}
	nm, _ = m.Update(keyRunes("`"))
	m = nm.(*Model)
	// esc in repl returns to shelf
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(*Model)
	if m.screen != screenShelf || m.repl != nil {
		t.Fatalf("esc in repl should go to shelf, screen=%v repl=%v", m.screen, m.repl)
	}
}

func TestModelReplPgUpScrolls(t *testing.T) {
	m := newReaderModel(t)
	m.height = 5
	m.repl = NewReplView(m.book, store.Progress{}, store.Prefs{Style: "log"}, 60, 5)
	for i := 0; i < 8; i++ { // generate scrollback past the window
		m.repl.input = "next"
		m.repl.Submit()
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = nm.(*Model)
	if m.repl.scrollOff == 0 {
		t.Fatalf("PgUp should scroll the repl up (scrollOff > 0)")
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = nm.(*Model)
	if m.repl.scrollOff != 0 {
		t.Fatalf("PgDown should scroll back to bottom, got %d", m.repl.scrollOff)
	}
}

func TestModelReplMouseWheelScrolls(t *testing.T) {
	m := newReaderModel(t)
	m.repl = NewReplView(m.book, store.Progress{}, store.Prefs{Style: "log"}, 60, 5)
	for i := 0; i < 8; i++ {
		m.repl.input = "next"
		m.repl.Submit()
	}
	nm, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	m = nm.(*Model)
	if m.repl.scrollOff == 0 {
		t.Fatalf("wheel up should scroll the repl up")
	}
	nm, _ = m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	m = nm.(*Model)
	if m.repl.scrollOff != 0 {
		t.Fatalf("wheel down should scroll back to bottom, got %d", m.repl.scrollOff)
	}
}

// Leaving the reader for an overlay must pause the reading clock, so the time
// spent browsing the overlay (and the gap until you return) is not counted as
// reading time.
func TestModelOpeningTOCPausesTiming(t *testing.T) {
	m := newReaderModel(t)
	m.lastActivity = time.Now() // pretend we were actively reading
	nm, _ := m.Update(keyRunes("g"))
	m = nm.(*Model)
	if m.screen != screenTOC {
		t.Fatalf("g should open TOC, got %v", m.screen)
	}
	if !m.lastActivity.IsZero() {
		t.Fatal("opening TOC should pause the reading clock (lastActivity zeroed)")
	}
}

func TestModelBossPausesTiming(t *testing.T) {
	m := newReaderModel(t)
	m.lastActivity = time.Now()
	nm, _ := m.Update(keyRunes("`"))
	m = nm.(*Model)
	if !m.bossActive {
		t.Fatal("backtick should activate boss")
	}
	if !m.lastActivity.IsZero() {
		t.Fatal("activating boss should pause the reading clock")
	}
}

func TestModelEnteringReplEnablesMouse(t *testing.T) {
	m := newReaderModel(t)            // shell
	nm, _ := m.Update(keyRunes("m"))  // shell -> inline
	m = nm.(*Model)
	nm, cmd := m.Update(keyRunes("m")) // inline -> repl
	m = nm.(*Model)
	if m.repl == nil {
		t.Fatal("second m should enter repl")
	}
	if cmdMsgType(cmd) != reflect.TypeOf(tea.EnableMouseCellMotion()) {
		t.Fatalf("entering repl should return EnableMouseCellMotion cmd, got %v", cmdMsgType(cmd))
	}
}

func TestModelLeavingReplDisablesMouse(t *testing.T) {
	m := newReaderModel(t)
	m.repl = NewReplView(m.book, store.Progress{}, store.Prefs{Style: "log"}, 40, 12)
	nm, cmd := m.Update(keyRunes("m")) // m in repl -> back to shell
	m = nm.(*Model)
	if m.repl != nil {
		t.Fatal("m in repl should leave repl")
	}
	if cmdMsgType(cmd) != reflect.TypeOf(tea.DisableMouse()) {
		t.Fatalf("leaving repl should return DisableMouse cmd, got %v", cmdMsgType(cmd))
	}
}

func TestModelReplEscDisablesMouse(t *testing.T) {
	m := newReaderModel(t)
	m.repl = NewReplView(m.book, store.Progress{}, store.Prefs{Style: "log"}, 40, 12)
	nm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(*Model)
	if m.screen != screenShelf || m.repl != nil {
		t.Fatalf("esc in repl should return to shelf, screen=%v repl=%v", m.screen, m.repl)
	}
	if cmdMsgType(cmd) != reflect.TypeOf(tea.DisableMouse()) {
		t.Fatalf("esc out of repl should disable the mouse, got %v", cmdMsgType(cmd))
	}
}
