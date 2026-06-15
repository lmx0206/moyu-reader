package disguise

// RenderShell renders reading mode A as a "log viewer": a theme top bar, a
// separator, the novel body (verbatim — the caller applies any indent/right
// gutter), a separator, and a theme bottom status bar. Decoration is 4 lines.
func RenderShell(th Theme, body []string, width int, status string) []string {
	sep := separatorLine(width)
	out := make([]string, 0, len(body)+4)
	out = append(out, th.Header(width, status))
	out = append(out, sep)
	out = append(out, body...)
	out = append(out, sep)
	out = append(out, th.Footer(width, status))
	return out
}
