package ui

import (
	"strings"
	"testing"

	"moyureader/internal/book"
	"moyureader/internal/disguise"
	"moyureader/internal/store"
)

func sampleBook() *book.Book {
	mk := func(title string, n int) book.Chapter {
		ps := []string{title}
		for i := 0; i < n; i++ {
			ps = append(ps, strings.Repeat("文", 10)) // 每段 10 个双宽字
		}
		return book.Chapter{Title: title, Paragraphs: ps}
	}
	return &book.Book{
		Title:    "样书",
		Author:   "佚名",
		Chapters: []book.Chapter{mk("第一章", 20), mk("第二章", 5)},
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

func TestReaderJumpTo(t *testing.T) {
	b := sampleBook()
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "shell"}, 40, 12)
	r.JumpTo(1)
	if r.Progress().Chapter != 1 || r.Progress().Para != 0 {
		t.Fatalf("JumpTo(1) -> %+v want {1,0}", r.Progress())
	}
	r.JumpTo(99) // clamp
	if r.Progress().Chapter != len(b.Chapters)-1 {
		t.Fatalf("JumpTo(99) should clamp, got %+v", r.Progress())
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
	if start.Chapter != 0 || start.Para != 0 {
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
	r := NewReaderView(b, store.Progress{Chapter: 99, Para: 99}, store.Prefs{Style: "log", Mode: "shell"}, 40, 10)
	if r.Progress().Chapter < 0 || r.Progress().Chapter >= len(b.Chapters) {
		t.Fatalf("chapter not clamped: %+v", r.Progress())
	}
	_ = r.Render() // 不得 panic
}

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

// In inline mode each body line gets a log-style prefix prepended, then the
// whole line is clipped to the terminal width. If wrapping ignores the prefix
// width, the tail of every paragraph line is silently dropped. The novel text
// must instead wrap to leave room for the prefix, losing no characters.
func TestReaderInlineWrapsLeavingRoomForPrefix(t *testing.T) {
	const n = 120
	para := strings.Repeat("字", n)
	b := &book.Book{Title: "T", Chapters: []book.Chapter{{Title: "第一章", Paragraphs: []string{para}}}}
	r := NewReaderView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "inline"}, 100, 40)
	th := disguise.ThemeByName("log")
	got := 0
	for i, line := range r.Render() {
		prefix := th.LinePrefix(i)
		if strings.HasPrefix(line, prefix) {
			got += strings.Count(line[len(prefix):], "字")
		}
	}
	if got != n {
		t.Fatalf("inline mode dropped novel content: kept %d of %d chars (prefix overflow truncation)", got, n)
	}
}
