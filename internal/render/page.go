package render

// LayoutChapter wraps every paragraph to width and joins them into a single
// flat slice of display lines, inserting one blank line between paragraphs.
func LayoutChapter(paragraphs []string, width int) []string {
	var lines []string
	for idx, p := range paragraphs {
		if idx > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, WrapParagraph(p, width)...)
	}
	if lines == nil {
		return []string{}
	}
	return lines
}

// Paginate splits lines into pages of at most height lines each.
func Paginate(lines []string, height int) [][]string {
	if height <= 0 {
		return [][]string{lines}
	}
	var pages [][]string
	for i := 0; i < len(lines); i += height {
		end := i + height
		if end > len(lines) {
			end = len(lines)
		}
		pages = append(pages, lines[i:end])
	}
	return pages
}

// LineToPage returns the zero-based page index containing the given line index.
func LineToPage(line, height int) int {
	if height <= 0 {
		return 0
	}
	return line / height
}
