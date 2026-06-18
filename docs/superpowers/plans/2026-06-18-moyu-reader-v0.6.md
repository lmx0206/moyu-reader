# moyu-reader v0.6 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fake-shell REPL reading mode and a "slacking stats" coverage panel to moyu-reader.

**Architecture:** Two independent components. **B. Stats** (Tasks 1–3): pure store/book functions accumulate reading time, char high-water and streak into `library.json`, surfaced by a `StatsView` disguised as a `coverage report`. **A. REPL** (Tasks 4–5): a new `ReplView` renders a typed fake shell where commands advance reading one paragraph at a time, wired into the reader as a third `m`-cycle mode. Tasks 6–7 update docs and verify.

**Tech Stack:** Go 1.26, bubbletea v1.3.x + lipgloss v1.1.x, go-runewidth (via `internal/render`). Module `moyureader`.

## Global Constraints

- Disguise stealth: nothing on screen may leak the app name, book titles into disguised CLI output, or read as "a novel reader". REPL output looks like shell/log output; the stats panel looks like a coverage report.
- Reading positions are paragraph-anchored: `store.Progress{Chapter int, Para int}`. Never anchor on display-line index.
- All display-width math uses `render.StringWidth` / `render.RuneWidth` / `render.PadRight` (CJK is double-width); never `len()` for column widths.
- bubbletea v1 (`github.com/charmbracelet/bubbletea`), key handling via `tea.KeyMsg`.
- Go toolchain is not on PATH in this environment: prefix every go command with `export PATH="/d/develop/Go/bin:$PATH"`.
- Backward compatible persistence: all new JSON fields use `omitempty`; old `library.json` needs no migration.
- TDD: failing test first; one logical change per commit. Commit author: `git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "..."` with footer `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`.

---

## Task 1: Stats data model + pure accumulation functions

**Files:**
- Create: `internal/book/chars.go`
- Test: `internal/book/chars_test.go`
- Modify: `internal/store/model.go` (add fields to `BookEntry` and `Library`)
- Create: `internal/store/stats.go`
- Test: `internal/store/stats_test.go`

**Interfaces:**
- Produces:
  - `book.TotalChars(b *book.Book) int`
  - `book.CharsUpTo(b *book.Book, chapter, para int) int`
  - `store.Stats` struct with `TotalSeconds, TodaySeconds, StreakDays int` and `LastReadDate string`
  - `store.RecordActivity(lib *store.Library, now time.Time, seconds int)`
  - `store.RecordReading(lib *store.Library, id string, chapter, para, charsUpTo int)`
  - `store.BookEntry` gains `TotalChars, CharsRead, FurthestChapter, FurthestPara int`
  - `store.Library` gains `Stats Stats`

- [ ] **Step 1: Write failing tests for book char counting**

Create `internal/book/chars_test.go`:
```go
package book

import "testing"

func charBook() *Book {
	return &Book{
		Title: "T",
		Chapters: []Chapter{
			{Title: "c1", Paragraphs: []string{"ab", "cde"}},   // 2 + 3 = 5
			{Title: "c2", Paragraphs: []string{"文文文", "x"}}, // 3 + 1 = 4
		},
	}
}

func TestTotalChars(t *testing.T) {
	if got := TotalChars(charBook()); got != 9 {
		t.Fatalf("TotalChars = %d, want 9", got)
	}
}

func TestCharsUpTo(t *testing.T) {
	b := charBook()
	// before (0,0): nothing read
	if got := CharsUpTo(b, 0, 0); got != 0 {
		t.Fatalf("CharsUpTo(0,0) = %d, want 0", got)
	}
	// before (0,1): paragraph 0 of chapter 0 -> "ab" = 2
	if got := CharsUpTo(b, 0, 1); got != 2 {
		t.Fatalf("CharsUpTo(0,1) = %d, want 2", got)
	}
	// before (1,1): all of chapter 0 (5) + paragraph 0 of chapter 1 ("文文文"=3) = 8
	if got := CharsUpTo(b, 1, 1); got != 8 {
		t.Fatalf("CharsUpTo(1,1) = %d, want 8", got)
	}
	// out-of-range clamps to the end (everything) = 9
	if got := CharsUpTo(b, 9, 9); got != 9 {
		t.Fatalf("CharsUpTo(9,9) = %d, want 9", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/book/ -run "TestTotalChars|TestCharsUpTo"`
Expected: FAIL (`undefined: TotalChars` / `CharsUpTo`).

- [ ] **Step 3: Implement book char helpers**

Create `internal/book/chars.go`:
```go
package book

import "unicode/utf8"

// TotalChars returns the total rune count of all paragraph text in b.
func TotalChars(b *Book) int {
	n := 0
	for _, ch := range b.Chapters {
		for _, p := range ch.Paragraphs {
			n += utf8.RuneCountInString(p)
		}
	}
	return n
}

// CharsUpTo returns the rune count of all paragraphs strictly before position
// (chapter, para): every paragraph in chapters before `chapter`, plus
// paragraphs[:para] of `chapter`. chapter/para are clamped to valid ranges, so
// an out-of-range position counts the whole book.
func CharsUpTo(b *Book, chapter, para int) int {
	if chapter < 0 {
		return 0
	}
	if chapter >= len(b.Chapters) {
		return TotalChars(b)
	}
	n := 0
	for i := 0; i < chapter; i++ {
		for _, p := range b.Chapters[i].Paragraphs {
			n += utf8.RuneCountInString(p)
		}
	}
	ps := b.Chapters[chapter].Paragraphs
	if para > len(ps) {
		para = len(ps)
	}
	for i := 0; i < para; i++ {
		n += utf8.RuneCountInString(ps[i])
	}
	return n
}
```

