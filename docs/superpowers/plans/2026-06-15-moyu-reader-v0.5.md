# 摸鱼阅读器 v0.5 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 v0.5：统一的「标注（书签+笔记）」功能，并把阅读位置改为排版无关的段落锚点，从根因修复「resize 后内容显示不全」。

**Architecture:** `render` 新增「行↔段落」纯函数映射；`store` 增 `Annotation` 模型与增删、`Progress` 由行号改段落号；`ReaderView` 用段落锚点（构造/`SetSize` 重锚/`Progress`/`JumpToPara`）；`ui` 新增「加标注」输入与「标注列表」覆盖层（伪装成调试器断点面板）。

**Tech Stack:** Go、现有依赖，无新依赖。

**运行 Go：** 命令前加 `export PATH="/d/develop/Go/bin:$PATH"`。

---

## 文件结构

```
internal/render/anchor.go            + ParagraphStartLines/ParaStartLine/LineToPara（新建）
internal/render/anchor_test.go       测试（新建）
internal/store/model.go              + Annotation 类型、BookEntry.Annotations、Progress 改 Para（修改）
internal/store/annotations.go        + AddAnnotation/DeleteAnnotation（新建）
internal/store/annotations_test.go   测试（新建）
internal/store/import_test.go        Progress.Line→Para（修改）
internal/ui/reader.go                段落锚点：构造/SetSize/Progress/JumpToPara（修改）
internal/ui/reader_test.go           Line→Para + resize 守恒 + JumpToPara（修改）
internal/stream/stream.go            Progress.Line→Para（修改）
internal/stream/stream_test.go       Progress.Line→Para（修改）
internal/ui/annotation.go            + AnnotationView（断点面板伪装）（新建）
internal/ui/model.go                 screen 枚举 + a/l 按键 + 输入/列表处理 + View + help（修改）
internal/ui/model_test.go            加标注 / 列表跳转删除测试（修改）
```

---

## Task 1: render 行↔段落映射

**Files:**
- Create: `internal/render/anchor.go`
- Create: `internal/render/anchor_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/render/anchor_test.go`:
```go
package render

import "testing"

func TestParagraphStartLines(t *testing.T) {
	// 两个单行段落：段间一个空行 -> 起始行 0 和 2
	got := ParagraphStartLines([]string{"abc", "def"}, 80)
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("starts = %v want [0 2]", got)
	}
}

func TestParaStartLineClamp(t *testing.T) {
	ps := []string{"a", "b"}
	if ParaStartLine(ps, 80, 99) != ParagraphStartLines(ps, 80)[1] {
		t.Fatalf("para should clamp to last")
	}
	if ParaStartLine(nil, 80, 0) != 0 {
		t.Fatalf("empty -> 0")
	}
}

func TestLineToParaRoundTrip(t *testing.T) {
	ps := []string{"一二三四五六七八九十", "甲乙丙丁", "x"}
	for p := range ps {
		start := ParaStartLine(ps, 6, p)
		if got := LineToPara(ps, 6, start); got != p {
			t.Fatalf("round trip para %d -> line %d -> para %d", p, start, got)
		}
	}
}

func TestLineToParaBlankAndOOB(t *testing.T) {
	ps := []string{"abc", "def"}
	if LineToPara(ps, 80, 1) != 0 { // 段间空行归前一段
		t.Fatalf("blank separator should map to preceding para")
	}
	if LineToPara(ps, 80, 999) != 1 { // 越界行归最后一段
		t.Fatalf("oob line should map to last para")
	}
	if LineToPara(nil, 80, 0) != 0 {
		t.Fatalf("empty -> 0")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/render/ -run "ParagraphStartLines|ParaStartLine|LineToPara"`
Expected: FAIL（函数未定义）

- [ ] **Step 3: 实现**

