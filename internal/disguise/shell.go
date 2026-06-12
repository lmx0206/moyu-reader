package disguise

// RenderShell wraps already-wrapped body lines with a theme header and footer.
// The body is returned verbatim (the UI layer applies width/padding/color); the
// status string is embedded into header/footer chrome. This is reading mode A.
func RenderShell(th Theme, body []string, width int, status string) []string {
	out := make([]string, 0, len(body)+2)
	out = append(out, th.Header(width, status))
	out = append(out, body...)
	out = append(out, th.Footer(width, status))
	return out
}
