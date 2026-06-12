# 摸鱼阅读器 · 计划二：前端（TUI + 内联流式 CLI + main）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在计划一核心引擎之上，构建可运行的两种前端：全屏 TUI（书架/阅读/老板键）与内联流式 CLI，并用 `cmd/reader/main.go` 装配命令行入口，产出单文件 exe。

**Architecture:** 真正的业务逻辑放进**纯可测单元**：`internal/ui` 的 `ReaderView`（阅读位置/分页/伪装组合）与 `ShelfView`（书架列表），`internal/stream` 的 `Streamer`（内联分段），`cmd/reader` 的 `parseArgs`/`resolveDataDir`。Bubbletea 的 `Model` 与 `main` 只做薄胶水。配色用 lipgloss 在 `View` 层叠加，不进可测逻辑（保持 golden 稳定）。

**Tech Stack:** Go、`github.com/charmbracelet/bubbletea`（v1，**非** v2 `charm.land/*` beta）、`github.com/charmbracelet/lipgloss`（v1）。复用计划一的 `internal/{epub,render,disguise,store}`。**不引入 bubbles**——书架列表与导入输入手写，规避 v1/v2 textinput 路径问题。

**运行 Go：** 本机 Go 在 `D:\develop\Go`，未必在 shell PATH 中。命令前加 `export PATH="/d/develop/Go/bin:$PATH"`。

---

## 模块文件结构

```
internal/ui/reader.go        ReaderView：位置/分页/伪装组合（纯逻辑）
internal/ui/reader_test.go
internal/ui/shelf.go         ShelfView：书架列表排序/光标（纯逻辑）
internal/ui/shelf_test.go
internal/ui/model.go         bubbletea Model：屏幕状态机 + 按键路由 + 老板键 tick（胶水）
internal/ui/model_test.go
internal/ui/style.go         lipgloss 配色（仅 View 层装饰）
internal/ui/run.go           Run(store, openID)：构造 Program 并运行（alt screen）
internal/stream/stream.go    Streamer + Run：内联流式输出
internal/stream/stream_test.go
cmd/reader/args.go           parseArgs / resolveDataDir（纯逻辑）
cmd/reader/args_test.go
cmd/reader/main.go           入口装配
README.md                    使用说明
```

---

## 依赖前置

- [ ] **Step 0: 添加 bubbletea / lipgloss（v1）**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
```
Expected: 二者出现在 go.mod，且版本是 v1.x（不是 v2 / charm.land）。验证：
```bash
go list -m github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss
```
Expected: 形如 `github.com/charmbracelet/bubbletea v1.3.x`。

> 若 `@latest` 解析到 v2（模块路径变为 `charm.land/...`），改用显式 v1：
> `go get github.com/charmbracelet/bubbletea@v1.3.4 && go get github.com/charmbracelet/lipgloss@v1.1.0`

---

## Task 1: ReaderView 阅读逻辑

**Files:**
- Create: `internal/ui/reader.go`
- Test: `internal/ui/reader_test.go`

ReaderView 把 `epub.Book` + 进度 + 偏好 + 终端尺寸，组合成"恰好 height 行"的伪装画面，并提供翻页/换行/切风格/切模式。

- [ ] **Step 1: 写失败测试**

Create `internal/ui/reader_test.go`:
```go
package ui

import (
	"strings"
	"testing"

	"moyureader/internal/epub"
	"moyureader/internal/store"
)

func sampleBook() *epub.Book {
	mk := func(title string, n int) epub.Chapter {
		ps := []string{title}
		for i := 0; i < n; i++ {
			ps = append(ps, strings.Repeat("文", 10)) // 每段 10 个双宽字
		}
		return epub.Chapter{Title: title, Paragraphs: ps}
	}
	return &epub.Book{
		Title:    "样书",
		Author:   "佚名",
		Chapters: []epub.Chapter{mk("第一章", 20), mk("第二章", 5)},
	}
}

func TestReaderRenderExactHeightShellMode(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	lines := r.Render()
	if len(lines) != 12 {
		t.Fatalf("shell render must be exactly height(12), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "tail -f") {
		t.Fatalf("shell header missing, line0=%q", lines[0])
	}
	if !strings.Contains(lines[11], "watching") {
		t.Fatalf("shell footer missing, last=%q", lines[11])
	}
}

func TestReaderRenderExactHeightInlineMode(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "inline"}, 40, 12)
	lines := r.Render()
	if len(lines) != 12 {
		t.Fatalf("inline render must be height(12), got %d", len(lines))
	}
	// inline 模式：含正文的行应带日志前缀
	found := false
	for _, l := range lines {
		if strings.Contains(l, " - ") && strings.Contains(l, "文") {
			found = true
		}
	}
	if !found {
		t.Fatal("inline mode should prefix novel lines with log decoration")
	}
}

