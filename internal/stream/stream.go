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