- [ ] **Step 4: Run book tests to verify they pass**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/book/ -run "TestTotalChars|TestCharsUpTo"`
Expected: PASS.

- [ ] **Step 5: Add stats fields to store model**

In `internal/store/model.go`, replace the `BookEntry` struct's tail:
```go
	Progress     Progress     `json:"progress"`
	Prefs        Prefs        `json:"prefs"`
	Annotations  []Annotation `json:"annotations,omitempty"`
	Broken       bool         `json:"broken,omitempty"`
}
```
with:
```go
	Progress        Progress     `json:"progress"`
	Prefs           Prefs        `json:"prefs"`
	Annotations     []Annotation `json:"annotations,omitempty"`
	TotalChars      int          `json:"totalChars,omitempty"`
	CharsRead       int          `json:"charsRead,omitempty"`
	FurthestChapter int          `json:"furthestChapter,omitempty"`
	FurthestPara    int          `json:"furthestPara,omitempty"`
	Broken          bool         `json:"broken,omitempty"`
}
```

In `internal/store/model.go`, replace the `Library` struct:
```go
// Library is the whole bookshelf plus global defaults.
type Library struct {
	LastBookID string      `json:"lastBookId"`
	Global     Prefs       `json:"global"`
	Books      []BookEntry `json:"books"`
}
```
with:
```go
// Library is the whole bookshelf plus global defaults.
type Library struct {
	LastBookID string      `json:"lastBookId"`
	Global     Prefs       `json:"global"`
	Books      []BookEntry `json:"books"`
	Stats      Stats       `json:"stats,omitempty"`
}
```

- [ ] **Step 6: Write failing tests for store stats functions**

Create `internal/store/stats_test.go`:
```go
package store

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestRecordActivityStreak(t *testing.T) {
	lib := NewLibrary()
	// first ever day -> streak 1
	RecordActivity(lib, day("2026-06-18"), 60)
	if lib.Stats.StreakDays != 1 || lib.Stats.TotalSeconds != 60 || lib.Stats.TodaySeconds != 60 {
		t.Fatalf("day1: %+v", lib.Stats)
	}
	// same day again -> no streak change, time accumulates
	RecordActivity(lib, day("2026-06-18"), 30)
	if lib.Stats.StreakDays != 1 || lib.Stats.TodaySeconds != 90 {
		t.Fatalf("same day: %+v", lib.Stats)
	}
	// next consecutive day -> streak 2, today resets then adds
	RecordActivity(lib, day("2026-06-19"), 10)
	if lib.Stats.StreakDays != 2 || lib.Stats.TodaySeconds != 10 || lib.Stats.TotalSeconds != 100 {
		t.Fatalf("day2: %+v", lib.Stats)
	}
	// skip a day -> streak resets to 1
	RecordActivity(lib, day("2026-06-21"), 5)
	if lib.Stats.StreakDays != 1 || lib.Stats.TodaySeconds != 5 {
		t.Fatalf("gap: %+v", lib.Stats)
	}
}

func TestRecordReadingHighWater(t *testing.T) {
	lib := NewLibrary()
	lib.Books = append(lib.Books, BookEntry{ID: "x"})
	RecordReading(lib, "x", 0, 3, 100)
	if e := lib.FindByID("x"); e.FurthestPara != 3 || e.CharsRead != 100 {
		t.Fatalf("advance: %+v", e)
	}
	// going backward does NOT lower the high-water mark
	RecordReading(lib, "x", 0, 1, 40)
	if e := lib.FindByID("x"); e.FurthestPara != 3 || e.CharsRead != 100 {
		t.Fatalf("backward should not change: %+v", e)
	}
	// advancing into a later chapter updates it
	RecordReading(lib, "x", 1, 0, 150)
	if e := lib.FindByID("x"); e.FurthestChapter != 1 || e.FurthestPara != 0 || e.CharsRead != 150 {
		t.Fatalf("next chapter: %+v", e)
	}
	// unknown id is a no-op (no panic)
	RecordReading(lib, "nope", 9, 9, 999)
}
```

- [ ] **Step 7: Run store tests to verify they fail**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/store/ -run "TestRecordActivityStreak|TestRecordReadingHighWater"`
Expected: FAIL (`undefined: RecordActivity` / `RecordReading` / `Stats`).

- [ ] **Step 8: Implement store stats functions**

