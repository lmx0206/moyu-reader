package render

// ParagraphStartLines returns, for each paragraph, the display-line index where
// its content begins when laid out at width. It matches LayoutChapter, which
// inserts one blank line between paragraphs.
func ParagraphStartLines(paragraphs []string, width int) []int {
	starts := make([]int, len(paragraphs))
	line := 0
	for i, p := range paragraphs {
		if i > 0 {
			line++ // blank separator line before this paragraph
		}
		starts[i] = line
		line += len(WrapParagraph(p, width))
	}
	return starts
}

// ParaStartLine returns the start display-line of paragraph para (clamped to a
// valid index). It returns 0 when there are no paragraphs.
func ParaStartLine(paragraphs []string, width, para int) int {
	if len(paragraphs) == 0 {
		return 0
	}
	if para < 0 {
		para = 0
	}
	if para >= len(paragraphs) {
		para = len(paragraphs) - 1
	}
	return ParagraphStartLines(paragraphs, width)[para]
}

// LineToPara returns the index of the paragraph that owns display-line line (the
// last paragraph whose start <= line), clamped to [0, len-1]. It returns 0 when
// there are no paragraphs.
func LineToPara(paragraphs []string, width, line int) int {
	if len(paragraphs) == 0 {
		return 0
	}
	para := 0
	for i, s := range ParagraphStartLines(paragraphs, width) {
		if s <= line {
			para = i
		} else {
			break
		}
	}
	return para
}
