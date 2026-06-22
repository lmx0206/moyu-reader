package ui

import (
	"strings"
	"testing"

	"moyureader/internal/store"
)

func TestShelfOrdersLastReadFirst(t *testing.T) {
	lib := &store.Library{
		LastBookID: "b",
		Books: []store.BookEntry{
			{ID: "a", Title: "A", LastOpenedAt: "2026-01-01T00:00:00Z"},
			{ID: "b", Title: "B", LastOpenedAt: "2025-01-01T00:00:00Z"},
			{ID: "c", Title: "C", LastOpenedAt: "2026-06-01T00:00:00Z"},
		},
	}
	s := NewShelfView(lib)
	if s.Selected().ID != "b" {
		t.Fatalf("last-read book should be first/selected, got %q", s.Selected().ID)
	}
}

func TestShelfMoveClamps(t *testing.T) {
	lib := &store.Library{Books: []store.BookEntry{{ID: "a"}, {ID: "b"}}}
	s := NewShelfView(lib)
	s.MoveUp() // already at top, stays
	if s.cursor != 0 {
		t.Fatalf("cursor should stay 0, got %d", s.cursor)
	}
	s.MoveDown()
	s.MoveDown() // clamp at last
	if s.cursor != 1 {
		t.Fatalf("cursor should clamp at 1, got %d", s.cursor)
	}
}

func TestShelfSelectedEmpty(t *testing.T) {
	s := NewShelfView(&store.Library{})
	if s.Selected() != nil {
		t.Fatal("empty shelf Selected() must be nil")
	}
	r := s.Render(40, 10)
	if len(r) == 0 {
		t.Fatal("empty shelf should still render a hint")
	}
}

// The shelf must read as developer-tool output (a git-log-style listing), never
// as a bookshelf: no "书架"/📚 chrome or plaintext reader UI.
func TestShelfDisguisedAsGitLog(t *testing.T) {
	lib := &store.Library{Books: []store.BookEntry{
		{ID: "a", Title: "三体", Author: "刘慈欣"},
		{ID: "b", Title: "活着", Author: "余华", Broken: true},
	}}
	joined := strings.Join(NewShelfView(lib).Render(80, 20), "\n")
	for _, leak := range []string{"书架", "📚", "选择", "阅读", "导入", "删除"} {
		if strings.Contains(joined, leak) {
			t.Fatalf("shelf leaked reader chrome %q:\n%s", leak, joined)
		}
	}
	if !strings.Contains(joined, "git log") {
		t.Fatalf("shelf should be framed as git log output:\n%s", joined)
	}
	if !strings.Contains(joined, "三体") || !strings.Contains(joined, "活着") {
		t.Fatalf("shelf should still list the books so the user can pick:\n%s", joined)
	}
}

func TestShelfEmptyDisguised(t *testing.T) {
	joined := strings.Join(NewShelfView(&store.Library{}).Render(80, 10), "\n")
	for _, leak := range []string{"书架", "📚", "导入"} {
		if strings.Contains(joined, leak) {
			t.Fatalf("empty shelf leaked reader chrome %q:\n%s", leak, joined)
		}
	}
}
