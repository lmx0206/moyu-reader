# 摸鱼阅读器 v0.3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 v0.3：修复 log 时间与老板键退出，新增翻页/滚动双模式（滚动条），状态栏显示本章页码，并更新 README。

**Architecture:** `render` 加纯函数 `PadRight`/`Scrollbar`；`disguise` 用真实基准时间生成日志时间戳且 `RenderShell` 不再缩进正文（缩进职责上移到 `ReaderView`）；`ui` 的 `ReaderView` 负责正文缩进、滚动条合成、本章页码，`Model` 收紧老板键退出并加 `s` 切换导航模式。

**Tech Stack:** Go、现有依赖（bubbletea v1 / lipgloss v1 / go-runewidth / x/net/html）。无新依赖。

**运行 Go：** 命令前加 `export PATH="/d/develop/Go/bin:$PATH"`。

---

## 文件结构

```
internal/render/scrollbar.go        + PadRight / Scrollbar（新建）
internal/render/scrollbar_test.go   测试（新建）
internal/disguise/theme.go          logTheme 时间戳用 clockBase + logClock（修改）
internal/disguise/clock_test.go     logClock 测试（新建）
internal/disguise/shell.go          RenderShell 不再缩进 + 删除 BodyIndent（修改）
internal/disguise/render_test.go    更新 shell 断言（修改）
internal/ui/reader.go               BodyIndent 常量 + nav/ToggleNav + Render 合成 + StatusText 页码（修改）
internal/ui/reader_test.go          页码/滚动条/缩进/ToggleNav 测试（修改）
internal/ui/model.go                老板键退出收紧 + s 键 + helpText（修改）
internal/ui/model_test.go           老板键退出 + s 切换测试（修改）
README.md                           详细使用说明（修改）
```

---

## Task 1: render.PadRight 与 render.Scrollbar

**Files:**
- Create: `internal/render/scrollbar.go`
- Create: `internal/render/scrollbar_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/render/scrollbar_test.go`:
```go
package render

import (
	"strings"
	"testing"
)

func TestPadRightASCII(t *testing.T) {
	if got := PadRight("ab", 5); got != "ab   " {
		t.Fatalf("PadRight = %q want %q", got, "ab   ")
	}
}

func TestPadRightCJKExactWidth(t *testing.T) {
	// "你好"=4 cells, pad to 6 -> 2 trailing spaces
	got := PadRight("你好", 6)
	if StringWidth(got) != 6 {
		t.Fatalf("width = %d want 6 (%q)", StringWidth(got), got)
	}
}

func TestPadRightTruncates(t *testing.T) {
	got := PadRight("abcdef", 3)
	if got != "abc" {
		t.Fatalf("PadRight truncate = %q want abc", got)
	}
}

func TestScrollbarLengthAndGlyphs(t *testing.T) {
	bars := Scrollbar(100, 0, 10)
	if len(bars) != 10 {
		t.Fatalf("len = %d want 10", len(bars))
	}
	joined := strings.Join(bars, "")
	if !strings.Contains(joined, "█") {
		t.Fatalf("should contain a thumb glyph: %q", joined)
	}
	if !strings.Contains(joined, "░") {
		t.Fatalf("should contain track glyph: %q", joined)
	}
}

func TestScrollbarThumbMovesDown(t *testing.T) {
	top := Scrollbar(100, 0, 10)
	bottom := Scrollbar(100, 90, 10)
	// thumb at top: first glyph is thumb; at bottom: last glyph is thumb
	if top[0] != "█" {
		t.Fatalf("thumb should be at top: %v", top)
	}
	if bottom[len(bottom)-1] != "█" {
		t.Fatalf("thumb should be at bottom: %v", bottom)
	}
}

func TestScrollbarNoScrollAllTrack(t *testing.T) {
	bars := Scrollbar(5, 0, 10) // content fits viewport
	for _, b := range bars {
		if b != "░" {
			t.Fatalf("no-scroll should be all track, got %v", bars)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/render/ -run "TestPadRight|TestScrollbar"`
Expected: FAIL（`PadRight`/`Scrollbar` undefined）

- [ ] **Step 3: 实现**

