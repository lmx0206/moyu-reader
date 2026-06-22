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
// spaces so the whole line is exactly width display cells. When there is not
// enough room, it prefers keeping the right marker visible (e.g. the "? help"
// hint), truncating the left to make space; if even the right marker plus a
// separator does not fit, it truncates the whole line to width.
func padBetween(left, right string, width int) string {
	lw := render.StringWidth(left)
	rw := render.StringWidth(right)
	if gap := width - lw - rw; gap >= 1 {
		return left + strings.Repeat(" ", gap) + right
	}
	if rw+1 < width {
		// Keep the SUFFIX of left (the footer puts the reading progress there)
		// rather than its decorative prefix, while still showing the right marker.
		return fitLineRight(left, width-rw-1) + " " + right
	}
	return fitLine(left, width)
}

// fitLineRight returns the rightmost `width` display cells of s (CJK-aware),
// keeping the suffix when s is too wide. width<=0 returns "".
func fitLineRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if render.StringWidth(s) <= width {
		return s
	}
	runes := []rune(s)
	w := 0
	// walk from the end, accumulating until we'd exceed width
	for i := len(runes) - 1; i >= 0; i-- {
		rw := render.RuneWidth(runes[i])
		if w+rw > width {
			return string(runes[i+1:])
		}
		w += rw
	}
	return s
}

// separatorLine returns a horizontal rule of width box-drawing dashes.
func separatorLine(width int) string {
	if width < 1 {
		width = 1
	}
	return strings.Repeat("─", width)
}
