package ui

import (
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"moyureader/internal/book"
	"moyureader/internal/disguise"
	"moyureader/internal/store"
	"moyureader/internal/version"
)

type screen int

const (
	screenShelf screen = iota
	screenReader
	screenImport
	screenTOC
	screenHelp
	screenAnnotate
	screenAnnotList
	screenStats
)

// bossTickMsg drives the boss-screen auto-scroll.
type bossTickMsg time.Time

func bossTick() tea.Cmd {
	return tea.Every(250*time.Millisecond, func(t time.Time) tea.Msg { return bossTickMsg(t) })
}

// Model is the root bubbletea model.
type Model struct {
	st  *store.Store
	lib *store.Library

	width, height int
	screen        screen

	shelf  *ShelfView
	reader *ReaderView
	repl   *ReplView
	book   *book.Book
	bookID string

	toc         *TOCView
	annot       *AnnotationView
	helpReturn  screen
	statsReturn screen

	bossActive bool
	bossTick   int

	importBuf    string    // typed path in import screen
	annotBuf     string    // typed note in annotate screen
	status       string    // transient status line (errors etc.)
	lastActivity time.Time // for accumulating reading time
}

// NewModel builds the root model. If openID is non-empty, it opens that book
// directly; otherwise it starts on the shelf.
func NewModel(st *store.Store, lib *store.Library, openID string) *Model {
	m := &Model{
		st:     st,
		lib:    lib,
		width:  80,
		height: 24,
		screen: screenShelf,
		shelf:  NewShelfView(lib),
	}
	if openID != "" {
		m.openBook(openID)
	}
	return m
}

func (m *Model) Init() tea.Cmd { return nil }

// openBook parses and enters the reader for the given book id.
func (m *Model) openBook(id string) {
	e := m.lib.FindByID(id)
	if e == nil {
		m.status = "fatal: pathspec did not match any object"
		return
	}
	bk, err := book.Open(filepath.Join(m.st.Dir(), filepath.FromSlash(e.File)))
	if err != nil {
		e.Broken = true
		_ = m.st.Save(m.lib)
		m.status = "error: object file could not be read (marked stale)"
		return
	}
	if e.TotalChars == 0 {
		e.TotalChars = book.TotalChars(bk)
	}
	m.reader = NewReaderView(bk, e.Progress, e.Prefs, m.width, m.height)
	m.book = bk
	m.bookID = id
	m.lastActivity = time.Time{}
	m.screen = screenReader
}

// saveProgress persists the current reader position + prefs and advances the
// per-book reading high-water mark.
func (m *Model) saveProgress() {
	if m.reader == nil || m.bookID == "" {
		return
	}
	p := m.reader.Progress()
	store.UpdateProgress(m.lib, m.bookID, p, m.reader.Prefs())
	if m.book != nil {
		store.RecordReading(m.lib, m.bookID, p.Chapter, p.Para, book.CharsUpTo(m.book, p.Chapter, p.Para))
	}
	_ = m.st.Save(m.lib)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.reader != nil {
			m.reader.SetSize(msg.Width, msg.Height)
		}
		if m.repl != nil {
			m.repl.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case bossTickMsg:
		if m.bossActive {
			m.bossTick++
			return m, bossTick()
		}
		return m, nil

	case tea.MouseMsg:
		// Mouse wheel scrolls the REPL scrollback; ignored elsewhere.
		if m.repl != nil && m.screen == screenReader && !m.bossActive {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.repl.ScrollUp(3)
			case tea.MouseButtonWheelDown:
				m.repl.ScrollDown(3)
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Boss screen swallows all keys; only the boss key itself restores reading,
	// so you can mash keys to look busy without revealing the novel.
	if m.bossActive {
		if key == "`" || key == "b" {
			m.bossActive = false
		}
		return m, nil
	}

	switch m.screen {
	case screenImport:
		return m.handleImportKey(msg)
	case screenHelp:
		m.screen = m.helpReturn // any key closes help
		return m, nil
	case screenTOC:
		return m.handleTOCKey(key)
	case screenAnnotate:
		return m.handleAnnotateKey(msg)
	case screenAnnotList:
		return m.handleAnnotListKey(key)
	case screenStats:
		return m.handleStatsKey(key)
	}

	// REPL captures typed input, so route it before the generic ?/boss/b checks.
	if m.screen == screenReader && m.repl != nil {
		return m.handleReplKey(msg)
	}

	// Help is available from shelf and reader.
	if key == "?" {
		m.helpReturn = m.screen
		m.screen = screenHelp
		m.pauseTiming()
		return m, nil
	}

	// Global: backtick / b activates boss screen (only meaningful while reading).
	if (key == "`" || key == "b") && m.screen == screenReader {
		m.bossActive = true
		m.bossTick = 0
		m.pauseTiming()
		return m, bossTick()
	}

	switch m.screen {
	case screenShelf:
		return m.handleShelfKey(key)
	case screenReader:
		return m.handleReaderKey(key)
	}
	return m, nil
}

func (m *Model) handleTOCKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.toc.MoveUp()
	case "down", "j":
		m.toc.MoveDown()
	case "enter":
		m.reader.JumpTo(m.toc.Selected())
		m.saveProgress()
		m.screen = screenReader
	case "esc", "q":
		m.screen = screenReader
	}
	return m, nil
}