Create `internal/store/stats.go`:
```go
package store

import "time"

// Stats holds global slacking-reading statistics, persisted in library.json.
type Stats struct {
	TotalSeconds int    `json:"totalSeconds,omitempty"`
	TodaySeconds int    `json:"todaySeconds,omitempty"`
	LastReadDate string `json:"lastReadDate,omitempty"` // YYYY-MM-DD, local time
	StreakDays   int    `json:"streakDays,omitempty"`
}

// RecordActivity rolls the day/streak on the first activity of a new calendar
// day and adds `seconds` of reading time. Call it on every reading action;
// `seconds` may be 0 (e.g. the first action of a session, or after an idle gap).
func RecordActivity(lib *Library, now time.Time, seconds int) {
	today := now.Format("2006-01-02")
	s := &lib.Stats
	if s.LastReadDate != today {
		switch {
		case s.LastReadDate == "":
			s.StreakDays = 1
		case isPrevDay(s.LastReadDate, today):
			s.StreakDays++
		default:
			s.StreakDays = 1
		}
		s.TodaySeconds = 0
		s.LastReadDate = today
	}
	if seconds > 0 {
		s.TotalSeconds += seconds
		s.TodaySeconds += seconds
	}
}

// isPrevDay reports whether prev is exactly the calendar day before today
// (both "2006-01-02").
func isPrevDay(prev, today string) bool {
	t, err := time.Parse("2006-01-02", today)
	if err != nil {
		return false
	}
	return prev == t.AddDate(0, 0, -1).Format("2006-01-02")
}

// RecordReading advances a book's furthest-read high-water mark. When
// (chapter, para) is beyond the stored furthest position it updates the mark and
// sets CharsRead to charsUpTo (rune count from book start to that position).
// Backward moves and re-reads do not change anything. Unknown id is a no-op.
func RecordReading(lib *Library, id string, chapter, para, charsUpTo int) {
	e := lib.FindByID(id)
	if e == nil {
		return
	}
	if chapter > e.FurthestChapter || (chapter == e.FurthestChapter && para > e.FurthestPara) {
		e.FurthestChapter = chapter
		e.FurthestPara = para
		e.CharsRead = charsUpTo
	}
}
```

- [ ] **Step 9: Run all store + book tests**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/store/ ./internal/book/`
Expected: PASS.

- [ ] **Step 10: Commit**

```bash
git add internal/book/chars.go internal/book/chars_test.go internal/store/model.go internal/store/stats.go internal/store/stats_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(store): reading stats model + char-count helpers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: StatsView (coverage-report disguise, pure rendering)

**Files:**
- Create: `internal/ui/stats.go`
- Test: `internal/ui/stats_test.go`

**Interfaces:**
- Consumes: `store.Library` (with `Stats` and per-book `TotalChars`/`CharsRead` from Task 1).
- Produces: `ui.StatsView` (stateless) with `Render(lib *store.Library, width, height int) []string` returning exactly `height` lines.

- [ ] **Step 1: Write failing test**

Create `internal/ui/stats_test.go`:
```go
package ui

import (
	"strings"
	"testing"

	"moyureader/internal/store"
)

func TestStatsViewRendersCoverageReport(t *testing.T) {
	lib := store.NewLibrary()
	lib.Books = []store.BookEntry{
		{ID: "a", Title: "三体", File: "books/a.epub", TotalChars: 1000, CharsRead: 800},
		{ID: "b", Title: "活着", File: "books/b.txt", TotalChars: 500, CharsRead: 500},
	}
	lib.Stats = store.Stats{TotalSeconds: 3600*12 + 60*34, TodaySeconds: 3600 + 60*12, StreakDays: 7}

	out := (StatsView{}).Render(lib, 60, 20)
	if len(out) != 20 {
		t.Fatalf("Render must return height (20) lines, got %d", len(out))
	}
	joined := strings.Join(out, "\n")
	for _, want := range []string{"Cover", "三体", "活着", "TOTAL", "80%", "100%", "streak 7d", "12h34m", "today 1h12m"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("stats output missing %q:\n%s", want, joined)
		}
	}
}

func TestStatsViewEmptyNoPanic(t *testing.T) {
	out := (StatsView{}).Render(store.NewLibrary(), 60, 10)
	if len(out) != 10 {
		t.Fatalf("empty render must still be height (10), got %d", len(out))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestStatsView"`
Expected: FAIL (`undefined: StatsView`).

- [ ] **Step 3: Implement StatsView**

