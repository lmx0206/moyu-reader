# 摸鱼阅读器 v0.2 增强 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 v0.1 上实现四项增强：阅读界面改为日志查看器风格、`g` 目录跳转（代码大纲样式）、`?` 帮助键（伪装成 --help）、GitHub Actions（CI + Release）。

**Architecture:** 复用现有分层。`internal/disguise` 重写 `RenderShell` 为 5 段布局并新增 `padBetween`/`separatorLine`/`BodyIndent`；`internal/ui` 调整 `ReaderView` 高度/边距并加 `JumpTo`、新增纯逻辑 `TOCView`、在 `Model` 接入 `screenTOC`/`screenHelp`；新增两个 `.github/workflows` YAML。

**Tech Stack:** Go、已有依赖（bubbletea v1 / lipgloss v1 / go-runewidth / x/net/html）。无新依赖。

**运行 Go：** 命令前加 `export PATH="/d/develop/Go/bin:$PATH"`（Go 在 `D:\develop\Go`，未必在 shell PATH）。

---

## 文件结构

```
internal/disguise/util.go      + padBetween / separatorLine（修改）
internal/disguise/shell.go     RenderShell 重写 + BodyIndent 常量（修改）
internal/disguise/theme.go     三风格 Header/Footer 重写（修改）
internal/disguise/render_test.go  更新 shell 断言（修改）
internal/disguise/layout_test.go  + padBetween/separator 测试（新建）
internal/ui/reader.go          bodyHeight/contentWidth 调整 + JumpTo（修改）
internal/ui/reader_test.go     更新 shell 高度断言 + JumpTo 测试（修改）
internal/ui/toc.go             TOCView（新建）
internal/ui/toc_test.go        TOCView 测试（新建）
internal/ui/model.go           screenTOC/screenHelp + 路由 + View + helpText（修改）
internal/ui/model_test.go      g/TOC/help 测试（修改）
.github/workflows/ci.yml       CI（新建）
.github/workflows/release.yml  Release（新建）
```

---

## Task 1: disguise 布局辅助（padBetween / separatorLine）

**Files:**
- Modify: `internal/disguise/util.go`
- Create: `internal/disguise/layout_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/disguise/layout_test.go`:
```go
package disguise

import (
	"testing"

	"moyureader/internal/render"
)

func TestPadBetween(t *testing.T) {
	got := padBetween("ab", "cd", 6) // 2 + gap(2) + 2
	if got != "ab  cd" {
		t.Fatalf("padBetween = %q want %q", got, "ab  cd")
	}
}

func TestPadBetweenCJKWidth(t *testing.T) {
	// "你好"=4 cells, "x"=1 cell, width 8 -> gap 3 spaces
	got := padBetween("你好", "x", 8)
	if render.StringWidth(got) != 8 {
		t.Fatalf("width = %d want 8 (%q)", render.StringWidth(got), got)
	}
}

func TestPadBetweenTooNarrowTruncates(t *testing.T) {
	got := padBetween("abcdef", "xyz", 4)
	if render.StringWidth(got) > 4 {
		t.Fatalf("should truncate to <=4, got width %d (%q)", render.StringWidth(got), got)
	}
}

func TestSeparatorLineWidth(t *testing.T) {
	got := separatorLine(10)
	if render.StringWidth(got) != 10 {
		t.Fatalf("separator width = %d want 10", render.StringWidth(got))
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/ -run "TestPadBetween|TestSeparator"`
Expected: FAIL（`padBetween` / `separatorLine` undefined）

- [ ] **Step 3: 实现辅助函数**

Replace the entire `internal/disguise/util.go` with:
```go
package disguise

import (
	"strings"

	"moyureader/internal/render"
)

// fitLine truncates or returns s. When width<=0 it returns s unchanged; when
// width>0 it truncates to width display cells (never pads, to keep golden tests
// stable and let the UI layer own padding/coloring).
func fitLine(s string, width int) string {
	if width <= 0 || render.StringWidth(s) <= width {
		return s
	}
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		rw := render.RuneWidth(r)
		if w+rw > width {
			return string(runes[:i])
		}
		w += rw
	}
	return s
}

// padBetween left-aligns left, right-aligns right, and fills the middle with
// spaces so the whole line is exactly width display cells. If there is not
// enough room, it keeps left and truncates to width.
func padBetween(left, right string, width int) string {
	gap := width - render.StringWidth(left) - render.StringWidth(right)
	if gap < 1 {
		return fitLine(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

// separatorLine returns a horizontal rule of width box-drawing dashes.
func separatorLine(width int) string {
	if width < 1 {
		width = 1
	}
	return strings.Repeat("─", width)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/ -run "TestPadBetween|TestSeparator"`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/disguise/util.go internal/disguise/layout_test.go
