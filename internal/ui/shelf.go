package ui

import (
	"fmt"
	"hash/fnv"
	"sort"

	"moyureader/internal/store"
)

// ShelfView is the bookshelf list with a cursor.
type ShelfView struct {
	items  []store.BookEntry
	cursor int
}

// NewShelfView builds a shelf ordered so the last-read book is first, then by
// most-recently-opened.
func NewShelfView(lib *store.Library) *ShelfView {
	items := make([]store.BookEntry, len(lib.Books))
	copy(items, lib.Books)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ID == lib.LastBookID {
			return true
		}
		if items[j].ID == lib.LastBookID {
			return false
		}
		return items[i].LastOpenedAt > items[j].LastOpenedAt
	})
	return &ShelfView{items: items}
}

// MoveUp / MoveDown move the cursor with clamping.
func (s *ShelfView) MoveUp() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s *ShelfView) MoveDown() {
	if s.cursor < len(s.items)-1 {
		s.cursor++
	}
}

// Selected returns the highlighted entry, or nil if the shelf is empty.
func (s *ShelfView) Selected() *store.BookEntry {
	if len(s.items) == 0 {
		return nil
	}
	return &s.items[s.cursor]
}

// shortHash returns a deterministic 7-hex-digit id for a book, so the shelf can
// present each book as a fake git commit.
func shortHash(id string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return fmt.Sprintf("%07x", h.Sum32()&0xfffffff)
}

// Render lists the books disguised as `git log --oneline` output: each book is a
// commit (hash + subject), the selected one carries the (HEAD) marker, and a
// broken book is annotated (stale). It never reads as a bookshelf.
func (s *ShelfView) Render(width, height int) []string {
	out := []string{"$ git log --oneline"}
	if len(s.items) == 0 {
		return append(out, "fatal: your current branch 'main' does not have any commits yet")
	}
	for i, e := range s.items {
		subject := e.Title
		if subject == "" {
			subject = e.ID
		}
		marker := ""
		if i == s.cursor {
			marker = " (HEAD)"
		}
		stale := ""
		if e.Broken {
			stale = " (stale)"
		}
		out = append(out, fmt.Sprintf("%s%s %s%s", shortHash(e.ID), marker, subject, stale))
		if len(out) >= height {
			break
		}
	}
	return out
}
