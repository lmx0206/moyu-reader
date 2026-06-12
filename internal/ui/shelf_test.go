package ui

import (
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