Create `internal/render/scrollbar.go`:
```go
package render

import "strings"

// PadRight pads s with spaces to exactly width display cells, or truncates it
// to width cells if it is longer. CJK width-aware.
func PadRight(s string, width int) string {
	w := StringWidth(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	runes := []rune(s)
	acc := 0
	for i, r := range runes {
		rw := RuneWidth(r)
		if acc+rw > width {
			return string(runes[:i])
		}
		acc += rw
	}
	return s
}

// Scrollbar returns viewport glyphs representing a vertical scrollbar for a list
// of total items with the window starting at top. "█" marks the thumb, "░" the
// track. When everything fits (total<=viewport) all glyphs are track.
func Scrollbar(total, top, viewport int) []string {
	bars := make([]string, viewport)
	if viewport <= 0 {
		return bars
	}
	if total <= viewport {
		for i := range bars {
			bars[i] = "░"
		}
		return bars
	}
	thumb := viewport * viewport / total
	if thumb < 1 {
		thumb = 1
	}
	maxTop := total - viewport
	pos := 0
	if maxTop > 0 {
		pos = top * (viewport - thumb) / maxTop
	}
	if pos < 0 {
		pos = 0
	}
	if pos > viewport-thumb {
		pos = viewport - thumb
	}
	for i := range bars {
		if i >= pos && i < pos+thumb {
			bars[i] = "█"
		} else {
			bars[i] = "░"
		}
	}
	return bars
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/render/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/render/scrollbar.go internal/render/scrollbar_test.go
git commit -m "feat(render): PadRight and Scrollbar helpers"
```

---

## Task 2: 修复 log 时间（真实基准时间）

**Files:**
- Modify: `internal/disguise/theme.go`
- Create: `internal/disguise/clock_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/disguise/clock_test.go`:
```go
package disguise

import (
	"testing"
	"time"
)

func TestLogClockFormatAndIncrement(t *testing.T) {
	base := time.Date(2026, 6, 12, 14, 23, 1, 0, time.UTC)
	if got := logClock(base, 0); got != "14:23:01" {
		t.Fatalf("logClock(base,0) = %q want 14:23:01", got)
	}
	if got := logClock(base, 5); got != "14:23:06" {
		t.Fatalf("logClock(base,5) = %q want 14:23:06", got)
	}
}

func TestLogPrefixUsesClock(t *testing.T) {
	// With a deterministic base, the prefix must contain that time.
	old := clockBase
	clockBase = time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC)
	defer func() { clockBase = old }()
	if got := (logTheme{}).LinePrefix(0); got[:10] != "[09:30:00]" {
		t.Fatalf("prefix should start with clock time, got %q", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/ -run "TestLogClock|TestLogPrefixUsesClock"`
Expected: FAIL（`logClock` / `clockBase` undefined）

- [ ] **Step 3: 实现**

In `internal/disguise/theme.go`, change the import block:
```go
import "fmt"
```
to:
```go
import (
	"fmt"
	"time"
)
```

Add after the import block (e.g. just before `// styleOrder ...`):
```go
// clockBase anchors fake log timestamps to the program's real start time so
// they look like a process that is currently emitting logs.
var clockBase = time.Now()

// logClock formats base advanced by seed seconds as HH:MM:SS.
func logClock(base time.Time, seed int) string {
	return base.Add(time.Duration(seed) * time.Second).Format("15:04:05")
}
```

Replace `logTheme.LinePrefix`:
```go
func (logTheme) LinePrefix(seed int) string {
	ts := fmt.Sprintf("%02d:%02d:%02d", 8+(seed/3600)%10, (seed/60)%60, seed%60)
	lvl := logLevels[seed%len(logLevels)]
	cls := logClasses[(seed/7)%len(logClasses)]
	return fmt.Sprintf("[%s] %-5s %s - ", ts, lvl, cls)
}
```
with:
```go
func (logTheme) LinePrefix(seed int) string {
	ts := logClock(clockBase, seed)
	lvl := logLevels[seed%len(logLevels)]
	cls := logClasses[(seed/7)%len(logClasses)]
	return fmt.Sprintf("[%s] %-5s %s - ", ts, lvl, cls)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/...`
Expected: PASS（含既有 `TestLogThemePrefixDeterministic`）

- [ ] **Step 5: 提交**

```bash
git add internal/disguise/theme.go internal/disguise/clock_test.go
git commit -m "fix(disguise): base log timestamps on real start time"
```

---

## Task 3: 缩进上移 + 翻页/滚动双模式 + 本章页码

