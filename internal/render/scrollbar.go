package render

import "strings"

// PadRight pads s with spaces to exactly width display cells, or truncates it
// to width cells if it is longer. CJK width-aware.
func PadRight(s string, width int) string {
	w := StringWidth(s)
	if w == width {
		return s
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	runes := []rune(s)
	acc := 0
	for i, r := range runes {
		rw := RuneWidth(r)
		if acc+rw > width {
			return string(runes[:i])
		}
		acc += rw
	}
	return s
}

// Scrollbar returns viewport glyphs representing a vertical scrollbar for a list
// of total items with the window starting at top. "█" marks the thumb, "░" the
// track. When everything fits (total<=viewport) all glyphs are track.
func Scrollbar(total, top, viewport int) []string {
	bars := make([]string, viewport)
	if viewport <= 0 {
		return bars
	}
	if total <= viewport {
		for i := range bars {
			bars[i] = "░"
		}
		return bars
	}
	thumb := viewport * viewport / total
	if thumb < 1 {
		thumb = 1
	}
	maxTop := total - viewport
	pos := 0
	if maxTop > 0 {
		pos = top * (viewport - thumb) / maxTop
	}
	if pos < 0 {
		pos = 0
	}
	if pos > viewport-thumb {
		pos = viewport - thumb
	}
	for i := range bars {
		if i >= pos && i < pos+thumb {
			bars[i] = "█"
		} else {
			bars[i] = "░"
		}
	}
	return bars
}