Create `internal/ui/stats.go`:
```go
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"moyureader/internal/render"
	"moyureader/internal/store"
)

// StatsView renders reading statistics disguised as a code-coverage report.
type StatsView struct{}

const statsNameWidth = 18

// Render returns exactly height lines of a coverage-styled stats report.
func (StatsView) Render(lib *store.Library, width, height int) []string {
	out := []string{
		render.PadRight("Name", statsNameWidth) + fmt.Sprintf("  %6s %6s  %5s", "Stmts", "Miss", "Cover"),
		strings.Repeat("-", statsNameWidth+22),
	}

	totalStmts, totalRead := 0, 0
	for _, b := range lib.Books {
		name := b.Title
		if name == "" {
			name = b.ID
		}
		stmts := b.TotalChars
		read := b.CharsRead
		if read > stmts {
			read = stmts
		}
		miss := stmts - read
		totalStmts += stmts
		totalRead += read
		out = append(out, statsRow(name, stmts, miss, coverPct(read, stmts)))
	}

	out = append(out, strings.Repeat("-", statsNameWidth+22))
	out = append(out, statsRow("TOTAL", totalStmts, totalStmts-totalRead, coverPct(totalRead, totalStmts)))
	out = append(out, "")
	out = append(out, fmt.Sprintf("elapsed %s · today %s · %s chars · streak %dd",
		formatDur(lib.Stats.TotalSeconds), formatDur(lib.Stats.TodaySeconds),
		humanChars(totalRead), lib.Stats.StreakDays))

	// pad/truncate to exactly height lines
	for len(out) < height {
		out = append(out, "")
	}
	if len(out) > height {
		out = out[:height]
	}
	return out
}

func statsRow(name string, stmts, miss, cover int) string {
	return render.PadRight(truncCells(name, statsNameWidth), statsNameWidth) +
		fmt.Sprintf("  %6d %6d  %4d%%", stmts, miss, cover)
}

func coverPct(read, stmts int) int {
	if stmts <= 0 {
		return 0
	}
	return read * 100 / stmts
}

func formatDur(sec int) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func humanChars(n int) string {
	if n >= 1000 {
		return strconv.Itoa(n/1000) + "k"
	}
	return strconv.Itoa(n)
}

// truncCells truncates s to at most w display cells (CJK-aware).
func truncCells(s string, w int) string {
	if render.StringWidth(s) <= w {
		return s
	}
	out := make([]rune, 0, len(s))
	used := 0
	for _, r := range s {
		rw := render.RuneWidth(r)
		if used+rw > w {
			break
		}
		out = append(out, r)
		used += rw
	}
	return string(out)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestStatsView"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/stats.go internal/ui/stats_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(ui): StatsView coverage-report disguise

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Wire stats into the model (collection + `c` screen)

**Files:**
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

**Interfaces:**
- Consumes: `store.RecordActivity`, `store.RecordReading`, `book.TotalChars`, `book.CharsUpTo` (Task 1); `StatsView` (Task 2).
- Produces: `screenStats` screen; `Model.lastActivity time.Time`; `Model.statsReturn screen`; `c` opens stats from shelf and reader.

- [ ] **Step 1: Write failing tests**

In `internal/ui/model_test.go`, append:
```go
func TestModelCOpensStatsFromReader(t *testing.T) {
	m := newReaderModel(t)
	m.lib.Books = append(m.lib.Books, store.BookEntry{ID: "x"})
	nm, _ := m.Update(keyRunes("c"))
	m = nm.(*Model)
	if m.screen != screenStats {
		t.Fatalf("c should open stats, got %v", m.screen)
	}
	if !contains(m.View(), "Cover") {
		t.Fatalf("stats view should show coverage report:\n%s", m.View())
	}
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(*Model)
	if m.screen != screenReader {
		t.Fatalf("esc should return to reader, got %v", m.screen)
	}
}

func TestModelRecordsCharsReadOnPageDown(t *testing.T) {
	m := newReaderModel(t)                              // sampleBook, bookID "x"
	m.book = sampleBook()                               // ensure book set
	m.lib.Books = append(m.lib.Books, store.BookEntry{ID: "x"})
	m.openBookForTest()                                 // sets TotalChars
	nm, _ := m.Update(keyRunes("f"))                    // page down -> saveProgress
	m = nm.(*Model)
	e := m.lib.FindByID("x")
	if e.TotalChars == 0 {
		t.Fatalf("TotalChars should be set on open")
	}
	if e.CharsRead <= 0 || e.FurthestPara == 0 {
		t.Fatalf("paging down should advance high-water: %+v", e)
	}
}
```

Note: `openBookForTest` is a tiny helper added in Step 3 so the test can set `TotalChars` without a real file. The `newReaderModel` helper already sets `book`, `reader`, `bookID="x"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestModelCOpensStatsFromReader|TestModelRecordsCharsReadOnPageDown"`
Expected: FAIL (`undefined: screenStats` / `openBookForTest`).

- [ ] **Step 3: Add screen, fields, and helpers in model.go**

In `internal/ui/model.go`, replace the screen const block:
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
	screenStats
)
```

In the `Model` struct, replace:
```go
	toc        *TOCView
	annot      *AnnotationView
	helpReturn screen
```
with:
```go
	toc         *TOCView
	annot       *AnnotationView
	helpReturn  screen
	statsReturn screen
```
and replace:
```go
	importBuf string // typed path in import screen
	annotBuf  string // typed note in annotate screen
	status    string // transient status line (errors etc.)
```
with:
```go
	importBuf    string    // typed path in import screen
	annotBuf     string    // typed note in annotate screen
	status       string    // transient status line (errors etc.)
	lastActivity time.Time // for accumulating reading time
```

Add the `version` import is not needed here. Add `"moyureader/internal/book"` is already imported. Add a stats-collection helper and a test helper at the end of `internal/ui/model.go`:
```go
// recordActivity accumulates reading time, attributing the gap since the last
// reading action (capped at 5 minutes so walking away is not counted) and
// rolling the daily streak.
func (m *Model) recordActivity() {
	now := time.Now()
	secs := 0
	if !m.lastActivity.IsZero() {
		if d := now.Sub(m.lastActivity); d > 0 && d <= 5*time.Minute {
			secs = int(d.Seconds())
		}
	}
	store.RecordActivity(m.lib, now, secs)
	m.lastActivity = now
}