> 本任务同时改 `disguise.RenderShell`（去掉缩进）与 `ui/reader.go`（接管缩进 + 滚动条 + 页码），必须一起提交以保持编译与测试通过。

**Files:**
- Modify: `internal/disguise/shell.go`
- Modify: `internal/disguise/render_test.go`
- Modify: `internal/ui/reader.go`
- Modify: `internal/ui/reader_test.go`

- [ ] **Step 1: 更新 disguise shell 测试（正文不再缩进）**

In `internal/disguise/render_test.go`, replace `TestRenderShellFiveSectionLayout` with:
```go
func TestRenderShellFiveSectionLayout(t *testing.T) {
	th := ThemeByName("build")
	body := []string{"正文一", "正文二"}
	out := RenderShell(th, body, 40, "ch.1/10 · 本章 1/1页 · 0%")
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
	// RenderShell no longer indents — body is placed verbatim (caller indents).
	if out[2] != "正文一" || out[3] != "正文二" {
		t.Fatalf("body must be verbatim: %#v", out)
	}
	if !strings.Contains(out[4], "─") {
		t.Fatalf("line4 should be separator: %q", out[4])
	}
	if !strings.Contains(out[5], "SUCCESSFUL") {
		t.Fatalf("bottom bar should be build theme: %q", out[5])
	}
}
```

- [ ] **Step 2: 改 RenderShell（去缩进、删 BodyIndent）**

Replace the entire `internal/disguise/shell.go` with:
```go
package disguise

// RenderShell renders reading mode A as a "log viewer": a theme top bar, a
// separator, the novel body (verbatim — the caller applies any indent/right
// gutter), a separator, and a theme bottom status bar. Decoration is 4 lines.
func RenderShell(th Theme, body []string, width int, status string) []string {
	sep := separatorLine(width)
	out := make([]string, 0, len(body)+4)
	out = append(out, th.Header(width, status))
	out = append(out, sep)
	out = append(out, body...)
	out = append(out, sep)
	out = append(out, th.Footer(width, status))
	return out
}
```

- [ ] **Step 3: 运行（预期 ui 包编译失败，因 disguise.BodyIndent 没了）**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go build ./... 2>&1 | head`
Expected: `internal/ui/reader.go` 报 `disguise.BodyIndent` 未定义 —— 下一步修复。

- [ ] **Step 4: 改 ui/reader.go（接管缩进 + nav + 滚动条 + 页码）**

In `internal/ui/reader.go`, add `"strings"` to the imports so the block is:
```go
import (
	"fmt"
	"strings"

	"moyureader/internal/disguise"
	"moyureader/internal/epub"
	"moyureader/internal/render"
	"moyureader/internal/store"
)
```

Add a `nav` field to the struct (after `mode string`):
```go
type ReaderView struct {
	book    *epub.Book
	chapter int
	line    int // top display-line index within current chapter
	width   int
	height  int
	style   string
	mode    string
	nav     string // "page" | "scroll"
}
```

In `NewReaderView`, set the default nav (add `nav: "page"` to the struct literal):
```go
	r := &ReaderView{
		book:   b,
		style:  orDefault(prefs.Style, "log"),
		mode:   orDefault(prefs.Mode, "shell"),
		nav:    "page",
		width:  width,
		height: height,
	}
```

Replace the `rightMargin` const + `contentWidth` block:
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
```
with:
```go
// BodyIndent is the left margin (spaces) applied to shell-mode body text.
const BodyIndent = 3

// rightMargin keeps a one-column gutter on the right (also where the scrollbar
// is drawn in scroll mode).
const rightMargin = 1

// contentWidth is the wrap width for novel body. In shell mode it reserves the
// left indent plus a right margin so text breathes.
func (r *ReaderView) contentWidth() int {
	w := r.width
	if r.mode == "shell" {
		w -= BodyIndent + rightMargin
	}
	if w < 10 {
		return 10
	}
	return w
}

// ToggleNav flips page<->scroll navigation.
func (r *ReaderView) ToggleNav() {
	if r.nav == "scroll" {
		r.nav = "page"
	} else {
		r.nav = "scroll"
	}
}

// Nav returns the current navigation mode.
func (r *ReaderView) Nav() string { return r.nav }
```