func TestReaderPageDownCrossesChapter(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "inline"}, 40, 6)
	start := r.Progress()
	if start.Chapter != 0 || start.Line != 0 {
		t.Fatalf("bad start %+v", start)
	}
	// 反复 PageDown 必须最终进入第 2 章
	reached := false
	for i := 0; i < 100; i++ {
		r.PageDown()
		if r.Progress().Chapter == 1 {
			reached = true
			break
		}
	}
	if !reached {
		t.Fatal("PageDown never reached chapter 1")
	}
}

func TestReaderToggleModeAndCycleStyle(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 10)
	r.ToggleMode()
	if r.Prefs().Mode != "inline" {
		t.Fatalf("toggle to inline failed: %q", r.Prefs().Mode)
	}
	r.ToggleMode()
	if r.Prefs().Mode != "shell" {
		t.Fatalf("toggle back to shell failed: %q", r.Prefs().Mode)
	}
	r.CycleStyle()
	if r.Prefs().Style != "build" {
		t.Fatalf("cycle style log->build failed: %q", r.Prefs().Style)
	}
}

func TestReaderProgressClampOnConstruct(t *testing.T) {
	b := sampleBook()
	// 越界进度应被夹紧，不 panic
	r := NewReaderView(b, store.Progress{Chapter: 99, Line: 99}, store.Prefs{Style: "log", Mode: "shell"}, 40, 10)
	if r.Progress().Chapter < 0 || r.Progress().Chapter >= len(b.Chapters) {
		t.Fatalf("chapter not clamped: %+v", r.Progress())
	}
	_ = r.Render() // 不得 panic
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestReader`
Expected: FAIL（`NewReaderView` undefined）

- [ ] **Step 3: 实现 ReaderView**

Create `internal/ui/reader.go`:
```go
// Package ui implements the full-screen terminal reader.
package ui

import (
	"fmt"

	"moyureader/internal/disguise"
	"moyureader/internal/epub"
	"moyureader/internal/render"
	"moyureader/internal/store"
)

// ReaderView holds reading position and renders the disguised page.
type ReaderView struct {
	book    *epub.Book
	chapter int
	line    int // top display-line index within current chapter
	width   int
	height  int
	style   string
	mode    string
}

// NewReaderView builds a reader at the given progress/prefs, clamped to valid
// bounds. width/height are terminal dimensions.
func NewReaderView(b *epub.Book, p store.Progress, prefs store.Prefs, width, height int) *ReaderView {
	r := &ReaderView{
		book:   b,
		style:  orDefault(prefs.Style, "log"),
		mode:   orDefault(prefs.Mode, "shell"),
		width:  width,
		height: height,
	}
	r.chapter = clamp(p.Chapter, 0, len(b.Chapters)-1)
	r.line = clampLine(p.Line, r.chapterLineCount())
	return r
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampLine(v, count int) int {
	if count <= 0 {
		return 0
	}
	return clamp(v, 0, count-1)
}

// SetSize updates terminal dimensions and re-clamps the line index.
func (r *ReaderView) SetSize(width, height int) {
	r.width, r.height = width, height
	r.line = clampLine(r.line, r.chapterLineCount())
}

// contentWidth is the wrap width for novel body.
func (r *ReaderView) contentWidth() int {
	if r.width < 10 {
		return 10
	}
	return r.width
}

// bodyHeight is the number of novel-body lines per page (chrome excluded).
func (r *ReaderView) bodyHeight() int {
	h := r.height
	if r.mode == "shell" {
		h -= 2 // header + footer
	}
	if h < 1 {
		return 1
	}
	return h
}

func (r *ReaderView) chapterLines() []string {
	if r.chapter < 0 || r.chapter >= len(r.book.Chapters) {
		return []string{""}
	}
	return render.LayoutChapter(r.book.Chapters[r.chapter].Paragraphs, r.contentWidth())
}

func (r *ReaderView) chapterLineCount() int { return len(r.chapterLines()) }

// Render returns exactly r.height display lines, fully disguised.
func (r *ReaderView) Render() []string {
	lines := r.chapterLines()
	bh := r.bodyHeight()
	end := r.line + bh
	if end > len(lines) {
		end = len(lines)
	}
	var page []string
	if r.line < len(lines) {
		page = append(page, lines[r.line:end]...)
	}
	for len(page) < bh { // pad to full body height
		page = append(page, "")
	}

	th := disguise.ThemeByName(r.style)
	if r.mode == "inline" {
		return disguise.RenderInline(th, page, r.width)
	}
	return disguise.RenderShell(th, page, r.width, r.StatusText())
}

// StatusText is the chrome status, e.g. "ch.1/2 · 0%".
func (r *ReaderView) StatusText() string {
	total := len(r.book.Chapters)
	pct := 0
	if total > 0 {
		pct = r.chapter * 100 / total
	}
	return fmt.Sprintf("ch.%d/%d · %d%%", r.chapter+1, total, pct)
}

func lastPageStart(n, h int) int {
	if n <= 0 || h <= 0 {
		return 0
	}
	return ((n - 1) / h) * h
}

// PageDown advances one page, crossing into the next chapter at the end.
func (r *ReaderView) PageDown() {
	lines := r.chapterLines()
	bh := r.bodyHeight()
	if r.line+bh < len(lines) {
		r.line += bh
		return
	}
	if r.chapter < len(r.book.Chapters)-1 {
		r.chapter++
		r.line = 0
	}
}

// PageUp goes back one page, crossing into the previous chapter at the top.
func (r *ReaderView) PageUp() {
	bh := r.bodyHeight()
	if r.line-bh >= 0 {
		r.line -= bh
		return
	}
	if r.line > 0 {
		r.line = 0
		return
	}
	if r.chapter > 0 {
		r.chapter--
		r.line = lastPageStart(r.chapterLineCount(), bh)
	}
}

// LineDown / LineUp scroll a single line, crossing chapters at the edges.
func (r *ReaderView) LineDown() {
	maxTop := len(r.chapterLines()) - r.bodyHeight()
	if maxTop < 0 {
		maxTop = 0
	}
	if r.line < maxTop {
		r.line++
		return
	}
	if r.chapter < len(r.book.Chapters)-1 {
		r.chapter++
		r.line = 0
	}
}

func (r *ReaderView) LineUp() {
	if r.line > 0 {
		r.line--
		return
	}
	if r.chapter > 0 {
		r.chapter--
		r.line = lastPageStart(r.chapterLineCount(), r.bodyHeight())
	}
}

// CycleStyle advances log->build->git->log.
func (r *ReaderView) CycleStyle() { r.style = disguise.NextStyle(r.style) }

// ToggleMode flips shell<->inline.
func (r *ReaderView) ToggleMode() {
	if r.mode == "shell" {
		r.mode = "inline"
	} else {
		r.mode = "shell"
	}
}

// Progress returns the current stable position.
func (r *ReaderView) Progress() store.Progress {
	return store.Progress{Chapter: r.chapter, Line: r.line}
}

// Prefs returns current style/mode.
func (r *ReaderView) Prefs() store.Prefs {
	return store.Prefs{Style: r.style, Mode: r.mode}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestReader`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ui/reader.go internal/ui/reader_test.go
git commit -m "feat(ui): ReaderView position, pagination and disguised render"
```

---

## Task 2: ShelfView 书架逻辑

**Files:**
- Create: `internal/ui/shelf.go`
- Test: `internal/ui/shelf_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/ui/shelf_test.go`:
```go
package ui

import (
	"testing"

	"moyureader/internal/store"
)

func TestShelfOrdersLastReadFirst(t *testing.T) {
	lib := &store.Library{
		LastBookID: "b",
		Books: []store.BookEntry{
			{ID: "a", Title: "A", LastOpenedAt: "2026-01-01T00:00:00Z"},
			{ID: "b", Title: "B", LastOpenedAt: "2025-01-01T00:00:00Z"},
			{ID: "c", Title: "C", LastOpenedAt: "2026-06-01T00:00:00Z"},
		},
	}
	s := NewShelfView(lib)
	if s.Selected().ID != "b" {
		t.Fatalf("last-read book should be first/selected, got %q", s.Selected().ID)
	}
}

func TestShelfMoveClamps(t *testing.T) {
	lib := &store.Library{Books: []store.BookEntry{{ID: "a"}, {ID: "b"}}}
	s := NewShelfView(lib)
	s.MoveUp() // already at top, stays
	if s.cursor != 0 {
		t.Fatalf("cursor should stay 0, got %d", s.cursor)
	}
	s.MoveDown()
	s.MoveDown() // clamp at last
	if s.cursor != 1 {
		t.Fatalf("cursor should clamp at 1, got %d", s.cursor)
	}
}

func TestShelfSelectedEmpty(t *testing.T) {
	s := NewShelfView(&store.Library{})
	if s.Selected() != nil {
		t.Fatal("empty shelf Selected() must be nil")
	}
	r := s.Render(40, 10)
	if len(r) == 0 {
		t.Fatal("empty shelf should still render a hint")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestShelf`
Expected: FAIL（`NewShelfView` undefined）

- [ ] **Step 3: 实现 ShelfView**

Create `internal/ui/shelf.go`:
```go
package ui

import (
	"fmt"
	"sort"

	"moyureader/internal/store"
)

// ShelfView is the bookshelf list with a cursor.
type ShelfView struct {
	items  []store.BookEntry
	cursor int
}

// NewShelfView builds a shelf ordered so the last-read book is first, then by
// most-recently-opened.
func NewShelfView(lib *store.Library) *ShelfView {
	items := make([]store.BookEntry, len(lib.Books))
	copy(items, lib.Books)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ID == lib.LastBookID {
			return true
		}
		if items[j].ID == lib.LastBookID {
			return false
		}
		return items[i].LastOpenedAt > items[j].LastOpenedAt
	})
	return &ShelfView{items: items}
}

// MoveUp / MoveDown move the cursor with clamping.
func (s *ShelfView) MoveUp() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s *ShelfView) MoveDown() {
	if s.cursor < len(s.items)-1 {
		s.cursor++
	}
}

// Selected returns the highlighted entry, or nil if the shelf is empty.
func (s *ShelfView) Selected() *store.BookEntry {
	if len(s.items) == 0 {
		return nil
	}
	return &s.items[s.cursor]
}

// Render returns up to height lines listing the books.
func (s *ShelfView) Render(width, height int) []string {
	if len(s.items) == 0 {
		return []string{
			"书架空空如也。",
			"按 i 导入一本 .epub，或用命令行: reader <某本书.epub>",
		}
	}
	var out []string
	out = append(out, "📚 书架   ↑↓ 选择 · Enter 阅读 · i 导入 · d 删除 · q 退出")
	out = append(out, "")
	for i, e := range s.items {
		cursor := "  "
		if i == s.cursor {
			cursor = "> "
		}
		broken := ""
		if e.Broken {
			broken = " [损坏]"
		}
		line := fmt.Sprintf("%s%s — %s%s", cursor, e.Title, e.Author, broken)
		out = append(out, line)
		if len(out) >= height {
			break
		}
	}
	return out
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestShelf`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ui/shelf.go internal/ui/shelf_test.go
git commit -m "feat(ui): ShelfView bookshelf list with last-read ordering"
```

---

## Task 3: bubbletea Model（屏幕状态机 + 按键 + 老板键）

**Files:**
- Create: `internal/ui/model.go`
- Create: `internal/ui/style.go`
- Test: `internal/ui/model_test.go`

Model 是薄胶水：持有 store、library、当前屏幕、ShelfView/ReaderView、老板键状态与 tick。打开书时用 `epub.Parse` 解析。进度在翻页与退出时存盘。

- [ ] **Step 1: 写失败测试（用合成按键驱动 Update）**

Create `internal/ui/model_test.go`:
```go
package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"moyureader/internal/epub"
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

var _ = epub.Book{} // keep epub import referenced if unused above
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestModel`
Expected: FAIL（`Model` / `screenReader` undefined）

- [ ] **Step 3: 实现 style.go（lipgloss 配色，仅装饰）**

Create `internal/ui/style.go`:
```go
package ui

import "github.com/charmbracelet/lipgloss"

// dimStyle colors chrome/prefix lines so the disguise looks like a real tool.
var (
	chromeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	bodyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// paintShell colors header (line0) and footer (last) dimmer than the body.
func paintShell(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if i == 0 || i == len(lines)-1 {
			out[i] = chromeStyle.Render(l)
		} else {
			out[i] = bodyStyle.Render(l)
		}
	}
	return out
}

// paintDim colors every line as chrome (used for inline + boss screens).
func paintDim(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = chromeStyle.Render(l)
	}
	return out
}
```

- [ ] **Step 4: 实现 model.go**

Create `internal/ui/model.go`:
```go
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
	bookID string

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

	if m.screen == screenImport {
		return m.handleImportKey(msg)
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
```

- [ ] **Step 5: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestModel`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/ui/model.go internal/ui/style.go internal/ui/model_test.go
git commit -m "feat(ui): bubbletea model with screens, keys and boss-key"
```

---

## Task 4: TUI 运行入口 run.go

**Files:**
- Create: `internal/ui/run.go`

run.go 只是把 Model 装进 alt-screen Program 并运行——无独立单元测试（由 main 的端到端构建验证）。

- [ ] **Step 1: 实现 run.go**

Create `internal/ui/run.go`:
```go
package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"moyureader/internal/store"
)

// Run launches the full-screen TUI. If openID is non-empty, it opens that book
// directly; otherwise it starts on the shelf.
func Run(st *store.Store, lib *store.Library, openID string) error {
	m := NewModel(st, lib, openID)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
```

- [ ] **Step 2: 编译确认**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go build ./internal/ui/`
Expected: 无错误。

- [ ] **Step 3: 提交**

```bash
git add internal/ui/run.go
git commit -m "feat(ui): alt-screen program runner"
```

---

## Task 5: 内联流式 CLI（Streamer + Run）

**Files:**
- Create: `internal/stream/stream.go`
- Test: `internal/stream/stream_test.go`

内联模式：不接管屏幕，逐段把伪装行打印到 stdout，回车出下一段。固定 wrap 宽度 80。

- [ ] **Step 1: 写失败测试**

Create `internal/stream/stream_test.go`:
```go
package stream

import (
	"bytes"
	"strings"
	"testing"

	"moyureader/internal/epub"
	"moyureader/internal/store"
)

func sampleBook() *epub.Book {
	mk := func(title string, n int) epub.Chapter {
		ps := []string{title}
		for i := 0; i < n; i++ {
			ps = append(ps, strings.Repeat("文", 20))
		}
		return epub.Chapter{Title: title, Paragraphs: ps}
	}
	return &epub.Book{Title: "样书", Chapters: []epub.Chapter{mk("第一章", 30)}}
}

func TestStreamerNextAdvances(t *testing.T) {
	s := NewStreamer(sampleBook(), store.Progress{}, "log", 5)
	first := s.Next()
	if len(first) != 5 {
		t.Fatalf("chunk should be height(5) lines, got %d", len(first))
	}
	p1 := s.Progress()
	_ = s.Next()
	p2 := s.Progress()
	if p2.Line <= p1.Line && p2.Chapter == p1.Chapter {
		t.Fatalf("position should advance: %+v -> %+v", p1, p2)
	}
	// inline 行应带日志前缀
	if !strings.Contains(first[0], " - ") {
		t.Fatalf("inline stream line should have log prefix: %q", first[0])
	}
}

func TestStreamRunEnterAndQuit(t *testing.T) {
	s := NewStreamer(sampleBook(), store.Progress{}, "log", 3)
	in := strings.NewReader("\n\nq\n") // two enters then quit
	var out bytes.Buffer
	var savedCalled bool
	Run(s, in, &out, func(p store.Progress, style string) { savedCalled = true })
	if !savedCalled {
		t.Fatal("Run should call onExit to save progress")
	}
	if out.Len() == 0 {
		t.Fatal("Run should print chunks")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/stream/`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现 stream.go**

Create `internal/stream/stream.go`:
```go
// Package stream implements the inline (non-fullscreen) streaming reader.
package stream

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"moyureader/internal/disguise"
	"moyureader/internal/epub"
	"moyureader/internal/render"
	"moyureader/internal/store"
)

const wrapWidth = 80

// Streamer emits successive inline-disguised chunks of a book.
type Streamer struct {
	book    *epub.Book
	chapter int
	line    int
	style   string
	height  int
}

// NewStreamer starts at the given progress.
func NewStreamer(b *epub.Book, p store.Progress, style string, height int) *Streamer {
	if height < 1 {
		height = 1
	}
	s := &Streamer{book: b, style: orDefault(style, "log"), height: height}
	if p.Chapter >= 0 && p.Chapter < len(b.Chapters) {
		s.chapter = p.Chapter
	}
	s.line = p.Line
	return s
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (s *Streamer) chapterLines() []string {
	if s.chapter < 0 || s.chapter >= len(s.book.Chapters) {
		return nil
	}
	return render.LayoutChapter(s.book.Chapters[s.chapter].Paragraphs, wrapWidth)
}

// Next returns the next chunk of inline-disguised lines and advances position.
func (s *Streamer) Next() []string {
	lines := s.chapterLines()
	if s.line >= len(lines) {
		if s.chapter < len(s.book.Chapters)-1 {
			s.chapter++
			s.line = 0
			lines = s.chapterLines()
		}
	}
	end := s.line + s.height
	if end > len(lines) {
		end = len(lines)
	}
	var page []string
	if s.line < len(lines) {
		page = lines[s.line:end]
	}
	s.line = end
	th := disguise.ThemeByName(s.style)
	return disguise.RenderInline(th, page, 0)
}

// Back rewinds roughly one chunk.
func (s *Streamer) Back() {
	s.line -= s.height
	if s.line < 0 {
		if s.chapter > 0 {
			s.chapter--
			s.line = 0
		} else {
			s.line = 0
		}
	}
}

// CycleStyle advances the disguise style.
func (s *Streamer) CycleStyle() { s.style = disguise.NextStyle(s.style) }

// Progress / Style expose current state for persistence.
func (s *Streamer) Progress() store.Progress {
	return store.Progress{Chapter: s.chapter, Line: s.line}
}
func (s *Streamer) Style() string { return s.style }

// Run drives the streamer: print a chunk, then read a command line from in.
// Empty line = next; "q" = quit; "b" = back; "t" = cycle style. onExit is
// called with the final progress + style so the caller can persist it.
func Run(s *Streamer, in io.Reader, out io.Writer, onExit func(store.Progress, string)) {
	sc := bufio.NewScanner(in)
	emit := func() {
		for _, l := range s.Next() {
			fmt.Fprintln(out, l)
		}
	}
	emit()
	for sc.Scan() {
		switch strings.TrimSpace(sc.Text()) {
		case "q", "quit", "exit":
			onExit(s.Progress(), s.Style())
			return
		case "b":
			s.Back()
			emit()
		case "t":
			s.CycleStyle()
			emit()
		default:
			emit()
		}
	}
	onExit(s.Progress(), s.Style())
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/stream/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/stream
git commit -m "feat(stream): inline streaming reader with Enter-to-continue"
```

---

## Task 6: 命令行参数与数据目录（纯逻辑）

**Files:**
- Create: `cmd/reader/args.go`
- Test: `cmd/reader/args_test.go`

- [ ] **Step 1: 写失败测试**

Create `cmd/reader/args_test.go`:
```go
package main

import (
	"path/filepath"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		in   []string
		mode string
		arg  string
	}{
		{[]string{}, "tui", ""},
		{[]string{"list"}, "list", ""},
		{[]string{"stream"}, "stream", ""},
		{[]string{"stream", "abc"}, "stream", "abc"},
		{[]string{"import", "x.epub"}, "import", "x.epub"},
		{[]string{"book.epub"}, "open", "book.epub"},
		{[]string{"C:\\x\\My Book.epub"}, "open", "C:\\x\\My Book.epub"},
	}
	for _, c := range cases {
		cmd := parseArgs(c.in)
		if cmd.Mode != c.mode || cmd.Arg != c.arg {
			t.Fatalf("parseArgs(%v) = {%q,%q}, want {%q,%q}", c.in, cmd.Mode, cmd.Arg, c.mode, c.arg)
		}
	}
}

func TestResolveDataDirEnvOverride(t *testing.T) {
	got := resolveDataDir("D:\\develop\\reader.exe", "D:\\mydata")
	if got != "D:\\mydata" {
		t.Fatalf("env override should win, got %q", got)
	}
}

func TestResolveDataDirExeAdjacent(t *testing.T) {
	got := resolveDataDir(filepath.Join("D:\\app", "reader.exe"), "")
	want := filepath.Join("D:\\app", "data")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./cmd/reader/`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现 args.go**

Create `cmd/reader/args.go`:
```go
package main

import (
	"path/filepath"
	"strings"
)

// command is the parsed CLI intent.
type command struct {
	Mode string // tui | open | import | stream | list
	Arg  string
}

// parseArgs interprets os.Args[1:].
func parseArgs(args []string) command {
	if len(args) == 0 {
		return command{Mode: "tui"}
	}
	switch args[0] {
	case "list":
		return command{Mode: "list"}
	case "stream":
		if len(args) > 1 {
			return command{Mode: "stream", Arg: args[1]}
		}
		return command{Mode: "stream"}
	case "import":
		if len(args) > 1 {
			return command{Mode: "import", Arg: args[1]}
		}
		return command{Mode: "import"}
	}
	// Otherwise treat the first arg as an epub path to open.
	return command{Mode: "open", Arg: args[0]}
}

// resolveDataDir picks the data directory: env override wins, else a "data"
// folder next to the executable.
func resolveDataDir(exePath, envOverride string) string {
	if strings.TrimSpace(envOverride) != "" {
		return envOverride
	}
	return filepath.Join(filepath.Dir(exePath), "data")
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./cmd/reader/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/reader/args.go cmd/reader/args_test.go
git commit -m "feat(cli): argument parsing and data dir resolution"
```

---

## Task 7: main 装配

**Files:**
- Create: `cmd/reader/main.go`

main 把各部分接起来。无单元测试，由构建 + 真实运行验证。

- [ ] **Step 1: 实现 main.go**

Create `cmd/reader/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"moyureader/internal/epub"
	"moyureader/internal/stream"
	"moyureader/internal/store"
	"moyureader/internal/ui"
)

func main() {
	cmd := parseArgs(os.Args[1:])

	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	dataDir := resolveDataDir(exe, os.Getenv("MOYU_DATA"))
	st := store.New(dataDir)
	lib, err := st.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法读取书架:", err)
		os.Exit(1)
	}

	switch cmd.Mode {
	case "list":
		runList(lib)
	case "import":
		runImport(st, lib, cmd.Arg)
	case "open":
		runOpen(st, lib, cmd.Arg)
	case "stream":
		runStream(st, lib, cmd.Arg)
	default:
		if err := ui.Run(st, lib, ""); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runList(lib *store.Library) {
	if len(lib.Books) == 0 {
		fmt.Println("书架为空。用 reader <某本书.epub> 导入。")
		return
	}
	for _, b := range lib.Books {
		mark := " "
		if b.ID == lib.LastBookID {
			mark = "*"
		}
		fmt.Printf("%s %-8s  %s — %s\n", mark, b.ID, b.Title, b.Author)
	}
}

func importPath(st *store.Store, lib *store.Library, path string) (*store.BookEntry, error) {
	book, err := epub.Parse(path)
	if err != nil {
		return nil, err
	}
	entry, err := st.Import(lib, path, book.Title, book.Author)
	if err != nil {
		return nil, err
	}
	if err := st.Save(lib); err != nil {
		return nil, err
	}
	return entry, nil
}

func runImport(st *store.Store, lib *store.Library, path string) {
	if path == "" {
		fmt.Fprintln(os.Stderr, "用法: reader import <某本书.epub>")
		os.Exit(2)
	}
	entry, err := importPath(st, lib, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "导入失败:", err)
		os.Exit(1)
	}
	fmt.Printf("已导入: %s — %s (id=%s)\n", entry.Title, entry.Author, entry.ID)
}

func runOpen(st *store.Store, lib *store.Library, path string) {
	entry, err := importPath(st, lib, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "打不开:", err)
		os.Exit(1)
	}
	if err := ui.Run(st, lib, entry.ID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runStream(st *store.Store, lib *store.Library, idOrEmpty string) {
	id := idOrEmpty
	if id == "" {
		id = lib.LastBookID
	}
	entry := lib.FindByID(id)
	if entry == nil {
		fmt.Fprintln(os.Stderr, "没有可续读的书。先用 reader <某本书.epub> 导入。")
		os.Exit(1)
	}
	book, err := epub.Parse(st.Dir() + string(os.PathSeparator) + entry.File)
	if err != nil {
		fmt.Fprintln(os.Stderr, "解析失败:", err)
		os.Exit(1)
	}
	s := stream.NewStreamer(book, entry.Progress, entry.Prefs.Style, 18)
	stream.Run(s, os.Stdin, os.Stdout, func(p store.Progress, style string) {
		store.UpdateProgress(lib, entry.ID, p, store.Prefs{Style: style, Mode: "inline"})
		_ = st.Save(lib)
	})
}
```

- [ ] **Step 2: 构建确认（含全量测试 + vet）**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go build ./...
go vet ./...
go test ./...
```
Expected: 全部成功。

- [ ] **Step 3: 提交**

```bash
git add cmd/reader/main.go
git commit -m "feat(cli): wire main entrypoint for tui/open/import/stream/list"
```

---

## Task 8: 单文件 exe 构建 + 真实端到端验证 + README

**Files:**
- Create: `README.md`

- [ ] **Step 1: 编译单文件 exe**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go build -ldflags "-s -w" -o reader.exe ./cmd/reader
ls -lh reader.exe
```
Expected: 生成 `reader.exe`（被 .gitignore 忽略）。

- [ ] **Step 2: 真实导入 + 列表验证（用 docs/book 里的真实 epub）**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
MOYU_DATA="$(pwd)/data" ./reader.exe import "docs/book/当下的力量 (白金版) = The Power of Now A Guide to Spiritual Enlightenment ([德] 埃克哈特 · 托利 (Eckhart Tolle) 著  曹植 译) (z-library.sk, 1lib.sk, z-lib.sk).epub"
MOYU_DATA="$(pwd)/data" ./reader.exe list
```
Expected: 打印"已导入: 当下的力量…"，list 显示该书带 `*`。

- [ ] **Step 3: 内联流式冒烟（脚本喂入回车与 q）**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
printf '\n\nq\n' | MOYU_DATA="$(pwd)/data" ./reader.exe stream
```
Expected: 终端打印若干带日志前缀的伪装行（正文藏其中），最后退出。再次 `list` 时进度应已更新（lastOpenedAt 变化）。

- [ ] **Step 4: 全屏 TUI 人工验证（你来操作）**

说明：全屏 TUI 需交互，自动化测试覆盖不到。请在**新终端窗口**手动验证：
```
reader.exe
```
检查清单：
- 书架显示《当下的力量》，回车进入阅读
- `Space` 翻页、`↑↓` 行滚动正常，中文不错位
- `Tab` 在 log/build/git 间切换；`m` 在外壳/内联两种模式间切换
- `` ` `` 或 `b` 进入老板键假屏（满屏滚动假日志、无正文），任意键返回
- `Esc` 回书架，重开后能续读到上次位置
- 清理临时构建文件：`rm -f reader.exe`（exe 已被 gitignore，不入库）

- [ ] **Step 5: 写 README**

Create `README.md`:
```markdown
# moyu-reader · 摸鱼终端阅读器

在终端里读 EPUB 小说，远看像在跑日志/编译/看 diff 的摸鱼神器。

## 构建

需要 Go（本机在 `D:\develop\Go`）：

    go build -ldflags "-s -w" -o reader.exe ./cmd/reader

产物是单文件 `reader.exe`，零依赖，拷到任意 Windows 机器即用。数据存在 exe 同级的 `data/` 文件夹（可用环境变量 `MOYU_DATA` 改）。

## 用法

    reader.exe                 打开全屏 TUI（书架）
    reader.exe 某本书.epub      导入并直接阅读
    reader.exe import 路径.epub  仅导入到书架
    reader.exe list            列出书架
    reader.exe stream [id]      内联流式模式（在集成终端里像一条吐日志的命令）

## 阅读快捷键（全屏 TUI）

- `Space`/`→`/`PgDn` 翻页 · `↑↓` 行滚动
- `Tab` 切伪装风格（log/build/git）
- `m` 切阅读模式（外壳伪装 / 正文藏日志行）
- `` ` `` 或 `b` 老板键（满屏假日志，任意键返回）
- `i` 导入 · `d` 删除 · `Esc` 回书架 · `q` 退出

## 内联流式模式

    reader.exe stream

回车出下一段；`b` 回退；`t` 切风格；`q` 退出。退出自动存进度。
```

- [ ] **Step 6: 提交**

```bash
git add README.md
git commit -m "docs: add README with build and usage"
```

---

## Self-Review（计划作者已核对）

- **Spec 覆盖**：全屏 TUI 书架/阅读/老板键(Task1-4) ✓ / 两种阅读模式 + Tab 切风格 + m 切模式(Task1,3) ✓ / 内联流式 CLI(Task5,7) ✓ / 命令行子命令 reader/open/import/stream/list(Task6-7) ✓ / data/ 定位与 exe 同级(Task6) ✓ / 进度续读与存盘(Task3,5,7) ✓ / 单文件 exe 与真实 epub 端到端(Task8) ✓。
- **类型一致性**：`NewReaderView/SetSize/Render/PageDown/PageUp/LineDown/LineUp/CycleStyle/ToggleMode/Progress/Prefs/StatusText`、`NewShelfView/MoveUp/MoveDown/Selected/Render`、`Model{Init,Update,View}` + 字段 `st/lib/screen/shelf/reader/bookID/bossActive/bossTick/importBuf`、`ui.Run`、`NewStreamer/Next/Back/CycleStyle/Progress/Style` + `stream.Run`、`parseArgs/resolveDataDir` + `command{Mode,Arg}` 在任务间签名一致。消费的计划一 API（`epub.Parse`、`render.LayoutChapter`、`disguise.{ThemeByName,NextStyle,RenderShell,RenderInline,BossScreen}`、`store.{New,Load,Save,Import,UpdateProgress,FindByID,Dir,Progress,Prefs,Library,BookEntry}`）均与计划一导出一致。
- **占位符扫描**：无 TBD/TODO；每个代码步骤含完整可编译代码。
- **依赖**：仅新增 bubbletea v1 + lipgloss v1；不引入 bubbles（书架/导入输入手写）。
- **已知取舍**：内联模式固定 wrap 宽 80；全屏 TUI 交互部分由人工验证（Task8 Step4）；lipgloss 配色仅在 View 层、不进可测逻辑。
```
