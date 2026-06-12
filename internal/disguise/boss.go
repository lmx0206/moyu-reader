package disguise

// BossScreen returns `height` fake-output lines for the given theme. `tick`
// advances the stream so repeated calls with increasing tick scroll the
// content, simulating a live, running process. No novel text is ever included.
func BossScreen(th Theme, tick, height int) []string {
	lines := make([]string, height)
	for i := 0; i < height; i++ {
		lines[i] = th.BossLine(tick + i)
	}
	return lines
}