Replace the `Render` method:
```go
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

	// shell mode: indent body; in scroll mode append a right-edge scrollbar.
	indent := strings.Repeat(" ", BodyIndent)
	var bars []string
	if r.nav == "scroll" {
		bars = render.Scrollbar(len(lines), r.line, bh)
	}
	body := make([]string, bh)
	for i := 0; i < bh; i++ {
		text := indent + page[i]
		if r.nav == "scroll" {
			text = render.PadRight(text, r.width-1) + bars[i]
		}
		body[i] = text
	}
	return disguise.RenderShell(th, body, r.width, r.StatusText())
}
```

Replace `StatusText`:
```go
// StatusText is the chrome status, e.g. "ch.1/2 · 0%".
func (r *ReaderView) StatusText() string {
	total := len(r.book.Chapters)
	pct := 0
	if total > 0 {
		pct = r.chapter * 100 / total
	}
	return fmt.Sprintf("ch.%d/%d · %d%%", r.chapter+1, total, pct)
}
```
with:
```go
// StatusText is the chrome status, e.g. "ch.1/2 · 本章 1/3页 · 0%".
func (r *ReaderView) StatusText() string {
	totalCh := len(r.book.Chapters)
	pct := 0
	if totalCh > 0 {
		pct = r.chapter * 100 / totalCh
	}
	bh := r.bodyHeight()
	totalPages := (r.chapterLineCount() + bh - 1) / bh
	if totalPages < 1 {
		totalPages = 1
	}
	page := r.line/bh + 1
	if page > totalPages {
		page = totalPages
	}
	return fmt.Sprintf("ch.%d/%d · 本章 %d/%d页 · %d%%", r.chapter+1, totalCh, page, totalPages, pct)
}
```

- [ ] **Step 5: 更新 ui/reader 测试**

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
	// body line is indented by ReaderView now
	if !strings.HasPrefix(lines[2], "   ") {
		t.Fatalf("body line should be indented by 3 spaces: %q", lines[2])
	}
	if !strings.Contains(lines[11], "? help") {
		t.Fatalf("bottom bar should show help hint, last=%q", lines[11])
	}
}

func TestReaderStatusShowsChapterPages(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	if !strings.Contains(r.StatusText(), "本章 1/") {
		t.Fatalf("status should show chapter page number: %q", r.StatusText())
	}
}

func TestReaderScrollModeDrawsScrollbar(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	r.ToggleNav() // -> scroll
	if r.Nav() != "scroll" {
		t.Fatalf("ToggleNav should switch to scroll, got %q", r.Nav())
	}
	lines := r.Render()
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "█") && !strings.Contains(joined, "░") {
		t.Fatalf("scroll mode should draw a scrollbar glyph:\n%s", joined)
	}
	if len(lines) != 12 {
		t.Fatalf("scroll render must still be height(12), got %d", len(lines))
	}
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/... ./internal/ui/...`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/disguise/shell.go internal/disguise/render_test.go internal/ui/reader.go internal/ui/reader_test.go
git commit -m "feat(ui): page/scroll nav with scrollbar, chapter page numbers; move body indent to ReaderView"
```

---

## Task 4: Model 老板键退出收紧 + s 键 + 帮助

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`

- [ ] **Step 1: 更新/新增 model 测试**

In `internal/ui/model_test.go`, replace `TestModelBossKeyToggles` with:
```go
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
```

Append:
```go
func TestModelSTogglesNav(t *testing.T) {
	m := newReaderModel(t)
	start := m.reader.Nav()
	nm, _ := m.Update(keyRunes("s"))
	m = nm.(*Model)
	if m.reader.Nav() == start {
		t.Fatalf("s should toggle nav mode, still %q", m.reader.Nav())
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestModelBossKeyToggles|TestModelSTogglesNav"`
Expected: FAIL（普通键仍退出；`s` 未处理）

- [ ] **Step 3: 收紧老板键退出**

In `internal/ui/model.go` `handleKey`, replace:
```go
	// Boss screen swallows all keys: any key restores reading.
	if m.bossActive {
		m.bossActive = false
		return m, nil
	}

	key := msg.String()
```
with:
```go
	key := msg.String()

	// Boss screen swallows all keys; only the boss key itself restores reading,
	// so you can mash keys to look busy without revealing the novel.
	if m.bossActive {
		if key == "`" || key == "b" {
			m.bossActive = false
		}
		return m, nil
	}
```

- [ ] **Step 4: 加 `s` 键**

In `internal/ui/model.go` `handleReaderKey`, replace:
```go
	case "tab":
		m.reader.CycleStyle()
	case "m":
		m.reader.ToggleMode()
	case "g":
