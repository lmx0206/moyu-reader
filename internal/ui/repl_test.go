package ui

import (
	"strings"
	"testing"

	"moyureader/internal/book"
	"moyureader/internal/store"
)

func replBook() *book.Book {
	return &book.Book{
		Title: "T",
		Chapters: []book.Chapter{
			{Title: "第一章", Paragraphs: []string{"甲甲甲", "乙乙乙", "丙丙丙"}},
			{Title: "第二章", Paragraphs: []string{"丁丁丁"}},
		},
	}
}

func newRepl() *ReplView {
	return NewReplView(replBook(), store.Progress{}, store.Prefs{Style: "log", Mode: "repl"}, 60, 10)
}

func TestReplNextEmitsParagraphAndAdvances(t *testing.T) {
	rv := newRepl()
	rv.input = "next"
	rv.Submit()
	joined := strings.Join(rv.Render(), "\n")
	if !strings.Contains(joined, "甲甲甲") {
		t.Fatalf("next should emit first paragraph:\n%s", joined)
	}
	if rv.Progress().Para != 1 {
		t.Fatalf("next should advance to para 1, got %d", rv.Progress().Para)
	}
}

func TestReplEmptyEnterIsNext(t *testing.T) {
	rv := newRepl()
	rv.input = ""
	rv.Submit()
	if !strings.Contains(strings.Join(rv.Render(), "\n"), "甲甲甲") {
		t.Fatalf("empty enter should act as next")
	}
}

func TestReplUnknownCommand(t *testing.T) {
	rv := newRepl()
	rv.input = "frobnicate"
	rv.Submit()
	if !strings.Contains(strings.Join(rv.Render(), "\n"), "command not found") {
		t.Fatalf("unknown command should print 'command not found'")
	}
}

func TestReplTocAndCd(t *testing.T) {
	rv := newRepl()
	rv.input = "toc"
	rv.Submit()
	if !strings.Contains(strings.Join(rv.Render(), "\n"), "第二章") {
		t.Fatalf("toc should list chapters")
	}
	rv.input = "cd 2"
	rv.Submit()
	if rv.Progress().Chapter != 1 {
		t.Fatalf("cd 2 should jump to chapter index 1, got %d", rv.Progress().Chapter)
	}
	rv.input = "cd 9"
	rv.Submit()
	if !strings.Contains(strings.Join(rv.Render(), "\n"), "no such chapter") {
		t.Fatalf("cd 9 should report no such chapter")
	}
}

