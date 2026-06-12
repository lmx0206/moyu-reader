package ui

import (
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"moyureader/internal/disguise"
	"moyureader/internal/epub"
	"moyureader/internal/store"
)

type screen int

const (
	screenShelf screen = iota
	screenReader
	screenImport
	screenTOC
	screenHelp
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
	book   *epub.Book
	bookID string

	toc        *TOCView
	helpReturn screen

	bossActive bool
	bossTick   int

	importBuf string // typed path in import screen
	status    string // transient status line (errors etc.)
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
		m.status = "找不到这本书"
		return
	}
	book, err := epub.Parse(filepath.Join(m.st.Dir(), filepath.FromSlash(e.File)))
	if err != nil {
		e.Broken = true
		_ = m.st.Save(m.lib)
		m.status = "这本书打不开（已标记损坏）"
		return
	}
	m.reader = NewReaderView(book, e.Progress, e.Prefs, m.width, m.height)
	m.book = book
	m.bookID = id
	m.screen = screenReader
}

// saveProgress persists the current reader position + prefs.
func (m *Model) saveProgress() {
	if m.reader == nil || m.bookID == "" {
		return
	}
	store.UpdateProgress(m.lib, m.bookID, m.reader.Progress(), m.reader.Prefs())
	_ = m.st.Save(m.lib)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.reader != nil {
			m.reader.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case bossTickMsg:
		if m.bossActive {
			m.bossTick++
			return m, bossTick()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Boss screen swallows all keys: any key restores reading.
	if m.bossActive {
		m.bossActive = false
		return m, nil
	}

	key := msg.String()

	switch m.screen {
	case screenImport:
		return m.handleImportKey(msg)
	case screenHelp:
		m.screen = m.helpReturn // any key closes help
		return m, nil
	case screenTOC:
		return m.handleTOCKey(key)
	}

	// Help is available from shelf and reader.
	if key == "?" {
		m.helpReturn = m.screen
		m.screen = screenHelp
		return m, nil
	}

	// Global: backtick / b activates boss screen (only meaningful while reading).
	if (key == "`" || key == "b") && m.screen == screenReader {
		m.bossActive = true
		m.bossTick = 0
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
	}
	return m, nil
}

func (m *Model) deleteSelected() {
	sel := m.shelf.Selected()
	if sel == nil {
		return
	}
	id := sel.ID
	books := m.lib.Books[:0]
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
	switch key {
	case "q", "esc":
		m.saveProgress()
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
		m.reader.ToggleMode()
	case "g":
		if m.book != nil {
			m.toc = NewTOCView(m.book, m.reader.Progress().Chapter)
			m.screen = screenTOC
		}
	}
	return m, nil
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
		if msg.Type == tea.KeyRunes {
			m.importBuf += string(msg.Runes)
		} else if msg.Type == tea.KeySpace {
			m.importBuf += " "
		}
	}
	return m, nil
}

func (m *Model) doImport(path string) {
	if path == "" {
		return
	}
	book, err := epub.Parse(path)
	if err != nil {
		m.status = "解析失败: " + err.Error()
		return
	}
	if _, err := m.st.Import(m.lib, path, book.Title, book.Author); err != nil {
		m.status = "导入失败: " + err.Error()
		return
	}
	_ = m.st.Save(m.lib)
	m.shelf = NewShelfView(m.lib)
	m.screen = screenShelf
	m.status = "已导入: " + book.Title
}

func (m *Model) View() string {
	if m.bossActive {
		th := disguise.ThemeByName(m.readerStyle())
		return strings.Join(paintDim(disguise.BossScreen(th, m.bossTick, m.height)), "\n")
	}
	switch m.screen {
	case screenReader:
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
		return "导入 EPUB（粘贴 .epub 完整路径后回车，Esc 取消）:\n\n> " + m.importBuf + "\n\n" + m.status
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

// helpText returns the keybinding help, disguised as a CLI --help dump.
func helpText() []string {
	return []string{
		"reader - a tail-style log viewer (v0.2)",
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
		"  g    goto section            `/b      minimize",
		"  ?    help                    esc      back to list",
		"",
		"KEYBINDINGS (stream/CLI):",
		"  enter  next    b  back    t  switch profile    q  quit",
		"",
		"(press any key to dismiss)",
	}
}
