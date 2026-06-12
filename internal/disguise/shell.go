package disguise

import "strings"

// BodyIndent is the left margin (in spaces) applied to novel body lines in
// shell-disguise mode. The UI's contentWidth must reserve this space.
const BodyIndent = 3

// RenderShell renders reading mode A as a "log viewer": a theme top bar, a
// separator, the indented novel body, a separator, and a theme bottom status
// bar. Total decoration is 4 lines. status is embedded in the bottom bar.
func RenderShell(th Theme, body []string, width int, status string) []string {
	sep := separatorLine(width)
	indent := strings.Repeat(" ", BodyIndent)
	out := make([]string, 0, len(body)+4)
	out = append(out, th.Header(width, status))
	out = append(out, sep)
	for _, l := range body {
		if l == "" {
			out = append(out, "")
		} else {
			out = append(out, indent+l)
		}
	}
	out = append(out, sep)
	out = append(out, th.Footer(width, status))
	return out
}