Create `internal/render/anchor.go`:
```go
package render

// ParagraphStartLines returns, for each paragraph, the display-line index where
// its content begins when laid out at width. It matches LayoutChapter, which
// inserts one blank line between paragraphs.
func ParagraphStartLines(paragraphs []string, width int) []int {
	starts := make([]int, len(paragraphs))
	line := 0
	for i, p := range paragraphs {
		if i > 0 {
			line++ // blank separator line before this paragraph
		}
		starts[i] = line
		line += len(WrapParagraph(p, width))
	}
	return starts
}

// ParaStartLine returns the start display-line of paragraph para (clamped to a
// valid index). It returns 0 when there are no paragraphs.
func ParaStartLine(paragraphs []string, width, para int) int {
	if len(paragraphs) == 0 {
		return 0
	}
	if para < 0 {
		para = 0
	}
	if para >= len(paragraphs) {
		para = len(paragraphs) - 1
	}
	return ParagraphStartLines(paragraphs, width)[para]
}

// LineToPara returns the index of the paragraph that owns display-line line (the
// last paragraph whose start <= line), clamped to [0, len-1]. It returns 0 when
// there are no paragraphs.
func LineToPara(paragraphs []string, width, line int) int {
	if len(paragraphs) == 0 {
		return 0
	}
	para := 0
	for i, s := range ParagraphStartLines(paragraphs, width) {
		if s <= line {
			para = i
		} else {
			break
		}
	}
	return para
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/render/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/render/anchor.go internal/render/anchor_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(render): paragraph<->line anchor mapping

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: store 标注模型 + 增删（additive）

**Files:**
- Modify: `internal/store/model.go`
- Create: `internal/store/annotations.go`
- Create: `internal/store/annotations_test.go`

> 本任务只新增 `Annotation` 与增删函数，不动 `Progress`（Progress 改在 Task 3）。

- [ ] **Step 1: 写失败测试**

Create `internal/store/annotations_test.go`:
```go
package store

import "testing"

func TestAddAndDeleteAnnotation(t *testing.T) {
	lib := NewLibrary()
	lib.Books = append(lib.Books, BookEntry{ID: "a"})
	AddAnnotation(lib, "a", Annotation{Chapter: 1, Para: 3, Note: "hi"})
	AddAnnotation(lib, "a", Annotation{Chapter: 2, Para: 0})
	e := lib.FindByID("a")
	if len(e.Annotations) != 2 {
		t.Fatalf("want 2 annotations, got %d", len(e.Annotations))
	}
	DeleteAnnotation(lib, "a", 0)
	if len(e.Annotations) != 1 || e.Annotations[0].Chapter != 2 {
		t.Fatalf("delete wrong: %+v", e.Annotations)
	}
	DeleteAnnotation(lib, "a", 99)  // out of range -> no-op
	DeleteAnnotation(lib, "zzz", 0) // unknown id -> no-op
	if len(e.Annotations) != 1 {
		t.Fatalf("no-op delete changed slice: %+v", e.Annotations)
	}
}

