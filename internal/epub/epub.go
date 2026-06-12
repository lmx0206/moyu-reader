// Package epub parses EPUB files into plain-text chapters.
package epub

// Book is a fully parsed EPUB ready for rendering.
type Book struct {
	Title    string
	Author   string
	Chapters []Chapter
}

// Chapter is one spine item reduced to plain-text paragraphs.
type Chapter struct {
	Title      string
	Paragraphs []string
}
