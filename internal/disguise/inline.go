package disguise

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