git commit -m "feat(disguise): padBetween and separatorLine layout helpers"
```

---

## Task 2: RenderShell 5 段布局 + 风格顶/底栏重写

**Files:**
- Modify: `internal/disguise/shell.go`
- Modify: `internal/disguise/theme.go`
- Modify: `internal/disguise/render_test.go`

- [ ] **Step 1: 更新 shell 测试为新布局**

Replace `TestRenderShellWrapsBodyWithChrome` in `internal/disguise/render_test.go` with:
```go
func TestRenderShellFiveSectionLayout(t *testing.T) {
	th := ThemeByName("build")
	body := []string{"正文一", "正文二"}
	out := RenderShell(th, body, 40, "ch.1/10 · 0%")
	// 4 装饰行(顶栏/分隔/分隔/底栏) + 2 正文 = 6
	if len(out) != 6 {
		t.Fatalf("want 6 lines, got %d: %#v", len(out), out)
	}
	if !strings.Contains(out[0], "gradle") {
		t.Fatalf("top bar should be build theme: %q", out[0])
	}
	if !strings.Contains(out[1], "─") {
		t.Fatalf("line1 should be separator: %q", out[1])
	}
	if !strings.Contains(out[2], "正文一") {
		t.Fatalf("body line should be present (indented): %q", out[2])
	}
	if !strings.HasPrefix(out[2], "   ") {
		t.Fatalf("body should be indented by 3 spaces: %q", out[2])
	}
	if !strings.Contains(out[4], "─") {
		t.Fatalf("line4 should be separator: %q", out[4])
	}
	if !strings.Contains(out[5], "SUCCESSFUL") {
		t.Fatalf("bottom bar should be build theme: %q", out[5])
	}
	if !strings.Contains(out[5], "ch.1/10") {
		t.Fatalf("bottom bar should embed status: %q", out[5])
	}
}
```
(Leave `TestRenderInlinePrefixesEachLine` unchanged.)

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/ -run TestRenderShell`
Expected: FAIL（旧 RenderShell 仅 4 行、无缩进/分隔）

- [ ] **Step 3: 重写 RenderShell**

Replace the entire `internal/disguise/shell.go` with:
```go
package disguise

import "strings"

// BodyIndent is the left margin (in spaces) applied to novel body lines in
// shell-disguise mode. The UI's contentWidth must reserve this space.
const BodyIndent = 3

// RenderShell renders reading mode A as a "log viewer": a theme top bar, a
// separator, the indented novel body, a separator, and a theme bottom status
// bar. Total decoration is 4 lines. status is embedded in the bottom bar.
func RenderShell(th Theme, body []string, width int, status string) []string {
	sep := separatorLine(width)
	indent := strings.Repeat(" ", BodyIndent)
	out := make([]string, 0, len(body)+4)
	out = append(out, th.Header(width, status))
	out = append(out, sep)
	for _, l := range body {
		if l == "" {
			out = append(out, "")
		} else {
			out = append(out, indent+l)
		}
	}
	out = append(out, sep)
	out = append(out, th.Footer(width, status))
	return out
}
```

- [ ] **Step 4: 重写三风格 Header/Footer**

In `internal/disguise/theme.go`, replace the six methods (logTheme/buildTheme/gitTheme Header & Footer). 

Replace logTheme's:
```go
func (logTheme) Header(width int, status string) string {
	return fitLine("app.log — tail -f   "+status, width)
}
func (logTheme) Footer(width int, status string) string {
	return fitLine("INFO  watching for changes   "+status, width)
}
```
with:
```go
func (logTheme) Header(width int, status string) string {
	return padBetween("app.log · tail -f", "● running", width)
}
func (logTheme) Footer(width int, status string) string {
	return padBetween("INFO  142 passed · "+status, "? help", width)
}
```

Replace buildTheme's:
```go
func (buildTheme) Header(width int, status string) string {
	return fitLine("> gradle build   "+status, width)
}
func (buildTheme) Footer(width int, status string) string {
	return fitLine("BUILD SUCCESSFUL in 12s   "+status, width)
}
```
with:
```go
func (buildTheme) Header(width int, status string) string {
	return padBetween("> gradle build", "building…", width)
}
func (buildTheme) Footer(width int, status string) string {
	return padBetween("BUILD SUCCESSFUL in 12s · "+status, "? help", width)
}
```