// openBookForTest seeds TotalChars from the in-memory book without a real file
// (used by tests; mirrors the TotalChars seeding that openBook does).
func (m *Model) openBookForTest() {
	if e := m.lib.FindByID(m.bookID); e != nil && m.book != nil && e.TotalChars == 0 {
		e.TotalChars = book.TotalChars(m.book)
	}
}
```

- [ ] **Step 4: Seed TotalChars in openBook and record reading in saveProgress**

In `internal/ui/model.go`, in `openBook`, replace:
```go
	m.reader = NewReaderView(bk, e.Progress, e.Prefs, m.width, m.height)
	m.book = bk
	m.bookID = id
	m.screen = screenReader
```
with:
```go
	if e.TotalChars == 0 {
		e.TotalChars = book.TotalChars(bk)
	}
	m.reader = NewReaderView(bk, e.Progress, e.Prefs, m.width, m.height)
	m.book = bk
	m.bookID = id
	m.lastActivity = time.Time{}
	m.screen = screenReader
```

Replace `saveProgress`:
```go
// saveProgress persists the current reader position + prefs.
func (m *Model) saveProgress() {
	if m.reader == nil || m.bookID == "" {
		return
	}
	store.UpdateProgress(m.lib, m.bookID, m.reader.Progress(), m.reader.Prefs())
	_ = m.st.Save(m.lib)
}
```
with:
```go
// saveProgress persists the current reader position + prefs and advances the
// per-book reading high-water mark.
func (m *Model) saveProgress() {
	if m.reader == nil || m.bookID == "" {
		return
	}
	p := m.reader.Progress()
	store.UpdateProgress(m.lib, m.bookID, p, m.reader.Prefs())
	if m.book != nil {
		store.RecordReading(m.lib, m.bookID, p.Chapter, p.Para, book.CharsUpTo(m.book, p.Chapter, p.Para))
	}
	_ = m.st.Save(m.lib)
}
```

- [ ] **Step 5: Route `c`, add handlers, record activity, and View**

In `handleKey`, replace:
```go
	case screenAnnotList:
		return m.handleAnnotListKey(key)
	}
```
with:
```go
	case screenAnnotList:
		return m.handleAnnotListKey(key)
	case screenStats:
		return m.handleStatsKey(key)
	}
```

In `handleReaderKey`, add a `recordActivity` call at the very top and a `c` case. Replace:
```go
func (m *Model) handleReaderKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "esc":
```
with:
```go
func (m *Model) handleReaderKey(key string) (tea.Model, tea.Cmd) {
	m.recordActivity()
	switch key {
	case "c":
		m.statsReturn = screenReader
		m.screen = screenStats
	case "q", "esc":
```

In `handleShelfKey`, add a `c` case. Replace:
```go
	case "d":
		m.deleteSelected()
	}
	return m, nil
}
```
with:
```go
	case "d":
		m.deleteSelected()
	case "c":
		m.statsReturn = screenShelf
		m.screen = screenStats
	}
	return m, nil
}
```

Add `handleStatsKey` right after `handleAnnotListKey`:
```go
func (m *Model) handleStatsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q", "c":
		m.screen = m.statsReturn
	}
	return m, nil
}
```

In `View`, replace:
```go
	case screenImport:
		return "导入 EPUB（粘贴 .epub 完整路径后回车，Esc 取消）:\n\n> " + m.importBuf + "\n\n" + m.status
```
with:
```go
	case screenImport:
		return "导入 EPUB（粘贴 .epub 完整路径后回车，Esc 取消）:\n\n> " + m.importBuf + "\n\n" + m.status
	case screenStats:
		return strings.Join(paintDim((StatsView{}).Render(m.lib, m.width, m.height)), "\n")
```

- [ ] **Step 6: Run the model tests**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestModelCOpensStatsFromReader|TestModelRecordsCharsReadOnPageDown"`
Expected: PASS.

- [ ] **Step 7: Run full suite + vet**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go vet ./... && go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(ui): collect reading stats + 'c' coverage screen

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: ReplView — fake-shell reading core

**Files:**
- Create: `internal/ui/repl.go`
- Test: `internal/ui/repl_test.go`

**Interfaces:**
- Consumes: `book.Book`, `store.Progress`/`store.Prefs`, `disguise.ThemeByName`/`PrefixWidth`/`Theme.LinePrefix`, `render.WrapParagraph`; package helpers `orDefault`, `clamp` (defined in `reader.go`).
- Produces: `ui.ReplView` with:
  - `NewReplView(b *book.Book, p store.Progress, prefs store.Prefs, width, height int) *ReplView`
  - `(*ReplView).SetSize(width, height int)`
  - `(*ReplView).Render() []string` (exactly height lines; prompt on last line)
  - `(*ReplView).Submit()` / `Insert(s string)` / `Backspace()` / `HistoryPrev()` / `HistoryNext()`
  - `(*ReplView).Progress() store.Progress` / `Prefs() store.Prefs`
  - exported-within-package field `quit bool` (set true by `q`/`exit`)

- [ ] **Step 1: Write failing tests**

Create `internal/ui/repl_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestRepl"`
Expected: FAIL (`undefined: NewReplView`).

- [ ] **Step 3: Implement ReplView**