func (m *Model) handleAnnotateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenReader
	case "enter":
		p := m.reader.Progress()
		store.AddAnnotation(m.lib, m.bookID, store.Annotation{
			Chapter:   p.Chapter,
			Para:      p.Para,
			Note:      strings.TrimSpace(m.annotBuf),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		_ = m.st.Save(m.lib)
		m.screen = screenReader
		m.status = "breakpoint set"
	case "backspace":
		if n := len(m.annotBuf); n > 0 {
			m.annotBuf = m.annotBuf[:n-1]
		}
	default:
		switch msg.Type {
		case tea.KeyRunes:
			m.annotBuf += string(msg.Runes)
		case tea.KeySpace:
			m.annotBuf += " "
		}
	}
	return m, nil
}

func (m *Model) handleAnnotListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		m.annot.MoveUp()
	case "down", "j":
		m.annot.MoveDown()
	case "enter":
		if a, ok := m.annot.Selected(); ok {
			m.reader.JumpToPara(a.Chapter, a.Para)
			m.saveProgress()
		}
		m.screen = screenReader
	case "d":
		if _, ok := m.annot.Selected(); ok {
			store.DeleteAnnotation(m.lib, m.bookID, m.annot.Index())
			_ = m.st.Save(m.lib)
			if e := m.lib.FindByID(m.bookID); e != nil {
				m.annot = NewAnnotationView(m.book, e.Annotations)
			}
		}
	case "esc", "q":
		m.screen = screenReader
	}
	return m, nil
}

func (m *Model) handleStatsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q", "c":
		m.screen = m.statsReturn
	}
	return m, nil
}

func (m *Model) handleShelfKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.shelf.MoveUp()
	case "down", "j":
		m.shelf.MoveDown()
	case "enter":
		if sel := m.shelf.Selected(); sel != nil {
			m.openBook(sel.ID)
		}
	case "i":
		m.screen = screenImport
		m.importBuf = ""
		m.status = ""
	case "d":
		m.deleteSelected()
	case "c":
		m.statsReturn = screenShelf
		m.screen = screenStats
	}
	return m, nil
}

func (m *Model) deleteSelected() {
	sel := m.shelf.Selected()
	if sel == nil {
		return
	}
	id := sel.ID
	books := make([]store.BookEntry, 0, len(m.lib.Books))
	for _, b := range m.lib.Books {
		if b.ID != id {
			books = append(books, b)
		}
	}
	m.lib.Books = books
	_ = m.st.Save(m.lib)
	m.shelf = NewShelfView(m.lib)
}

func (m *Model) handleReaderKey(key string) (tea.Model, tea.Cmd) {
	m.recordActivity()
	switch key {
	case "c":
		m.statsReturn = screenReader
		m.screen = screenStats
		m.pauseTiming()
	case "q", "esc":
		m.saveProgress()
		m.pauseTiming()
		m.screen = screenShelf
		m.shelf = NewShelfView(m.lib)
	case "ctrl+c":
		m.saveProgress()
		return m, tea.Quit
	case " ", "right", "pgdown", "f":
		m.reader.PageDown()
		m.saveProgress()
	case "left", "pgup", "B":
		m.reader.PageUp()
		m.saveProgress()
	case "down", "j":
		m.reader.LineDown()
	case "up", "k":
		m.reader.LineUp()
	case "tab":
		m.reader.CycleStyle()
	case "m":
		return m, m.cycleMode()
	case "s":
		m.reader.ToggleNav()
	case "a":
		m.annotBuf = ""
		m.screen = screenAnnotate
		m.pauseTiming()
	case "l":
		if e := m.lib.FindByID(m.bookID); e != nil && m.book != nil {
			m.annot = NewAnnotationView(m.book, e.Annotations)
			m.screen = screenAnnotList
			m.pauseTiming()
		}
	case "g":
		if m.book != nil {
			m.toc = NewTOCView(m.book, m.reader.Progress().Chapter)
			m.screen = screenTOC
			m.pauseTiming()
		}
	}
	return m, nil
}