Replace gitTheme's:
```go
func (gitTheme) Header(width int, status string) string {
	return fitLine("git diff --stat   "+status, width)
}
func (gitTheme) Footer(width int, status string) string {
	return fitLine("3 files changed, 128 insertions(+)   "+status, width)
}
```
with:
```go
func (gitTheme) Header(width int, status string) string {
	return padBetween("git log -p", "● HEAD", width)
}
func (gitTheme) Footer(width int, status string) string {
	return padBetween("3 files changed · "+status, "? help", width)
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/...`
Expected: PASS（全部 disguise 测试）

- [ ] **Step 6: 提交**

```bash
git add internal/disguise/shell.go internal/disguise/theme.go internal/disguise/render_test.go
git commit -m "feat(disguise): log-viewer shell layout with top/bottom bars"
```

---

## Task 3: ReaderView 适配新布局 + JumpTo

**Files:**
- Modify: `internal/ui/reader.go`
- Modify: `internal/ui/reader_test.go`

- [ ] **Step 1: 更新/新增 reader 测试**

In `internal/ui/reader_test.go`, replace `TestReaderRenderExactHeightShellMode` with:
```go
func TestReaderRenderExactHeightShellMode(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	lines := r.Render()
	if len(lines) != 12 {
		t.Fatalf("shell render must be exactly height(12), got %d", len(lines))
	}
	if !strings.Contains(lines[0], "tail -f") {
		t.Fatalf("top bar missing, line0=%q", lines[0])
	}
	if !strings.Contains(lines[1], "─") {
		t.Fatalf("line1 should be separator, got %q", lines[1])
	}
	if !strings.Contains(lines[11], "? help") {
		t.Fatalf("bottom bar should show help hint, last=%q", lines[11])
	}
}

func TestReaderJumpTo(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	r.JumpTo(1)
	if r.Progress().Chapter != 1 || r.Progress().Line != 0 {
		t.Fatalf("JumpTo(1) -> %+v want {1,0}", r.Progress())
	}
	r.JumpTo(99) // clamp
	if r.Progress().Chapter != len(b.Chapters)-1 {
		t.Fatalf("JumpTo(99) should clamp, got %+v", r.Progress())
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestReaderRenderExactHeightShellMode|TestReaderJumpTo"`
Expected: FAIL（footer 不含 "? help"；`JumpTo` undefined）

- [ ] **Step 3: 调整 bodyHeight / contentWidth 并加 JumpTo**

In `internal/ui/reader.go`, replace `contentWidth` and `bodyHeight`:
```go
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
```
with:
```go
// rightMargin keeps a small gutter on the right of shell-mode body text.
const rightMargin = 1

// contentWidth is the wrap width for novel body. In shell mode it reserves the
// left indent (disguise.BodyIndent) plus a right margin so text breathes.
func (r *ReaderView) contentWidth() int {
	w := r.width
	if r.mode == "shell" {
		w -= disguise.BodyIndent + rightMargin
	}
	if w < 10 {
		return 10
	}
	return w
}

// bodyHeight is the number of novel-body lines per page (chrome excluded). Shell
// mode now has 4 decoration lines (top bar + 2 separators + bottom bar).
func (r *ReaderView) bodyHeight() int {
	h := r.height
	if r.mode == "shell" {
		h -= 4
	}
	if h < 1 {
		return 1
	}
	return h
}

// JumpTo moves to the start of the given chapter (clamped).
func (r *ReaderView) JumpTo(chapter int) {
	r.chapter = clamp(chapter, 0, len(r.book.Chapters)-1)
	r.line = 0
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestReader"`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ui/reader.go internal/ui/reader_test.go
git commit -m "feat(ui): adapt ReaderView to 4-line shell chrome and add JumpTo"
```

---

## Task 4: TOCView（章节目录，代码大纲样式）

**Files:**
- Create: `internal/ui/toc.go`
- Create: `internal/ui/toc_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/ui/toc_test.go`:
```go
package ui

import (
	"strings"
	"testing"
)

func TestTOCStartsAtCurrentChapter(t *testing.T) {
	tv := NewTOCView(sampleBook(), 1)
	if tv.Selected() != 1 {
		t.Fatalf("cursor should start at current chapter 1, got %d", tv.Selected())
	}
}

func TestTOCMoveClamps(t *testing.T) {
	tv := NewTOCView(sampleBook(), 0) // sampleBook has 2 chapters
	tv.MoveUp()
	if tv.Selected() != 0 {
		t.Fatalf("up at top stays 0, got %d", tv.Selected())
	}
	tv.MoveDown()
	tv.MoveDown() // clamp at last (index 1)
	if tv.Selected() != 1 {
		t.Fatalf("down clamps at 1, got %d", tv.Selected())
	}
}