Create `internal/ui/repl.go`:
```go
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
	rv.histPos = 0
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

func (rv *ReplView) Insert(s string)  { rv.input += s }
func (rv *ReplView) Backspace()        { if n := len(rv.input); n > 0 { rv.input = rv.input[:n-1] } }

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

// Render returns exactly height lines: the tail of the scrollback bottom-aligned
// above a final prompt line.
func (rv *ReplView) Render() []string {
	bodyH := rv.height - 1
	if bodyH < 0 {
		bodyH = 0
	}
	start := 0
	if len(rv.scroll) > bodyH {
		start = len(rv.scroll) - bodyH
	}
	visible := rv.scroll[start:]

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
```

- [ ] **Step 4: Run repl tests to verify they pass**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestRepl"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/repl.go internal/ui/repl_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(ui): ReplView fake-shell reading core

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Wire the REPL mode into the model (`m` cycle)

**Files:**
- Modify: `internal/ui/model.go`
- Test: `internal/ui/model_test.go`

**Interfaces:**
- Consumes: `ReplView` (Task 4); existing `recordActivity`/`saveProgress` (Task 3).
- Produces: `Model.repl *ReplView`; `m` cycles shell→inline→repl→shell; REPL key routing; boss/esc/ctrl+c handling in REPL.

- [ ] **Step 1: Write failing tests**

In `internal/ui/model_test.go`, append:
```go
func TestModelMCyclesIntoRepl(t *testing.T) {
	m := newReaderModel(t) // starts shell
	nm, _ := m.Update(keyRunes("m"))
	m = nm.(*Model)
	if m.reader.Prefs().Mode != "inline" || m.repl != nil {
		t.Fatalf("first m should go shell->inline, got mode=%q repl=%v", m.reader.Prefs().Mode, m.repl)
	}
	nm, _ = m.Update(keyRunes("m"))
	m = nm.(*Model)
	if m.repl == nil {
		t.Fatalf("second m should enter repl")
	}
	// typing inside repl feeds the buffer, not the page
	nm, _ = m.Update(keyRunes("toc"))
	m = nm.(*Model)
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(*Model)
	if !contains(m.View(), "第二章") {
		t.Fatalf("repl 'toc' should list chapters:\n%s", m.View())
	}
	// m again leaves repl back to shell reading
	nm, _ = m.Update(keyRunes("m"))
	m = nm.(*Model)
	if m.repl != nil || m.reader.Prefs().Mode != "shell" {
		t.Fatalf("m in repl should return to shell, repl=%v mode=%q", m.repl, m.reader.Prefs().Mode)
	}
}

func TestModelReplBossAndEsc(t *testing.T) {
	m := newReaderModel(t)
	m.repl = NewReplView(m.book, store.Progress{}, store.Prefs{Style: "log"}, 40, 12)
	// backtick still triggers boss even in repl
	nm, _ := m.Update(keyRunes("`"))
	m = nm.(*Model)
	if !m.bossActive {
		t.Fatalf("backtick should trigger boss in repl")
	}
	nm, _ = m.Update(keyRunes("`"))
	m = nm.(*Model)
	// esc in repl returns to shelf
	nm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(*Model)
	if m.screen != screenShelf || m.repl != nil {
		t.Fatalf("esc in repl should go to shelf, screen=%v repl=%v", m.screen, m.repl)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestModelMCyclesIntoRepl|TestModelReplBossAndEsc"`
Expected: FAIL (`m.repl undefined`).

- [ ] **Step 3: Add the repl field**

In `internal/ui/model.go`, in the `Model` struct, replace:
```go
	shelf  *ShelfView
	reader *ReaderView
	book   *book.Book
	bookID string
```
with:
```go
	shelf  *ShelfView
	reader *ReaderView
	repl   *ReplView
	book   *book.Book
	bookID string
```

- [ ] **Step 4: Route repl keys and render repl**

In `handleKey`, immediately after the `switch m.screen { ... }` block that dispatches the overlay screens (the block ending with the `screenStats` case from Task 3) and BEFORE the `if key == "?"` line, insert:
```go
	// REPL captures typed input, so route it before the generic ?/boss/b checks.
	if m.screen == screenReader && m.repl != nil {
		return m.handleReplKey(msg)
	}
```

In `Update`'s `tea.WindowSizeMsg` case, replace:
```go
		m.width, m.height = msg.Width, msg.Height
		if m.reader != nil {
			m.reader.SetSize(msg.Width, msg.Height)
		}
		return m, nil
```
with:
```go
		m.width, m.height = msg.Width, msg.Height
		if m.reader != nil {
			m.reader.SetSize(msg.Width, msg.Height)
		}
		if m.repl != nil {
			m.repl.SetSize(msg.Width, msg.Height)
		}
		return m, nil
```

In `View`, replace:
```go
	case screenReader:
		lines := m.reader.Render()
		if m.reader.Prefs().Mode == "shell" {
			return strings.Join(paintShell(lines), "\n")
		}
		return strings.Join(paintDim(lines), "\n")
```
with:
```go
	case screenReader:
		if m.repl != nil {
			return strings.Join(paintDim(m.repl.Render()), "\n")
		}
		lines := m.reader.Render()
		if m.reader.Prefs().Mode == "shell" {
			return strings.Join(paintShell(lines), "\n")
		}
		return strings.Join(paintDim(lines), "\n")
```

- [ ] **Step 5: Change the `m` key to a three-way cycle + add handleReplKey**

In `handleReaderKey`, replace:
```go
	case "m":
		m.reader.ToggleMode()
```
with:
```go
	case "m":
		m.cycleMode()
```

Add `cycleMode` and `handleReplKey` right after `handleReaderKey`:
```go
// cycleMode advances the reading presentation shell -> inline -> repl. (The
// repl -> shell leg is handled inside handleReplKey, since the 'm' key is
// intercepted there while the REPL is active.)
func (m *Model) cycleMode() {
	if m.book == nil {
		m.reader.ToggleMode()
		return
	}
	if m.reader.Prefs().Mode == "inline" {
		m.repl = NewReplView(m.book, m.reader.Progress(), m.reader.Prefs(), m.width, m.height)
		return
	}
	m.reader.ToggleMode() // shell -> inline
}

func (m *Model) handleReplKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.recordActivity()
	switch msg.String() {
	case "`":
		m.bossActive = true
		m.bossTick = 0
		return m, bossTick()
	case "m":
		// leave repl, resume shell-mode reading at the same paragraph
		p := m.repl.Progress()
		m.reader.JumpToPara(p.Chapter, p.Para)
		if m.reader.Prefs().Mode != "shell" {
			m.reader.ToggleMode() // inline -> shell
		}
		m.repl = nil
		m.saveProgress()
	case "esc":
		m.replExitToShelf()
	case "ctrl+c":
		m.replSyncProgress()
		m.saveProgress()
		return m, tea.Quit
	case "enter":
		m.repl.Submit()
		m.replSyncProgress()
		m.saveProgress()
		if m.repl.quit {
			m.replExitToShelf()
		}
	case "backspace":
		m.repl.Backspace()
	case "up":
		m.repl.HistoryPrev()
	case "down":
		m.repl.HistoryNext()
	default:
		if msg.Type == tea.KeyRunes {
			m.repl.Insert(string(msg.Runes))
		} else if msg.Type == tea.KeySpace {
			m.repl.Insert(" ")
		}
	}
	return m, nil
}

