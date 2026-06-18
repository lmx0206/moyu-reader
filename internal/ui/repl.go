package ui

import (
	"fmt"
	"strconv"
	"strings"

	"moyureader/internal/book"
	"moyureader/internal/disguise"
	"moyureader/internal/render"
	"moyureader/internal/store"
)

const replPrompt = "PS C:\\proj> "

// ReplView is the fake-shell reading mode: a typed prompt whose commands reveal
// the novel one paragraph at a time as "command output".
type ReplView struct {
	book          *book.Book
	chapter, para int      // para = last emitted paragraph index (-1 = none yet)
	style         string
	width, height int
	scroll        []string // rendered output/echo lines (oldest first)
	input         string
	history       []string
	histPos       int // browse index into history; == len(history) when not browsing
	seed          int // running seed so inline prefixes vary across the session
	scrollOff     int // lines scrolled up from the bottom (0 = following newest)
	quit          bool
}

// NewReplView builds a fake shell positioned so the next paragraph emitted is
// p.Para of p.Chapter (clamped).
func NewReplView(b *book.Book, p store.Progress, prefs store.Prefs, width, height int) *ReplView {
	rv := &ReplView{
		book:   b,
		style:  orDefault(prefs.Style, "log"),
		width:  width,
		height: height,
	}
	rv.chapter = clamp(p.Chapter, 0, len(b.Chapters)-1)
	rv.para = clamp(p.Para, 0, len(rv.paras())) - 1 // last emitted = (next-to-read) - 1
	rv.histPos = len(rv.history)                     // not browsing: cursor past end of (empty) history
	return rv
}

func (rv *ReplView) paras() []string {
	if rv.chapter < 0 || rv.chapter >= len(rv.book.Chapters) {
		return nil
	}
	return rv.book.Chapters[rv.chapter].Paragraphs
}

// SetSize updates terminal dimensions. Scrollback is historical text and is
// kept as-is; only the visible window changes.
func (rv *ReplView) SetSize(width, height int) { rv.width, rv.height = width, height }

// bodyHeight is the number of scrollback lines visible above the prompt.
func (rv *ReplView) bodyHeight() int {
	if rv.height < 1 {
		return 0
	}
	return rv.height - 1
}

// maxScroll is how far up the scrollback can be scrolled (0 when it fits).
func (rv *ReplView) maxScroll() int {
	m := len(rv.scroll) - rv.bodyHeight()
	if m < 0 {
		return 0
	}
	return m
}

// ScrollUp moves the view n lines toward older output (clamped).
func (rv *ReplView) ScrollUp(n int) {
	rv.scrollOff += n
	if max := rv.maxScroll(); rv.scrollOff > max {
		rv.scrollOff = max
	}
}

// ScrollDown moves the view n lines toward the newest output (clamped at the bottom).
func (rv *ReplView) ScrollDown(n int) {
	rv.scrollOff -= n
	if rv.scrollOff < 0 {
		rv.scrollOff = 0
	}
}

// Progress returns the next-to-read position (last emitted + 1).
func (rv *ReplView) Progress() store.Progress {
	p := rv.para + 1
	if p < 0 {
		p = 0
	}
	return store.Progress{Chapter: rv.chapter, Para: p}
}

// Prefs returns the current style with mode "repl".
func (rv *ReplView) Prefs() store.Prefs { return store.Prefs{Style: rv.style, Mode: "repl"} }

func (rv *ReplView) Insert(s string) { rv.input += s }
func (rv *ReplView) Backspace() {
	if n := len(rv.input); n > 0 {
		rv.input = rv.input[:n-1]
	}
}

func (rv *ReplView) HistoryPrev() {
	if len(rv.history) == 0 {
		return
	}
	if rv.histPos > 0 {
		rv.histPos--
	}
	rv.input = rv.history[rv.histPos]
}

func (rv *ReplView) HistoryNext() {
	if rv.histPos < len(rv.history)-1 {
		rv.histPos++
		rv.input = rv.history[rv.histPos]
		return
	}
	rv.histPos = len(rv.history)
	rv.input = ""
}

// Submit echoes the current input as a command line, records history, runs it,
// and clears the buffer.
func (rv *ReplView) Submit() {
	cmd := rv.input
	rv.echo(replPrompt + cmd)
	if t := strings.TrimSpace(cmd); t != "" {
		rv.history = append(rv.history, t)
	}
	rv.histPos = len(rv.history)
	rv.run(strings.TrimSpace(cmd))
	rv.input = ""
	rv.scrollOff = 0 // new output: snap the view back to the bottom
}

