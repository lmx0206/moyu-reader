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
	if av.book == nil {
		return ""
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