```
with:
```go
	case "tab":
		m.reader.CycleStyle()
	case "m":
		m.reader.ToggleMode()
	case "s":
		m.reader.ToggleNav()
	case "g":
```

- [ ] **Step 5: 更新帮助文本**

In `internal/ui/model.go` `helpText`, replace the reader keybindings block:
```go
		"KEYBINDINGS (reader):",
		"  space/→/pgdn  next page      up/down  scroll line",
		"  tab  switch profile          m        toggle view",
		"  g    goto section            `/b      minimize",
		"  ?    help                    esc      back to list",
```
with:
```go
		"KEYBINDINGS (reader):",
		"  space/→/pgdn  next page      up/down  scroll line",
		"  tab  switch profile          m        toggle view",
		"  s    scroll/page mode        g        goto section",
		"  `/b  minimize (same key restores)     ?  help",
		"  esc  back to list            q        quit",
```

- [ ] **Step 6: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/...`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): boss key exits only on boss key; s toggles scroll/page; help updated"
```

---

## Task 5: 跑给你看效果（帧渲染）+ 全量验证

**Files:** 临时 `cmd/_frames/main.go`（验证后删除，不提交）

- [ ] **Step 1: 全量 vet + test**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go vet ./...
go test ./...
```
Expected: 全部 PASS。

- [ ] **Step 2: 写临时帧渲染程序**

Create `cmd/_frames/main.go`:
```go
package main

import (
	"fmt"
	"strings"

	"moyureader/internal/epub"
	"moyureader/internal/store"
	"moyureader/internal/ui"
)

func main() {
	book, err := epub.Parse("docs/book/当下的力量 (白金版) = The Power of Now A Guide to Spiritual Enlightenment ([德] 埃克哈特 · 托利 (Eckhart Tolle) 著  曹植 译) (z-library.sk, 1lib.sk, z-lib.sk).epub")
	if err != nil {
		fmt.Println("parse:", err)
		return
	}
	show := func(title string, r *ui.ReaderView) {
		fmt.Println("\n===== " + title + " =====")
		fmt.Println(strings.Join(r.Render(), "\n"))
	}
	// 翻页模式 shell（log 风格）
	r := ui.NewReaderView(book, store.Progress{Chapter: 3, Line: 4}, store.Prefs{Style: "log", Mode: "shell"}, 64, 16)
	show("PAGE mode (shell/log) — 注意真实时间在内联里才有，这里看顶/底栏与缩进", r)
	// 滚动模式 shell（看右侧滚动条）
	r.ToggleNav()
	show("SCROLL mode (shell) — 右侧应有 █/░ 滚动条", r)
	// 内联模式（看真实时间戳）
	r2 := ui.NewReaderView(book, store.Progress{Chapter: 3, Line: 4}, store.Prefs{Style: "log", Mode: "inline"}, 64, 16)
	show("INLINE mode (log) — 时间戳应为真实当前时间", r2)
}
```

- [ ] **Step 3: 运行帧程序，肉眼检查效果**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go run ./cmd/_frames/`
Expected 检查点：
- PAGE：顶栏 `app.log · tail -f … ● running`、正文左缩进、底栏含 `本章 X/Y页` 与 `? help`。
- SCROLL：每行最右出现 `█`/`░` 滚动条。
- INLINE：行首时间戳是**真实当前时间**（非 08:00:00）。

- [ ] **Step 4: 删除临时程序**

Run: `rm -rf cmd/_frames`

- [ ] **Step 5: 提交（仅在有遗留改动时；正常无改动可跳过）**

```bash
git status --short
```
（cmd/_frames 已删，无需提交。）

---

## Task 6: 更新 README（详细版）

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 重写 README**

Replace the entire `README.md` with:
```markdown
# moyu-reader · 摸鱼终端阅读器

在终端里读 EPUB 小说，远看像在跑日志 / 编译 / 看 git diff 的「摸鱼神器」。
单文件 `.exe`、零依赖、可放 U 盘随身带；支持全屏 TUI 与内联流式两种用法。

## 特性

- 📖 **EPUB 阅读**：导入 epub、记忆每本书的阅读进度、重开自动续读
- 🕵️ **三层伪装**
  - 阅读模式 A（外壳）：日志查看器外观，正文正常排版、舒服读
  - 阅读模式 B（内联）：每行正文伪装成带真实时间戳的日志行
  - 老板键：满屏自动滚动的假日志，**只有老板键本身能退出**（可随便敲键装忙）
