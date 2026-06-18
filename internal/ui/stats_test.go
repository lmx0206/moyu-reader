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
