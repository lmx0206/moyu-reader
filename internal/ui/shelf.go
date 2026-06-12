package ui

import (
	"fmt"
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

// Render returns up to height lines listing the books.
func (s *ShelfView) Render(width, height int) []string {
	if len(s.items) == 0 {
		return []string{
			"书架空空如也。",
			"按 i 导入一本 .epub，或用命令行: reader <某本书.epub>",
		}
	}
	var out []string
	out = append(out, "📚 书架   ↑↓ 选择 · Enter 阅读 · i 导入 · d 删除 · q 退出")
	out = append(out, "")
	for i, e := range s.items {
		cursor := "  "
		if i == s.cursor {
			cursor = "> "
		}
		broken := ""
		if e.Broken {
			broken = " [损坏]"
		}
		line := fmt.Sprintf("%s%s — %s%s", cursor, e.Title, e.Author, broken)
		out = append(out, line)
		if len(out) >= height {
			break
		}
	}
	return out
}
