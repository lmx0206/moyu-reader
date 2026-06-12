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