func TestAddAnnotationUnknownID(t *testing.T) {
	lib := NewLibrary()
	AddAnnotation(lib, "nope", Annotation{}) // must not panic
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/store/ -run "Annotation"`
Expected: FAIL（`Annotation`/`AddAnnotation`/`DeleteAnnotation` 未定义）

- [ ] **Step 3: 实现 — 数据结构**

In `internal/store/model.go`, add the `Annotation` type (place it just after the `Progress` type definition):
```go
// Annotation marks a reading position with an optional note. An empty Note is a
// plain bookmark; a non-empty Note is an annotation.
type Annotation struct {
	Chapter   int    `json:"chapter"`
	Para      int    `json:"para"`
	Note      string `json:"note,omitempty"`
	CreatedAt string `json:"createdAt"`
}
```

In the same file, add the `Annotations` field to `BookEntry` (after the `Prefs Prefs` field):
```go
	Prefs        Prefs        `json:"prefs"`
	Annotations  []Annotation `json:"annotations,omitempty"`
	Broken       bool         `json:"broken,omitempty"`
```
(The existing `BookEntry` already has `Prefs` and `Broken` fields; insert `Annotations` between them and keep the struct tags aligned.)

- [ ] **Step 4: 实现 — 增删函数**

Create `internal/store/annotations.go`:
```go
package store

// AddAnnotation appends an annotation to the book with the given id. It is a
// no-op if the id is unknown. The caller is responsible for Save.
func AddAnnotation(lib *Library, id string, a Annotation) {
	e := lib.FindByID(id)
	if e == nil {
		return
	}
	e.Annotations = append(e.Annotations, a)
}

// DeleteAnnotation removes the annotation at index i for the book with the given
// id. It is a no-op if the id is unknown or i is out of range. The caller is
// responsible for Save.
func DeleteAnnotation(lib *Library, id string, i int) {
	e := lib.FindByID(id)
	if e == nil || i < 0 || i >= len(e.Annotations) {
		return
	}
	e.Annotations = append(e.Annotations[:i], e.Annotations[i+1:]...)
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/store/...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/store/model.go internal/store/annotations.go internal/store/annotations_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(store): Annotation model + add/delete helpers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Progress 改段落锚点 + ReaderView 段落锚定 + stream 迁移（原子）

> `Progress` 字段从 `Line` 改为 `Para`，这会牵动 ui/stream/相关测试，必须一次提交保持编译与测试通过。内含 resize bug 的「红→绿」回归测试。

**Files:**
- Modify: `internal/store/model.go`, `internal/store/import_test.go`
- Modify: `internal/ui/reader.go`, `internal/ui/reader_test.go`
- Modify: `internal/stream/stream.go`, `internal/stream/stream_test.go`

- [ ] **Step 1: 改 Progress 类型 + 调用方迁移（保留旧 SetSize，先让仓库编译通过）**

In `internal/store/model.go`, replace:
```go
// Progress is a stable reading position: chapter index + display-line index.
type Progress struct {
	Chapter int `json:"chapter"`
	Line    int `json:"line"`
}
```
with:
```go
// Progress is a stable reading position: chapter index + paragraph index
// (layout-independent, so it survives window-width changes).
type Progress struct {
	Chapter int `json:"chapter"`
	Para    int `json:"para"`
}
```

In `internal/store/import_test.go`, replace:
```go
	UpdateProgress(lib, "a", Progress{Chapter: 2, Line: 40}, Prefs{Style: "git", Mode: "inline"})
	e := lib.FindByID("a")
	if e.Progress.Chapter != 2 || e.Progress.Line != 40 {
```
with:
```go
	UpdateProgress(lib, "a", Progress{Chapter: 2, Para: 40}, Prefs{Style: "git", Mode: "inline"})
	e := lib.FindByID("a")
	if e.Progress.Chapter != 2 || e.Progress.Para != 40 {
```

In `internal/ui/reader.go`, add a `paras()` helper (place it just before `chapterLines`):
```go
// paras returns the current chapter's paragraphs, or nil if out of range.
func (r *ReaderView) paras() []string {
	if r.chapter < 0 || r.chapter >= len(r.book.Chapters) {
		return nil
	}
	return r.book.Chapters[r.chapter].Paragraphs
}
```

In `NewReaderView`, replace:
```go
	r.chapter = clamp(p.Chapter, 0, len(b.Chapters)-1)
	r.line = clampLine(p.Line, r.chapterLineCount())
	return r
```
with:
```go
	r.chapter = clamp(p.Chapter, 0, len(b.Chapters)-1)
	r.line = clampLine(render.ParaStartLine(r.paras(), r.contentWidth(), p.Para), r.chapterLineCount())
	return r
```

Replace the whole `JumpTo` method:
```go
// JumpTo moves to the start of the given chapter (clamped).
func (r *ReaderView) JumpTo(chapter int) {
	r.chapter = clamp(chapter, 0, len(r.book.Chapters)-1)
	r.line = 0
}
```
with:
```go
// JumpTo moves to the start of the given chapter (clamped).
func (r *ReaderView) JumpTo(chapter int) { r.JumpToPara(chapter, 0) }

// JumpToPara moves to the start of the given paragraph in the given chapter
// (both clamped).
func (r *ReaderView) JumpToPara(chapter, para int) {
	r.chapter = clamp(chapter, 0, len(r.book.Chapters)-1)
	r.line = clampLine(render.ParaStartLine(r.paras(), r.contentWidth(), para), r.chapterLineCount())
}
```

Replace the `Progress` method:
```go
// Progress returns the current stable position.
func (r *ReaderView) Progress() store.Progress {
	return store.Progress{Chapter: r.chapter, Line: r.line}
}
```
with:
```go
// Progress returns the current stable position (top paragraph).
func (r *ReaderView) Progress() store.Progress {
	return store.Progress{Chapter: r.chapter, Para: render.LineToPara(r.paras(), r.contentWidth(), r.line)}
}
```

In `internal/stream/stream.go`, add a `chapterParas` helper (place it just before `chapterLines`):
```go
func (s *Streamer) chapterParas() []string {
	if s.chapter < 0 || s.chapter >= len(s.book.Chapters) {
		return nil
	}
	return s.book.Chapters[s.chapter].Paragraphs
}
```
Replace:
```go
	s.line = p.Line
	return s
```
with:
```go
	s.line = render.ParaStartLine(s.chapterParas(), wrapWidth, p.Para)
	return s
```
Replace:
```go
func (s *Streamer) Progress() store.Progress {
	return store.Progress{Chapter: s.chapter, Line: s.line}
}
```
with:
```go
func (s *Streamer) Progress() store.Progress {
	return store.Progress{Chapter: s.chapter, Para: render.LineToPara(s.chapterParas(), wrapWidth, s.line)}
}
```

In `internal/stream/stream_test.go`, replace:
```go
	if p2.Line <= p1.Line && p2.Chapter == p1.Chapter {
```
with:
```go
	if p2.Para <= p1.Para && p2.Chapter == p1.Chapter {
```

In `internal/ui/reader_test.go`, update the existing `Line` references:
- In `TestReaderJumpTo`, replace `if r.Progress().Chapter != 1 || r.Progress().Line != 0 {` with `if r.Progress().Chapter != 1 || r.Progress().Para != 0 {`.
- In `TestReaderPageDownCrossesChapter`, replace `if start.Chapter != 0 || start.Line != 0 {` with `if start.Chapter != 0 || start.Para != 0 {`.
- In `TestReaderProgressClampOnConstruct`, replace `store.Progress{Chapter: 99, Line: 99}` with `store.Progress{Chapter: 99, Para: 99}`.

- [ ] **Step 2: 运行（确认仓库编译 + 既有测试通过；旧 SetSize 仍是 clamp）**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go build ./... && go test ./...`
Expected: PASS（此时 resize 仍未修，但既有断言不覆盖该场景）

- [ ] **Step 3: 加 resize 回归测试 + JumpToPara 测试（应看到 resize 测试失败）**

In `internal/ui/reader_test.go`, append:
```go
func longParaBook() *book.Book {
	long := strings.Repeat("文", 200) // ~400 cells, wraps differently per width
	return &book.Book{
		Title: "L",
		Chapters: []book.Chapter{{Title: "第一章", Paragraphs: []string{
			strings.Repeat("甲", 30), long, strings.Repeat("乙", 30), long,
		}}},
	}
}

func TestReaderResizePreservesParagraph(t *testing.T) {
	b := longParaBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	r.PageDown()
	r.PageDown()
	before := r.Progress().Para
	r.SetSize(100, 12) // much wider -> fewer wrapped lines
	if got := r.Progress().Para; got != before {
		t.Fatalf("resize should preserve top paragraph: before=%d after=%d", before, got)
	}
	if len(r.Render()) != 12 {
		t.Fatalf("render must stay full height after resize")
	}
}

func TestReaderJumpToPara(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	r.JumpToPara(1, 2)
	if r.Progress().Chapter != 1 || r.Progress().Para != 2 {
		t.Fatalf("JumpToPara(1,2) -> %+v", r.Progress())
	}
}
```
Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestReaderResizePreservesParagraph|TestReaderJumpToPara"`
Expected: `TestReaderResizePreservesParagraph` FAILS (before/after paragraph differs — the bug); `TestReaderJumpToPara` PASSES.

- [ ] **Step 4: 实现 SetSize 段落重锚（修复 bug）**

In `internal/ui/reader.go`, replace:
```go
// SetSize updates terminal dimensions and re-clamps the line index.
func (r *ReaderView) SetSize(width, height int) {
	r.width, r.height = width, height
	r.line = clampLine(r.line, r.chapterLineCount())
}
```
with:
```go
// SetSize updates terminal dimensions, re-anchoring to the paragraph currently
// at the top so resizing the window keeps the reader in place (and never leaves
// a half-empty page from a now-stale line index).
func (r *ReaderView) SetSize(width, height int) {
	topPara := render.LineToPara(r.paras(), r.contentWidth(), r.line) // at old width
	r.width, r.height = width, height
	r.line = clampLine(render.ParaStartLine(r.paras(), r.contentWidth(), topPara), r.chapterLineCount())
}
```

- [ ] **Step 5: 运行测试确认全绿**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./...`
Expected: PASS（resize 测试现在通过）

- [ ] **Step 6: 提交**

```bash
git add -A
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "fix(ui): paragraph-anchored positions; resize keeps content in place

Progress is now (chapter, paragraph); ReaderView re-anchors on SetSize, fixing
content being cut off / lost after a window resize. stream migrated likewise.

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: UI 加标注（`a`）+ 标注列表（`l`）

**Files:**
- Create: `internal/ui/annotation.go`
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`

- [ ] **Step 1: 写失败测试**

In `internal/ui/model_test.go`, append:
```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestModelAddAnnotation|TestModelAnnotationList"`
Expected: FAIL（`screenAnnotate`/`screenAnnotList`/`m.annot` 未定义）

- [ ] **Step 3: 实现 AnnotationView**

Create `internal/ui/annotation.go`:
```go
package ui

import (
	"fmt"

	"moyureader/internal/book"
	"moyureader/internal/render"
	"moyureader/internal/store"
)

// AnnotationView lists a book's annotations, disguised as a debugger breakpoint
// panel.
type AnnotationView struct {
	book   *book.Book
	items  []store.Annotation
	cursor int
	top    int
}

// NewAnnotationView builds a list over a copy of items.
func NewAnnotationView(b *book.Book, items []store.Annotation) *AnnotationView {
	return &AnnotationView{book: b, items: append([]store.Annotation(nil), items...)}
}

func (av *AnnotationView) MoveUp() {
	if av.cursor > 0 {
		av.cursor--
	}
}
func (av *AnnotationView) MoveDown() {
	if av.cursor < len(av.items)-1 {
		av.cursor++
	}
}

// Selected returns the highlighted annotation and whether one exists.
func (av *AnnotationView) Selected() (store.Annotation, bool) {
	if av.cursor < 0 || av.cursor >= len(av.items) {
		return store.Annotation{}, false
	}
	return av.items[av.cursor], true
}

// Index returns the highlighted row index (matches the stored slice order).
func (av *AnnotationView) Index() int { return av.cursor }

// summary previews an annotation: its note, else the start of its paragraph.
func (av *AnnotationView) summary(a store.Annotation) string {
	if a.Note != "" {
		return a.Note
	}
	if a.Chapter >= 0 && a.Chapter < len(av.book.Chapters) {
		ps := av.book.Chapters[a.Chapter].Paragraphs
		if a.Para >= 0 && a.Para < len(ps) {
			return ps[a.Para]
		}
	}
	return ""
}

// Render draws the breakpoint-styled list within height lines.
func (av *AnnotationView) Render(width, height int) []string {
	out := []string{fmt.Sprintf("breakpoints (%d)", len(av.items))}
	if len(av.items) == 0 {
		return append(out, "  — none set —")
	}
	rows := height - 1
	if rows < 1 {
		rows = 1
	}
	if av.cursor < av.top {
		av.top = av.cursor
	}
	if av.cursor >= av.top+rows {
		av.top = av.cursor - rows + 1
	}
	if av.top < 0 {
		av.top = 0
	}
	for i := av.top; i < len(av.items) && i < av.top+rows; i++ {
		a := av.items[i]
		marker := "  "
		if i == av.cursor {
			marker = "● "
		}
		head := fmt.Sprintf("%sbrk %d  ch%d:¶%d  ", marker, i+1, a.Chapter+1, a.Para+1)
		out = append(out, render.PadRight(head+av.summary(a), width))
	}
	return out
}
```

- [ ] **Step 4: 接线 model.go — screen 枚举 + 字段**

In `internal/ui/model.go`, replace:
```go
const (
	screenShelf screen = iota
	screenReader
	screenImport
	screenTOC
	screenHelp
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
	screenAnnotate
	screenAnnotList
)
```

Add the `annot` field and `annotBuf` to the `Model` struct. Replace:
```go
	toc        *TOCView
	helpReturn screen
```
with:
```go
	toc        *TOCView
	annot      *AnnotationView
	helpReturn screen
```
and replace:
```go
	importBuf string // typed path in import screen
	status    string // transient status line (errors etc.)
```
with:
```go
	importBuf string // typed path in import screen
	annotBuf  string // typed note in annotate screen
	status    string // transient status line (errors etc.)
```

- [ ] **Step 5: 接线 model.go — 按键路由**

In `handleKey`, replace:
```go
	case screenTOC:
		return m.handleTOCKey(key)
	}
```
with:
```go
	case screenTOC:
		return m.handleTOCKey(key)
	case screenAnnotate:
		return m.handleAnnotateKey(msg)
	case screenAnnotList:
		return m.handleAnnotListKey(key)
	}
```

In `handleReaderKey`, replace:
```go
	case "s":
		m.reader.ToggleNav()
	case "g":
```
with:
```go
	case "s":
		m.reader.ToggleNav()
	case "a":
		m.annotBuf = ""
		m.screen = screenAnnotate
	case "l":
		if e := m.lib.FindByID(m.bookID); e != nil && m.book != nil {
			m.annot = NewAnnotationView(m.book, e.Annotations)
			m.screen = screenAnnotList
		}
	case "g":
```

- [ ] **Step 6: 接线 model.go — 两个处理函数**

In `internal/ui/model.go`, add these two functions (place them right after `handleTOCKey`):
```go
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
		m.status = "已加标注"
	case "backspace":
		if n := len(m.annotBuf); n > 0 {
			m.annotBuf = m.annotBuf[:n-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.annotBuf += string(msg.Runes)
		} else if msg.Type == tea.KeySpace {
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
```
(`time` and `strings` are already imported in model.go.)

- [ ] **Step 7: 接线 model.go — View + help**

In `View`, replace:
```go
	case screenImport:
		return "导入 EPUB（粘贴 .epub 完整路径后回车，Esc 取消）:\n\n> " + m.importBuf + "\n\n" + m.status
```
with:
```go
	case screenImport:
		return "导入 EPUB（粘贴 .epub 完整路径后回车，Esc 取消）:\n\n> " + m.importBuf + "\n\n" + m.status
	case screenAnnotList:
		return strings.Join(paintDim(m.annot.Render(m.width, m.height)), "\n")
	case screenAnnotate:
		return "加标注（批注可留空 = 书签，回车保存，Esc 取消）:\n\n> " + m.annotBuf
```

In `helpText`, replace:
```go
		"  s    scroll/page mode        g        goto section",
		"  `/b  minimize (same key restores)     ?  help",
```
with:
```go
		"  s    scroll/page mode        g        goto section",
		"  a    add bookmark/note       l        list bookmarks",
		"  `/b  minimize (same key restores)     ?  help",
```

- [ ] **Step 8: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/... && go vet ./internal/ui/`
Expected: PASS

- [ ] **Step 9: 提交**

```bash
git add internal/ui/annotation.go internal/ui/model.go internal/ui/model_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(ui): annotations — add (a) and breakpoint-styled list (l)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: 全量验证 + 跑给你看效果

**Files:** 临时 `cmd/_frames/main.go`（验证后删除，不提交）

- [ ] **Step 1: 全量 vet + test**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go vet ./...
go test ./...
```
Expected: 全部 PASS。

- [ ] **Step 2: 写临时帧渲染程序（看标注列表 + resize 守恒）**

Create `cmd/_frames/main.go`:
```go
package main

import (
	"fmt"
	"strings"

	"moyureader/internal/book"
	"moyureader/internal/store"
	"moyureader/internal/ui"
)

func main() {
	long := strings.Repeat("文", 200)
	b := &book.Book{Title: "L", Chapters: []book.Chapter{{Title: "第一章", Paragraphs: []string{
		strings.Repeat("甲", 30), long, strings.Repeat("乙", 30), long,
	}}}}

	r := ui.NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	r.PageDown()
	r.PageDown()
	fmt.Printf("resize 前 顶部段落=%d\n", r.Progress().Para)
	r.SetSize(100, 12)
	fmt.Printf("resize 后 顶部段落=%d（应相等，内容不丢）\n", r.Progress().Para)

	av := ui.NewAnnotationView(b, []store.Annotation{
		{Chapter: 0, Para: 1, Note: "这里有伏笔"},
		{Chapter: 0, Para: 3},
	})
	fmt.Println("\n===== 标注列表（断点面板伪装） =====")
	fmt.Println(strings.Join(av.Render(64, 8), "\n"))
}
```

- [ ] **Step 3: 运行帧程序，肉眼检查**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go run ./cmd/_frames/`
Expected 检查点：
- 「resize 前 顶部段落」与「resize 后 顶部段落」**数字相等**（bug 已修）。
- 标注列表：顶栏 `breakpoints (2)`；首行带 `●` 高亮、形如 `● brk 1  ch1:¶2  这里有伏笔`；第二条无批注时显示段落开头摘要。

- [ ] **Step 4: 删除临时程序**

Run: `rm -rf cmd/_frames`
Then: `git status --porcelain`（应只剩无关或为空；`cmd/_frames` 已删）

- [ ] **Step 5: 无需提交（临时程序已删，无遗留改动）**

```bash
git status --short
```

---

## Self-Review（计划作者已核对）

- **Spec 覆盖**：① render 行↔段落(Task1) ✓ ② Annotation 模型+增删(Task2) ✓ ③ Progress 改段落 + ReaderView 锚定 + resize 修复(Task3) ✓ ④ stream 迁移(Task3) ✓ ⑤ UI 加标注+列表+断点伪装+help(Task4) ✓ ⑥ 测试策略：render 往返 / store 增删 / resize 守恒 / JumpToPara / 加标注 / 列表跳删(Task1-4) ✓ ⑦ 跑给你看效果(Task5) ✓。
- **类型一致性**：`render.{ParagraphStartLines,ParaStartLine,LineToPara}`、`store.{Annotation,AddAnnotation,DeleteAnnotation,Progress.Para}`、`ReaderView.{paras,JumpToPara,Progress,SetSize}`、`Streamer.{chapterParas,Progress}`、`ui.{AnnotationView,screenAnnotate,screenAnnotList,Model.annot,Model.annotBuf,handleAnnotateKey,handleAnnotListKey}` 跨任务一致。
- **占位符扫描**：无 TBD/TODO；每个代码步骤为完整代码或精确 old→new 替换。
- **构建顺序**：Task1/2 独立绿；Task3 先迁移类型保编译绿(Step1-2)，再加回归测试见红(Step3)，最后修 SetSize 转绿(Step4-5)；Task4 在 Task3 之上。
- **已知取舍**：旧存档 `Progress.line` 不迁移、一次性回章首；标注列表按存储顺序展示，删除按下标，无下标错位；帧程序仅本地肉眼验证、不入库。
```
