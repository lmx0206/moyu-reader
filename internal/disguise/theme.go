// Package disguise renders novel text so it resembles developer tool output.
package disguise

import "fmt"

// Theme produces fake "work output" decoration for a given style.
type Theme interface {
	// Name returns the style id ("log"/"build"/"git").
	Name() string
	// LinePrefix returns a deterministic decoration placed before a body line.
	// seed makes the decoration vary per line while staying reproducible.
	LinePrefix(seed int) string
	// Header / Footer return chrome lines for shell-disguise mode.
	Header(width int, status string) string
	Footer(width int, status string) string
	// BossLine returns a pure fake-output line (no novel content) for seed.
	BossLine(seed int) string
}

// styleOrder defines the Tab cycle order.
var styleOrder = []string{"log", "build", "git"}

var registry = map[string]Theme{
	"log":   logTheme{},
	"build": buildTheme{},
	"git":   gitTheme{},
}

// ThemeByName returns the theme for name, falling back to "log" if unknown.
func ThemeByName(name string) Theme {
	if th, ok := registry[name]; ok {
		return th
	}
	return registry["log"]
}

// NextStyle returns the next style id in the Tab cycle.
func NextStyle(name string) string {
	for i, s := range styleOrder {
		if s == name {
			return styleOrder[(i+1)%len(styleOrder)]
		}
	}
	return styleOrder[0]
}

// --- log theme ---

type logTheme struct{}

func (logTheme) Name() string { return "log" }

var logLevels = []string{"INFO", "DEBUG", "INFO", "WARN", "INFO"}
var logClasses = []string{"OrderSvc", "CacheManager", "HttpPool", "AuthFilter", "TaskRunner"}

func (logTheme) LinePrefix(seed int) string {
	ts := fmt.Sprintf("%02d:%02d:%02d", 8+(seed/3600)%10, (seed/60)%60, seed%60)
	lvl := logLevels[seed%len(logLevels)]
	cls := logClasses[(seed/7)%len(logClasses)]
	return fmt.Sprintf("[%s] %-5s %s - ", ts, lvl, cls)
}

func (logTheme) Header(width int, status string) string {
	return fitLine("app.log — tail -f   "+status, width)
}
func (logTheme) Footer(width int, status string) string {
	return fitLine("INFO  watching for changes   "+status, width)
}
func (logTheme) BossLine(seed int) string {
	return fitLine(logTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- build theme ---

type buildTheme struct{}

func (buildTheme) Name() string { return "build" }

var buildVerbs = []string{"Compiling", "Linking", "Bundling", "Transpiling", "Resolving"}
var buildMods = []string{"core", "ui", "data", "net", "auth", "render"}

func (buildTheme) LinePrefix(seed int) string {
	return fmt.Sprintf("> %s %s … ", buildVerbs[seed%len(buildVerbs)], buildMods[(seed/3)%len(buildMods)])
}
func (buildTheme) Header(width int, status string) string {
	return fitLine("> gradle build   "+status, width)
}
func (buildTheme) Footer(width int, status string) string {
	return fitLine("BUILD SUCCESSFUL in 12s   "+status, width)
}
func (buildTheme) BossLine(seed int) string {
	return fitLine(buildTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- git theme ---

type gitTheme struct{}

func (gitTheme) Name() string { return "git" }

func (gitTheme) LinePrefix(seed int) string {
	if seed%4 == 0 {
		return "+ // "
	}
	return "  // "
}
func (gitTheme) Header(width int, status string) string {
	return fitLine("git diff --stat   "+status, width)
}
func (gitTheme) Footer(width int, status string) string {
	return fitLine("3 files changed, 128 insertions(+)   "+status, width)
}
func (gitTheme) BossLine(seed int) string {
	return fitLine(fmt.Sprintf("commit %08x  %s", seed*2654435761, bossPayload[seed%len(bossPayload)]), 0)
}

// bossPayload are neutral fake messages used only in boss-screen lines.
var bossPayload = []string{
	"processing batch 0x1f", "cache hit ratio 0.92", "retry attempt 1/3",
	"flushing write buffer", "connection pool resized", "gc pause 4ms",
	"index rebuilt", "checkpoint committed", "heartbeat ok",
}
