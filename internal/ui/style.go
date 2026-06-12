package ui

import "github.com/charmbracelet/lipgloss"

// dimStyle colors chrome/prefix lines so the disguise looks like a real tool.
var (
	chromeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	bodyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// paintShell colors header (line0) and footer (last) dimmer than the body.
func paintShell(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if i == 0 || i == len(lines)-1 {
			out[i] = chromeStyle.Render(l)
		} else {
			out[i] = bodyStyle.Render(l)
		}
	}
	return out
}

// paintDim colors every line as chrome (used for inline + boss screens).
func paintDim(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = chromeStyle.Render(l)
	}
	return out
}