// replSyncProgress copies the repl position into the reader so saveProgress and
// later modes see the latest position.
func (m *Model) replSyncProgress() {
	if m.repl != nil && m.reader != nil {
		p := m.repl.Progress()
		m.reader.JumpToPara(p.Chapter, p.Para)
	}
}

func (m *Model) replExitToShelf() {
	m.replSyncProgress()
	m.saveProgress()
	m.repl = nil
	m.screen = screenShelf
	m.shelf = NewShelfView(m.lib)
}
```

- [ ] **Step 6: Run the model tests**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/ui/ -run "TestModelMCyclesIntoRepl|TestModelReplBossAndEsc"`
Expected: PASS.

- [ ] **Step 7: Run full suite + vet**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go vet ./... && go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "feat(ui): wire REPL as third reading mode (m cycle)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: README — command-line mode, stats, keybindings, roadmap

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add features**

In `README.md`, replace:
```markdown
- 🔖 **书签 / 笔记**：`a` 在当前位置加书签（笔记可留空）或写批注，`l` 打开标注列表（伪装成调试器「断点面板」`breakpoints (N)`），回车跳转、`d` 删除
```
with:
```markdown
- 🔖 **书签 / 笔记**：`a` 在当前位置加书签（笔记可留空）或写批注，`l` 打开标注列表（伪装成调试器「断点面板」`breakpoints (N)`），回车跳转、`d` 删除
- ⌨️ **命令行阅读模式**：第三种阅读模式（`m` 循环到它），全屏伪 shell——敲命令一段段往下读，正文作为「命令输出」滚动打印
- 📈 **摸鱼统计**：`c` 打开伪装成 `coverage report` 的统计面板（每本书进度、累计/今日时长、已读字数、连续摸鱼天数）
```

- [ ] **Step 2: Update the reading keybinding table**

In `README.md`, replace:
```markdown
| `m` | 切阅读模式（外壳 / 内联） |
| `g` | 目录跳转 |
| `a` | 加书签 / 笔记 |
| `l` | 标注列表（回车跳转，`d` 删除） |
```
with:
```markdown
| `m` | 切阅读模式（外壳 → 内联 → 命令行 循环） |
| `g` | 目录跳转 |
| `a` | 加书签 / 笔记 |
| `l` | 标注列表（回车跳转，`d` 删除） |
| `c` | 摸鱼统计面板 |
```

- [ ] **Step 3: Add a command-line mode section**

In `README.md`, after the "内联流式模式" table block:
```markdown
### 内联流式模式
| 输入 | 作用 |
|------|------|
| 回车 | 下一段 |
| `b` | 回退 |
| `t` | 切风格 |
| `q` | 退出 |
```
insert:
```markdown
### 命令行阅读模式（按 `m` 循环进入）
全屏伪 shell，提示符在底部，敲命令驱动阅读。命令不区分大小写：

| 输入 | 作用 |
|------|------|
| 回车 / `n` / `next` / `git log` | 下一段 |
| `p` / `prev` | 上一段 |
| `toc` / `ls` | 列出章节 |
| `cd <章号>` | 跳到第 N 章（无效 → `cd: no such chapter`） |
| `status` / `git status` | 阅读进度 |
| `clear` / `cls` | 清屏 |
| `help` / `?` | 命令清单 |
| `q` / `exit` | 回书架 |
| 其他 | `command not found` |
| `` ` `` | 老板键（命令行模式下仍可用） |
| `Esc` | 回书架 |
```

- [ ] **Step 4: Tick the roadmap**

In `README.md`, replace:
```markdown
- [ ] **v0.6** 假命令行（在终端里假装执行命令）+ 摸鱼统计（阅读时长 / 字数 / 进度）
```
with:
```markdown
- [x] **v0.6** 命令行阅读模式（全屏伪 shell）+ 摸鱼统计（时长 / 字数 / 进度 / streak）
```

- [ ] **Step 5: Commit**

```bash
git add README.md
git -c user.name="Claude" -c user.email="noreply@anthropic.com" commit -m "docs: README for v0.6 (command-line mode + stats)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Full verification + visual smoke check

