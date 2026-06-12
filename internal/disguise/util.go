package disguise

import (
	"strings"

	"moyureader/internal/render"
)

// fitLine truncates or returns s. When width<=0 it returns s unchanged; when
// width>0 it truncates to width display cells (never pads, to keep golden tests
// stable and let the UI layer own padding/coloring).
func fitLine(s string, width int) string {
	if width <= 0 || render.StringWidth(s) <= width {
		return s
	}
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		rw := render.RuneWidth(r)
		if w+rw > width {
			return string(runes[:i])
		}
		w += rw
	}
	return s
}

// padBetween left-aligns left, right-aligns right, and fills the middle with
// spaces so the whole line is exactly width display cells. If there is not
// enough room, it keeps left and truncates to width.
func padBetween(left, right string, width int) string {
	gap := width - render.StringWidth(left) - render.StringWidth(right)
	if gap < 1 {
		return fitLine(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

// separatorLine returns a horizontal rule of width box-drawing dashes.
func separatorLine(width int) string {
	if width < 1 {
		width = 1
	}
	return strings.Repeat("─", width)
}
