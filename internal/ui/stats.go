package ui

import (
	"fmt"
	"strconv"
	"strings"

	"moyureader/internal/render"
	"moyureader/internal/store"
)

// StatsView renders reading statistics disguised as a code-coverage report.
type StatsView struct{}

const statsNameWidth = 18

// Render returns exactly height lines of a coverage-styled stats report.
func (StatsView) Render(lib *store.Library, width, height int) []string {
	out := []string{
		render.PadRight("Name", statsNameWidth) + fmt.Sprintf("  %6s %6s  %5s", "Stmts", "Miss", "Cover"),
		strings.Repeat("-", statsNameWidth+22),
	}

	totalStmts, totalRead := 0, 0
	for _, b := range lib.Books {
		name := b.Title
		if name == "" {
			name = b.ID
		}
		stmts := b.TotalChars
		read := b.CharsRead
		if read > stmts {
			read = stmts
		}
		miss := stmts - read
		totalStmts += stmts
		totalRead += read
		out = append(out, statsRow(name, stmts, miss, coverPct(read, stmts)))
	}

	out = append(out, strings.Repeat("-", statsNameWidth+22))
	out = append(out, statsRow("TOTAL", totalStmts, totalStmts-totalRead, coverPct(totalRead, totalStmts)))
	out = append(out, "")
	out = append(out, fmt.Sprintf("elapsed %s · today %s · %s chars · streak %dd",
		formatDur(lib.Stats.TotalSeconds), formatDur(lib.Stats.TodaySeconds),
		humanChars(totalRead), lib.Stats.StreakDays))

	// pad/truncate to exactly height lines
	for len(out) < height {
		out = append(out, "")
	}
	if len(out) > height {
		out = out[:height]
	}
	return out
}

func statsRow(name string, stmts, miss, cover int) string {
	return render.PadRight(truncCells(name, statsNameWidth), statsNameWidth) +
		fmt.Sprintf("  %6d %6d  %4d%%", stmts, miss, cover)
}

func coverPct(read, stmts int) int {
	if stmts <= 0 {
		return 0
	}
	return read * 100 / stmts
}

func formatDur(sec int) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func humanChars(n int) string {
	if n >= 1000 {
		return strconv.Itoa(n/1000) + "k"
	}
	return strconv.Itoa(n)
}

// truncCells truncates s to at most w display cells (CJK-aware).
func truncCells(s string, w int) string {
	if render.StringWidth(s) <= w {
		return s
	}
	out := make([]rune, 0, len(s))
	used := 0
	for _, r := range s {
		rw := render.RuneWidth(r)
		if used+rw > w {
			break
		}
		out = append(out, r)
		used += rw
	}
	return string(out)
}
