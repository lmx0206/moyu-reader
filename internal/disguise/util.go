package disguise

import "moyureader/internal/render"

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
