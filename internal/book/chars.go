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