func TestTOCRenderShowsTitlesAndCursor(t *testing.T) {
	tv := NewTOCView(sampleBook(), 0)
	out := tv.Render(60, 10)
	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "第一章") || !strings.Contains(joined, "第二章") {
		t.Fatalf("TOC should list real chapter titles: %q", joined)
	}
	if !strings.Contains(joined, "func ch") {
		t.Fatalf("TOC should look like a code outline: %q", joined)
	}
}

func TestTOCScrollsWhenManyChapters(t *testing.T) {
	// build a book with 50 chapters
	b := sampleBook()
	for i := 0; i < 50; i++ {
		b.Chapters = append(b.Chapters, b.Chapters[0])
	}
	tv := NewTOCView(b, 0)
	for i := 0; i < 40; i++ {
		tv.MoveDown()
	}
	out := tv.Render(60, 8)
	if len(out) > 8 {
		t.Fatalf("render must fit height(8), got %d", len(out))
	}
	// cursor (chapter 40) must be visible within the rendered window
	if !strings.Contains(strings.Join(out, "\n"), "ch41()") {
		t.Fatalf("cursor chapter should be visible after scrolling: %v", out)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestTOC`
Expected: FAIL（`NewTOCView` undefined）

- [ ] **Step 3: 实现 TOCView**

Create `internal/ui/toc.go`:
```go
package ui

import (
	"fmt"

	"moyureader/internal/epub"
)

// TOCView is a scrollable chapter list disguised as a code outline.
type TOCView struct {
	titles []string
	cursor int
	top    int // index of the first visible row
}

// NewTOCView builds a TOC with the cursor on the current chapter.
func NewTOCView(b *epub.Book, current int) *TOCView {
	titles := make([]string, len(b.Chapters))
	for i, c := range b.Chapters {
		titles[i] = c.Title
	}
	tv := &TOCView{titles: titles}
	tv.cursor = clamp(current, 0, len(titles)-1)
	return tv
}

// MoveUp / MoveDown move the cursor with clamping.
func (tv *TOCView) MoveUp() {
	if tv.cursor > 0 {
		tv.cursor--
	}
}

func (tv *TOCView) MoveDown() {
	if tv.cursor < len(tv.titles)-1 {
		tv.cursor++
	}
}

// Selected returns the highlighted chapter index.
func (tv *TOCView) Selected() int { return tv.cursor }

// Render draws the outline, scrolling so the cursor stays visible, fitting
// within height lines (1 header + rows).
func (tv *TOCView) Render(width, height int) []string {
	rows := height - 1
	if rows < 1 {
		rows = 1
	}
	// scroll window so cursor is visible
	if tv.cursor < tv.top {
		tv.top = tv.cursor
	}
	if tv.cursor >= tv.top+rows {
		tv.top = tv.cursor - rows + 1
	}
	if tv.top < 0 {
		tv.top = 0
	}

	out := []string{fmt.Sprintf("outline · book.go        %d symbols", len(tv.titles))}
	for i := tv.top; i < len(tv.titles) && i < tv.top+rows; i++ {
		marker := "   ▸"
		if i == tv.cursor {
			marker = " ▸ ▸"
		}
		out = append(out, fmt.Sprintf("%s func ch%02d()   %s", marker, i+1, tv.titles[i]))
	}
	return out
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run TestTOC`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/ui/toc.go internal/ui/toc_test.go
git commit -m "feat(ui): TOCView code-outline chapter list"
```

---

## Task 5: Model 接入 TOC 与帮助

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`

- [ ] **Step 1: 写失败测试**

Add to `internal/ui/model_test.go` (append these functions; keep existing ones). Also update `newReaderModel` to set `book`:

Replace the `newReaderModel` helper with:
```go
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
```

Append:
```go
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
```

Add `"strings"` to the imports of `model_test.go` if not present.

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestModelG|TestModelTOC|TestModelHelp"`
Expected: FAIL（`screenTOC` / `screenHelp` / `book` / `toc` undefined）

- [ ] **Step 3: 扩展 Model 字段与屏幕常量**

In `internal/ui/model.go`, replace the screen const block:
```go
const (
	screenShelf screen = iota
	screenReader
	screenImport
)
```
with:
```go
const (
	screenShelf screen = iota
	screenReader
	screenImport
	screenTOC
	screenHelp
)
```

Replace the Model struct field block (the part after `screen        screen`) — add `book`, `toc`, `helpReturn`. The full struct becomes:
```go
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
```

- [ ] **Step 4: 在 openBook 中保存 book 引用**

In `internal/ui/model.go` `openBook`, replace:
```go
	m.reader = NewReaderView(book, e.Progress, e.Prefs, m.width, m.height)
	m.bookID = id
	m.screen = screenReader
```
with:
```go
	m.reader = NewReaderView(book, e.Progress, e.Prefs, m.width, m.height)
	m.book = book
	m.bookID = id
	m.screen = screenReader
```

- [ ] **Step 5: 重写 handleKey 路由（加入 TOC / Help）**

In `internal/ui/model.go`, replace the whole `handleKey` method with:
```go
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
```

- [ ] **Step 6: 在 reader 按键里加入 `g`**

In `internal/ui/model.go` `handleReaderKey`, replace:
```go
	case "tab":
		m.reader.CycleStyle()
	case "m":
		m.reader.ToggleMode()
	}
	return m, nil
}
```
with:
```go
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
```

- [ ] **Step 7: 更新 View 处理 TOC / Help，并加 helpText**

In `internal/ui/model.go` `View`, replace the `switch m.screen {` block's body so the cases include TOC and Help. Replace:
```go
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
```
with:
```go
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
```

- [ ] **Step 8: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/...`
Expected: PASS（全部 ui 测试）

- [ ] **Step 9: 提交**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): TOC jump (g) and help overlay (?) wired into model"
```

---

## Task 6: GitHub Actions（CI + Release）

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: 创建 CI 工作流**

Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26.x'
      - name: Vet
        run: go vet ./...
      - name: Test
        run: go test ./...
      - name: Build
        run: go build ./...
```

- [ ] **Step 2: 创建 Release 工作流**

Create `.github/workflows/release.yml`:
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build-windows-exe:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26.x'
      - name: Build reader.exe
        run: GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o reader.exe ./cmd/reader
      - name: Publish release
        uses: softprops/action-gh-release@v2
        with:
          files: reader.exe
```

- [ ] **Step 3: 确认两个工作流文件已就位**

Run: `ls -la .github/workflows/`
Expected: 列出 `ci.yml` 与 `release.yml`（YAML 语法在首次推送后由 GitHub 校验）

- [ ] **Step 4: 提交**

```bash
git add .github/workflows/ci.yml .github/workflows/release.yml
git commit -m "ci: add CI (test) and Release (build exe on tag) workflows"
```

---

## Task 7: 全量验证 + 真实构建

**Files:** 无（验证）

- [ ] **Step 1: vet + 全量测试**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go vet ./...
go test ./...
```
Expected: 全部 PASS。

- [ ] **Step 2: 构建单文件 exe**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go build -ldflags "-s -w" -o reader.exe ./cmd/reader
ls -lh reader.exe
rm -f reader.exe
```
Expected: 成功生成后删除（exe 已 gitignore）。

- [ ] **Step 3: 全屏 TUI 人工验证（你来操作）**

在新终端：
```
go build -o reader.exe ./cmd/reader
.\reader.exe
```
检查清单：
- 阅读界面是"日志查看器"样式：顶部 `app.log · tail -f … ● running`、分隔线、留白正文、分隔线、底部状态栏带 `? help`
- `g` 弹出代码大纲样式的章节目录，↑↓ 选择、回车跳章、Esc 取消
- `?` 弹出 --help 样式帮助，任意键关闭
- `Tab` 切风格、`m` 切模式、`` ` `` 老板键仍正常
- 清理：`rm -f reader.exe`

---

## Self-Review（计划作者已核对）

- **Spec 覆盖**：① 日志查看器布局(Task1-3) ✓ ② `g` 目录跳转代码大纲(Task4-5) ✓ ③ `?` 帮助伪装 --help(Task5) ✓ ④ GitHub Actions CI+Release(Task6) ✓ 测试策略全覆盖(各 Task) ✓。
- **类型一致性**：`disguise.{padBetween,separatorLine,BodyIndent,RenderShell}`、各 Theme `Header/Footer` 签名不变（仅实现变）、`ReaderView.{contentWidth,bodyHeight,JumpTo}` + 常量 `rightMargin`、`TOCView.{NewTOCView,MoveUp,MoveDown,Selected,Render}`、`Model` 新字段 `book/toc/helpReturn` + `screenTOC/screenHelp` + `handleTOCKey/helpText` 跨任务一致。
- **占位符扫描**：无 TBD/TODO；每个代码步骤含完整代码或精确 old→new 替换。
- **已知取舍**：`paintShell` 仅给首/尾行上色（分隔线按正文色），纯装饰、不影响功能与测试；TOC 顶栏不做右对齐（用固定空格），避免跨包依赖 `padBetween`。
```
