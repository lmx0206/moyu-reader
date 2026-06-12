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
