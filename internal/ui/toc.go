package ui

import (
	"fmt"

	"moyureader/internal/book"
)

// TOCView is a scrollable chapter list disguised as a code outline.
type TOCView struct {
	titles []string
	cursor int
	top    int // index of the first visible row
}

// NewTOCView builds a TOC with the cursor on the current chapter.
func NewTOCView(b *book.Book, current int) *TOCView {
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