// cycleMode advances the reading presentation shell -> inline -> repl. (The
// repl -> shell leg is handled inside handleReplKey, since the 'm' key is
// intercepted there while the REPL is active.) Entering the REPL returns a
// command enabling the mouse so the wheel can scroll the scrollback.
func (m *Model) cycleMode() tea.Cmd {
	if m.book == nil {
		m.reader.ToggleMode()
		return nil
	}
	if m.reader.Prefs().Mode == "inline" {
		m.repl = NewReplView(m.book, m.reader.Progress(), m.reader.Prefs(), m.width, m.height)
		return tea.EnableMouseCellMotion
	}
	m.reader.ToggleMode() // shell -> inline
	return nil
}

func (m *Model) handleReplKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.recordActivity()
	switch msg.String() {
	case "`":
		m.bossActive = true
		m.bossTick = 0
		m.pauseTiming()
		return m, bossTick()
	case "m":
		// leave repl, resume shell-mode reading at the same paragraph
		p := m.repl.Progress()
		m.reader.JumpToPara(p.Chapter, p.Para)
		if m.reader.Prefs().Mode != "shell" {
			m.reader.ToggleMode() // inline -> shell
		}
		m.repl = nil
		m.saveProgress()
		return m, tea.DisableMouse
	case "esc":
		m.replExitToShelf()
		return m, tea.DisableMouse
	case "ctrl+c":
		m.replSyncProgress()
		m.saveProgress()
		return m, tea.Quit // quitting restores the terminal (mouse off)
	case "enter":
		m.repl.Submit()
		m.replSyncProgress()
		m.saveProgress()
		if m.repl.quit {
			m.replExitToShelf()
			return m, tea.DisableMouse
		}
	case "backspace":
		m.repl.Backspace()
	case "up":
		m.repl.HistoryPrev()
	case "down":
		m.repl.HistoryNext()
	case "pgup":
		m.repl.ScrollUp(m.replScrollPage())
	case "pgdown":
		m.repl.ScrollDown(m.replScrollPage())
	default:
		if msg.Type == tea.KeyRunes {
			m.repl.Insert(string(msg.Runes))
		} else if msg.Type == tea.KeySpace {
			m.repl.Insert(" ")
		}
	}
	return m, nil
}

// replScrollPage is how many lines PgUp/PgDn scroll the REPL scrollback (about
// one screenful, leaving the prompt and a line of overlap).
func (m *Model) replScrollPage() int {
	if p := m.height - 2; p > 1 {
		return p
	}
	return 1
}

// replSyncProgress copies the repl position into the reader so saveProgress and
// later modes see the latest position.
func (m *Model) replSyncProgress() {
	if m.repl != nil && m.reader != nil {
		p := m.repl.Progress()
		m.reader.JumpToPara(p.Chapter, p.Para)
	}
}

func (m *Model) replExitToShelf() {
	m.replSyncProgress()
	m.saveProgress()
	m.repl = nil
	m.screen = screenShelf
	m.shelf = NewShelfView(m.lib)
}

func (m *Model) handleImportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenShelf
	case "enter":
		m.doImport(strings.TrimSpace(strings.Trim(m.importBuf, `"`)))
	case "backspace":
		if n := len(m.importBuf); n > 0 {
			m.importBuf = m.importBuf[:n-1]
		}
	default:
		switch msg.Type {
		case tea.KeyRunes:
			m.importBuf += string(msg.Runes)
		case tea.KeySpace:
			m.importBuf += " "
		}
	}
	return m, nil
}

