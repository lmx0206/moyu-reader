package disguise

import "moyureader/internal/render"

// RenderInline turns each body line into a fake log/build/git line by prefixing
// it with theme decoration. The novel text becomes the "message" payload. This
// is reading mode B (highest stealth). width truncates the whole line; pass 0
// to leave untruncated (callers that own padding should pass 0).
func RenderInline(th Theme, body []string, width int) []string {
	out := make([]string, len(body))
	for i, line := range body {
		out[i] = fitLine(th.LinePrefix(i)+line, width)
	}
	return out
}

// prefixWidthCache memoizes PrefixWidth per theme name (themes are stateless
// singletons, so the widest prefix is constant for a given style).
var prefixWidthCache = map[string]int{}

// PrefixWidth returns the maximum display width of any line prefix th emits.
// Inline-mode callers wrap their body text to (terminalWidth - PrefixWidth) so
// that prefix + text fits the terminal and is never clipped at the right edge.
func PrefixWidth(th Theme) int {
	name := th.Name()
	if w, ok := prefixWidthCache[name]; ok {
		return w
	}
	max := 0
	for seed := 0; seed < 200; seed++ {
		if w := render.StringWidth(th.LinePrefix(seed)); w > max {
			max = w
		}
	}
	prefixWidthCache[name] = max
	return max
}