func TestReplClearAndQuit(t *testing.T) {
	rv := newRepl()
	rv.input = "next"
	rv.Submit()
	rv.input = "clear"
	rv.Submit()
	// after clear, only the prompt line should carry content
	nonEmpty := 0
	for _, l := range rv.Render() {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 1 {
		t.Fatalf("clear should wipe scrollback, non-empty lines = %d", nonEmpty)
	}
	rv.input = "q"
	rv.Submit()
	if !rv.quit {
		t.Fatalf("q should set quit")
	}
}

func TestReplRenderHeightAndPrompt(t *testing.T) {
	rv := newRepl()
	rv.input = "hel"
	lines := rv.Render()
	if len(lines) != 10 {
		t.Fatalf("Render must be exactly height (10), got %d", len(lines))
	}
	if !strings.HasPrefix(lines[9], "PS ") || !strings.HasSuffix(lines[9], "hel") {
		t.Fatalf("last line must be the prompt with input: %q", lines[9])
	}
}

func TestReplHistory(t *testing.T) {
	rv := newRepl()
	rv.input = "toc"
	rv.Submit()
	rv.input = "status"
	rv.Submit()
	rv.HistoryPrev() // -> status
	if rv.input != "status" {
		t.Fatalf("history prev 1 = %q, want status", rv.input)
	}
	rv.HistoryPrev() // -> toc
	if rv.input != "toc" {
		t.Fatalf("history prev 2 = %q, want toc", rv.input)
	}
	rv.HistoryNext() // -> status
	if rv.input != "status" {
		t.Fatalf("history next = %q, want status", rv.input)
	}
}

// TestReplPrevInChapter verifies that after two next submits, a prev moves
// back one paragraph (Progress().Para decrements by one).
func TestReplPrevInChapter(t *testing.T) {
	rv := newRepl()
	rv.input = "next"
	rv.Submit() // emits 甲甲甲, para=0, Progress.Para=1
	rv.input = "next"
	rv.Submit() // emits 乙乙乙, para=1, Progress.Para=2
	rv.input = "prev"
	rv.Submit() // re-emits 甲甲甲, para=0, Progress.Para=1
	joined := strings.Join(rv.Render(), "\n")
	if !strings.Contains(joined, "甲甲甲") {
		t.Fatalf("prev should re-emit previous paragraph:\n%s", joined)
	}
	if got := rv.Progress().Para; got != 1 {
		t.Fatalf("prev should leave Progress.Para=1, got %d", got)
	}
}

// TestReplPrevAtBOF verifies that prev at the very start emits "-- BOF --".
func TestReplPrevAtBOF(t *testing.T) {
	rv := newRepl()
	rv.input = "prev"
	rv.Submit()
	joined := strings.Join(rv.Render(), "\n")
	if !strings.Contains(joined, "-- BOF --") {
		t.Fatalf("prev at beginning-of-book should print -- BOF --:\n%s", joined)
	}
}

// TestReplGitLogAlias verifies that "git log" advances like next.
func TestReplGitLogAlias(t *testing.T) {
	rv := newRepl()
	rv.input = "git log"
	rv.Submit()
	joined := strings.Join(rv.Render(), "\n")
	if !strings.Contains(joined, "甲甲甲") {
		t.Fatalf("git log should emit first paragraph:\n%s", joined)
	}
	if got := rv.Progress().Para; got != 1 {
		t.Fatalf("git log should advance Progress.Para to 1, got %d", got)
	}
}

// TestReplGitStatusAlias verifies that "git status" prints a progress line
// containing "on chapter".
func TestReplGitStatusAlias(t *testing.T) {
	rv := newRepl()
	rv.input = "git status"
	rv.Submit()
	joined := strings.Join(rv.Render(), "\n")
	if !strings.Contains(joined, "on chapter") {
		t.Fatalf("git status should print progress line with 'on chapter':\n%s", joined)
	}
}

// TestReplResumePositionClamp verifies that a Progress pointing to para 0 of
// chapter 1 (the last chapter in replBook) causes the next emit to show that
// chapter's first paragraph ("丁丁丁") and leaves Progress().Chapter == 1.
func TestReplResumePositionClamp(t *testing.T) {
	rv := NewReplView(replBook(), store.Progress{Chapter: 1, Para: 0}, store.Prefs{Style: "log"}, 60, 10)
	rv.input = "next"
	rv.Submit()
	joined := strings.Join(rv.Render(), "\n")
	if !strings.Contains(joined, "丁丁丁") {
		t.Fatalf("resume at chapter 1 para 0 should emit '丁丁丁':\n%s", joined)
	}
	if got := rv.Progress().Chapter; got != 1 {
		t.Fatalf("Progress().Chapter should be 1, got %d", got)
	}
}

// scrollBook has enough short paragraphs that the rendered scrollback overflows
// a small window, so scrolling can be exercised.
func scrollBook() *book.Book {
	return &book.Book{Title: "S", Chapters: []book.Chapter{
		{Title: "c", Paragraphs: []string{"一", "二", "三", "四", "五", "六"}},
	}}
}

func TestReplScrollbackUpDown(t *testing.T) {
	rv := NewReplView(scrollBook(), store.Progress{}, store.Prefs{Style: "log"}, 60, 5) // bodyH=4
	for i := 0; i < 6; i++ {
		rv.input = "next"
		rv.Submit()
	}
	bottom := strings.Join(rv.Render(), "\n")

	rv.ScrollUp(3)
	up := strings.Join(rv.Render(), "\n")
	if up == bottom {
		t.Fatalf("ScrollUp should change the visible window:\n%s", up)
	}
	if len(rv.Render()) != 5 {
		t.Fatalf("Render must stay height(5) while scrolled, got %d", len(rv.Render()))
	}

	rv.ScrollDown(100) // clamp back to bottom
	if got := strings.Join(rv.Render(), "\n"); got != bottom {
		t.Fatalf("ScrollDown past bottom should match original bottom view:\nwant:\n%s\ngot:\n%s", bottom, got)
	}
}

func TestReplSubmitSnapsToBottom(t *testing.T) {
	rv := NewReplView(scrollBook(), store.Progress{}, store.Prefs{Style: "log"}, 60, 5)
	for i := 0; i < 6; i++ {
		rv.input = "next"
		rv.Submit()
	}
	bottom := strings.Join(rv.Render(), "\n")
	rv.ScrollUp(3)
	if strings.Join(rv.Render(), "\n") == bottom {
		t.Fatalf("precondition: ScrollUp should move away from bottom")
	}
	rv.input = "status"
	rv.Submit() // any command output should snap the view back to the bottom
	if rv.scrollOff != 0 {
		t.Fatalf("Submit should reset scrollOff to 0, got %d", rv.scrollOff)
	}
}

func TestReplScrollUpClampsToContent(t *testing.T) {
	// few lines (well under the window) -> nothing to scroll
	rv := NewReplView(scrollBook(), store.Progress{}, store.Prefs{Style: "log"}, 60, 20)
	rv.input = "next"
	rv.Submit()
	rv.ScrollUp(50)
	if rv.scrollOff != 0 {
		t.Fatalf("ScrollUp with no overflow should stay at 0, got %d", rv.scrollOff)
	}
}
