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