**Files:** temporary `cmd/_frames/main.go` (deleted before finishing; not committed)

- [ ] **Step 1: Full vet + test**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go vet ./... && go test ./...`
Expected: all PASS.

- [ ] **Step 2: Write a temporary frame harness**

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
	b := &book.Book{Title: "L", Chapters: []book.Chapter{
		{Title: "第一章", Paragraphs: []string{
			"曾经有个乞丐在路边坐了三十多年。一天，一位陌生人经过。",
			"乞丐机械地举起他的旧棒球帽，喃喃地说：我没有什么东西可以给你。",
		}},
		{Title: "第二章", Paragraphs: []string{"陌生人问：你坐着的是什么？"}},
	}}

	fmt.Println("===== 命令行阅读模式 =====")
	rv := ui.NewReplView(b, store.Progress{}, store.Prefs{Style: "log", Mode: "repl"}, 70, 12)
	for _, cmd := range []string{"toc", "next", "next", "status"} {
		rv.Insert(cmd)
		rv.Submit()
	}
	fmt.Println(strings.Join(rv.Render(), "\n"))

	fmt.Println("\n===== 摸鱼统计（coverage 伪装）=====")
	lib := store.NewLibrary()
	lib.Books = []store.BookEntry{
		{ID: "a", Title: "三体.epub", TotalChars: 1500, CharsRead: 1200},
		{ID: "b", Title: "活着.txt", TotalChars: 500, CharsRead: 500},
	}
	lib.Stats = store.Stats{TotalSeconds: 3600*12 + 60*34, TodaySeconds: 3600 + 60*12, StreakDays: 7}
	fmt.Println(strings.Join((ui.StatsView{}).Render(lib, 64, 12), "\n"))
}
```

- [ ] **Step 3: Run the harness and eyeball**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go run ./cmd/_frames/`
Expected checkpoints:
- REPL: a `PS C:\proj> toc` echo + chapter list; `next` lines prefixed like logs (`[..] INFO ... - 曾经有个乞丐…`); a `status` line `on chapter 1/2 · 0%`; the prompt `PS C:\proj> ` on the last line.
- Stats: a `Name / Stmts / Miss / Cover` header, two book rows (三体 80%, 活着 100%), a `TOTAL` row, and a summary `elapsed 12h34m · today 1h12m · ... chars · streak 7d`.

- [ ] **Step 4: Delete the temporary harness**

Run: `rm -rf cmd/_frames`
Then: `git status --short` (should be clean — `cmd/_frames` gone, nothing staged).

- [ ] **Step 5: Final full run**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go vet ./... && go test ./...`
Expected: PASS. No commit (harness deleted, no residual changes).

---

## Self-Review (plan author checked)

- **Spec coverage:** ① REPL third mode via `m` (Task 5) ✓ ② full-screen scrollback + bottom prompt (Task 4 Render) ✓ ③ command vocabulary next/prev/toc/cd/status/clear/help/quit + dev aliases + `command not found` (Task 4 run) ✓ ④ one-paragraph advance, paragraph-anchored (Task 4 next/Progress) ✓ ⑤ `` ` `` boss + Esc/Ctrl+C in repl (Task 5 handleReplKey) ✓ ⑥ stats: time w/ 5-min idle cap + streak (Task 1 RecordActivity), chars high-water (Task 1 RecordReading), coverage panel (Task 2), `c` from shelf+reader (Task 3) ✓ ⑦ TotalChars seeded on open (Task 3) ✓ ⑧ README incl. command list (Task 6) ✓ ⑨ verification harness (Task 7) ✓.
- **Placeholder scan:** none; every code step is complete code or an exact old→new replacement.
- **Type consistency:** `store.Stats`, `store.RecordActivity`/`RecordReading`, `book.TotalChars`/`CharsUpTo`, `ReplView.{Submit,Insert,Backspace,HistoryPrev,HistoryNext,Progress,Prefs,SetSize,Render,quit}`, `StatsView.Render`, `Model.{repl,lastActivity,statsReturn,recordActivity,cycleMode,handleReplKey,handleStatsKey,replSyncProgress,replExitToShelf}` referenced consistently across tasks.
- **Known trade-offs (from spec):** old `Progress.line` not migrated (already shipped in v0.5); char high-water counts skipped paragraphs on forward `cd` (intentional approximation); REPL `m`/`` ` `` are reserved keys (no command contains those characters).
- **Build order:** Tasks 1→2→3 (stats) then 4→5 (repl) each leave the repo green; Task 5 depends on Task 4's `ReplView`; Task 3's `recordActivity`/`saveProgress` are reused by Task 5.
```