func (rv *ReplView) echo(line string) { rv.scroll = append(rv.scroll, line) }

func (rv *ReplView) run(cmd string) {
	f := strings.Fields(cmd)
	if len(f) == 0 {
		rv.next()
		return
	}
	switch strings.ToLower(f[0]) {
	case "n", "next":
		rv.next()
	case "git":
		if len(f) >= 2 && strings.ToLower(f[1]) == "status" {
			rv.status()
		} else {
			rv.next()
		}
	case "p", "prev":
		rv.prev()
	case "toc", "ls":
		rv.toc()
	case "cd":
		rv.cd(f)
	case "status":
		rv.status()
	case "clear", "cls":
		rv.scroll = nil
	case "help", "?":
		rv.help()
	case "q", "exit":
		rv.quit = true
	default:
		rv.echo(cmd + ": command not found")
	}
}

func (rv *ReplView) next() {
	ps := rv.paras()
	if rv.para+1 < len(ps) {
		rv.para++
	} else if rv.chapter+1 < len(rv.book.Chapters) {
		rv.chapter++
		rv.para = 0
		ps = rv.paras()
	} else {
		rv.echo("-- EOF --")
		return
	}
	rv.emit(ps[rv.para])
}

func (rv *ReplView) prev() {
	if rv.para > 0 {
		rv.para--
	} else if rv.chapter > 0 {
		rv.chapter--
		ps := rv.paras()
		rv.para = len(ps) - 1
		if rv.para < 0 {
			rv.para = 0
		}
	} else {
		rv.echo("-- BOF --")
		return
	}
	ps := rv.paras()
	if rv.para < len(ps) {
		rv.emit(ps[rv.para])
	}
}

func (rv *ReplView) emit(text string) {
	th := disguise.ThemeByName(rv.style)
	w := rv.width - disguise.PrefixWidth(th)
	if w < 10 {
		w = 10
	}
	for _, ln := range render.WrapParagraph(text, w) {
		rv.echo(th.LinePrefix(rv.seed) + ln)
		rv.seed++
	}
}

func (rv *ReplView) status() {
	total := len(rv.book.Chapters)
	pct := 0
	if total > 0 {
		pct = rv.chapter * 100 / total
	}
	rv.echo(fmt.Sprintf("on chapter %d/%d · %d%%", rv.chapter+1, total, pct))
}

func (rv *ReplView) toc() {
	for i, ch := range rv.book.Chapters {
		rv.echo(fmt.Sprintf("  %3d  %s", i+1, ch.Title))
	}
}

func (rv *ReplView) cd(f []string) {
	if len(f) < 2 {
		rv.echo("cd: no such chapter")
		return
	}
	n, err := strconv.Atoi(f[1])
	if err != nil || n < 1 || n > len(rv.book.Chapters) {
		rv.echo("cd: no such chapter")
		return
	}
	rv.chapter = n - 1
	rv.para = -1
	rv.echo("~/" + rv.book.Chapters[rv.chapter].Title)
}

func (rv *ReplView) help() {
	for _, l := range []string{
		"commands:",
		"  next, n, git log     show next paragraph",
		"  prev, p              previous paragraph",
		"  toc, ls              list chapters",
		"  cd <n>               jump to chapter n",
		"  status, git status   reading progress",
		"  clear, cls           clear screen",
		"  q, exit              quit",
	} {
		rv.echo(l)
	}
}

// Render returns exactly height lines: a scrollback window above a final prompt
// line. scrollOff shifts the window up toward older output; at offset 0 it shows
// the newest lines (bottom-aligned).
func (rv *ReplView) Render() []string {
	bodyH := rv.bodyHeight()
	if rv.scrollOff > rv.maxScroll() { // clamp in case scrollback shrank
		rv.scrollOff = rv.maxScroll()
	}
	start := len(rv.scroll) - bodyH - rv.scrollOff
	if start < 0 {
		start = 0
	}
	end := start + bodyH
	if end > len(rv.scroll) {
		end = len(rv.scroll)
	}
	var visible []string
	if start < end {
		visible = rv.scroll[start:end]
	}

	out := make([]string, 0, rv.height)
	for i := 0; i < bodyH-len(visible); i++ {
		out = append(out, "")
	}
	out = append(out, visible...)
	out = append(out, replPrompt+rv.input)
	for len(out) < rv.height {
		out = append(out, "")
	}
	if len(out) > rv.height {
		out = out[len(out)-rv.height:]
	}
	return out
}