func (m *Model) doImport(path string) {
	if path == "" {
		return
	}
	bk, err := book.Open(path)
	if err != nil {
		m.status = "error: parse failed: " + err.Error()
		return
	}
	entry, err := m.st.Import(m.lib, path, bk.Title, bk.Author)
	if err != nil {
		m.status = "error: " + err.Error()
		return
	}
	entry.TotalChars = book.TotalChars(bk) // seed now so unopened books show real coverage
	_ = m.st.Save(m.lib)
	m.shelf = NewShelfView(m.lib)
	m.screen = screenShelf
	m.status = "+ " + bk.Title
}

func (m *Model) View() string {
	if m.bossActive {
		th := disguise.ThemeByName(m.readerStyle())
		return strings.Join(paintDim(disguise.BossScreen(th, m.bossTick, m.height)), "\n")
	}
	switch m.screen {
	case screenReader:
		if m.repl != nil {
			return strings.Join(paintDim(m.repl.Render()), "\n")
		}
		lines := m.reader.Render()
		if m.reader.Prefs().Mode == "shell" {
			return strings.Join(paintShell(lines), "\n")
		}
		return strings.Join(paintDim(lines), "\n")
	case screenTOC:
		return strings.Join(paintDim(m.toc.Render(m.width, m.height)), "\n")
	case screenHelp:
		return strings.Join(paintDim(helpText()), "\n")
	case screenImport:
		return "fetch> " + m.importBuf + "\n(paste a path, Enter to fetch, Esc to cancel)\n\n" + m.status
	case screenStats:
		return strings.Join(paintDim((StatsView{}).Render(m.lib, m.width, m.height)), "\n")
	case screenAnnotList:
		return strings.Join(paintDim(m.annot.Render(m.width, m.height)), "\n")
	case screenAnnotate:
		return "break> " + m.annotBuf + "\n(condition optional, Enter to set breakpoint, Esc to cancel)"
	default:
		body := m.shelf.Render(m.width, m.height-1)
		if m.status != "" {
			body = append(body, "", m.status)
		}
		return strings.Join(body, "\n")
	}
}

func (m *Model) readerStyle() string {
	if m.reader != nil {
		return m.reader.Prefs().Style
	}
	return m.lib.Global.Style
}

// idleCap bounds the gap between two reading actions that still counts as
// reading: dwelling on a page longer than this (or walking away) is not added.
const idleCap = 2 * time.Minute

// recordActivity accumulates reading time, attributing the gap since the last
// reading action (capped at idleCap) and rolling the daily streak.
func (m *Model) recordActivity() {
	now := time.Now()
	secs := 0
	if !m.lastActivity.IsZero() {
		if d := now.Sub(m.lastActivity); d > 0 && d <= idleCap {
			secs = int(d.Seconds())
		}
	}
	store.RecordActivity(m.lib, now, secs)
	m.lastActivity = now
}

// pauseTiming stops the reading clock when leaving the reading view for an
// overlay (TOC/stats/help/annotate), the boss screen, or the shelf, so the time
// spent away — and the gap until reading resumes — is not counted as reading.
func (m *Model) pauseTiming() { m.lastActivity = time.Time{} }

// openBookForTest seeds TotalChars from the in-memory book without a real file
// (used by tests; mirrors the TotalChars seeding that openBook does).
func (m *Model) openBookForTest() {
	if e := m.lib.FindByID(m.bookID); e != nil && m.book != nil && e.TotalChars == 0 {
		e.TotalChars = book.TotalChars(m.book)
	}
}

// helpText returns the keybinding help, disguised as a CLI --help dump.
func helpText() []string {
	return []string{
		"reader - a tail-style log viewer (v" + version.Version + ")",
		"",
		"USAGE:",
		"  reader [command]",
		"",
		"KEYBINDINGS (shelf):",
		"  up/down  select    enter  open    i  import    d  delete    q  quit",
		"",
		"KEYBINDINGS (reader):",
		"  space/→/pgdn  next page      up/down  scroll line",
		"  tab  switch profile          m        toggle view",
		"  s    scroll/page mode        g        goto section",
		"  a    add bookmark/note       l        list bookmarks",
		"  `/b  minimize (same key restores)     ?  help",
		"  esc  back to list            q        quit",
		"",
		"KEYBINDINGS (stream/CLI):",
		"  enter  next    b  back    t  switch profile    q  quit",
		"",
		"(press any key to dismiss)",
	}
}