- 🎭 **三种伪装风格**：日志 `log` / 构建 `build` / `git`，`Tab` 循环切换
- 🧭 **翻页 / 滚动双模式**：`s` 切换；滚动模式右侧带滚动条
- 📑 **目录跳转**：`g` 弹出「代码大纲」样式章节列表，回车跳章
- ❓ **帮助键**：`?` 弹出伪装成 `--help` 的快捷键说明
- 📊 状态栏显示章节进度与本章页码

## 安装

### 方式一：直接下载（推荐）
到 [Releases](https://github.com/lmx0206/moyu-reader/releases) 下载最新 `reader.exe`，双击即用。

### 方式二：自行编译
需要 Go 1.26+：
```
go build -ldflags "-s -w" -o reader.exe ./cmd/reader
```
产物是单文件 `reader.exe`。数据（书架 + 进度 + 导入的 epub）存在 exe 同级的 `data/` 文件夹，可用环境变量 `MOYU_DATA` 改到别处。

> 小贴士：可把 exe 改名成 `tail.exe` / `gradlew.exe`，进程列表里更不显眼。

## 用法

```
reader.exe                 打开全屏 TUI（书架）
reader.exe 某本书.epub      导入并直接阅读
reader.exe import 路径.epub  仅导入到书架
reader.exe list            列出书架
reader.exe stream [id]      内联流式模式（在集成终端里像一条吐日志的命令）
```

## 快捷键

### 书架
| 键 | 作用 |
|----|------|
| `↑` `↓` | 选择 |
| `Enter` | 打开阅读 |
| `i` | 导入 .epub（输入路径） |
| `d` | 删除 |
| `?` | 帮助 |
| `q` | 退出 |

### 阅读
| 键 | 作用 |
|----|------|
| `Space` `→` `PgDn` | 下一页 |
| `←` `PgUp` | 上一页 |
| `↑` `↓` | 逐行滚动 |
| `s` | 翻页 / 滚动模式切换 |
| `Tab` | 切伪装风格（log/build/git） |
| `m` | 切阅读模式（外壳 / 内联） |
| `g` | 目录跳转 |
| `` ` `` 或 `b` | 老板键（再按一次返回） |
| `?` | 帮助 |
| `Esc` | 回书架 |

### 内联流式模式
| 输入 | 作用 |
|------|------|
| 回车 | 下一段 |
| `b` | 回退 |
| `t` | 切风格 |
| `q` | 退出 |

## 发布新版本

打一个 `v*` tag 并推送，GitHub Actions 会自动交叉编译 `reader.exe` 并发到 Releases：
```
git tag v0.3.0
git push origin v0.3.0
```

## 开发

```
go test ./...     # 跑全部测试
go vet ./...      # 静态检查
```
CI（`.github/workflows/ci.yml`）会在每次 push / PR 自动跑测试。
```

- [ ] **Step 2: 提交**

```bash
git add README.md
git commit -m "docs: detailed README for v0.3"
```

---

## Self-Review（计划作者已核对）

- **Spec 覆盖**：① log 时间真实基准(Task2) ✓ ② 老板键仅自身退出(Task4) ✓ ③ 翻页/滚动双模式+滚动条(Task1+Task3) ✓ ④ 本章页码(Task3) ✓ README(Task6) ✓ 跑给你看效果(Task5) ✓。
- **类型一致性**：`render.{PadRight,Scrollbar}`、`disguise.{clockBase,logClock}`、`disguise.RenderShell`（签名不变、行为去缩进）、`ui.ReaderView.{BodyIndent 常量,nav,ToggleNav,Nav,Render,StatusText,contentWidth}`、`Model` 的 `s`/老板键退出/`helpText` 跨任务一致。`disguise.BodyIndent` 删除后仅 `ui.BodyIndent` 存在，reader.go 引用已改为本地常量。
- **占位符扫描**：无 TBD/TODO；每个代码步骤为完整代码或精确 old→new 替换。
- **构建顺序**：Task3 故意先让 ui 编译失败（Step3）再在 Step4 修复，二者同一次提交，最终绿。
- **已知取舍**：内联模式不画滚动条；nav 不持久化（每次启动默认翻页）；帧程序仅本地肉眼验证、不入库。
```
