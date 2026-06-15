// Package disguise renders novel text so it resembles developer tool output.
package disguise

import (
	"fmt"
	"time"
)

// clockBase anchors fake log timestamps to the program's real start time so
// they look like a process that is currently emitting logs.
var clockBase = time.Now()

// logClock formats base advanced by seed seconds as HH:MM:SS.
func logClock(base time.Time, seed int) string {
	return base.Add(time.Duration(seed) * time.Second).Format("15:04:05")
}

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
	ts := logClock(clockBase, seed)
	lvl := logLevels[seed%len(logLevels)]
	cls := logClasses[(seed/7)%len(logClasses)]
	return fmt.Sprintf("[%s] %-5s %s - ", ts, lvl, cls)
}

func (logTheme) Header(width int, status string) string {
	return padBetween("app.log · tail -f", "● running", width)
}
func (logTheme) Footer(width int, status string) string {
	return padBetween("INFO  142 passed · "+status, "? help", width)
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
	return padBetween("> gradle build", "building…", width)
}
func (buildTheme) Footer(width int, status string) string {
	return padBetween("BUILD SUCCESSFUL in 12s · "+status, "? help", width)
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
	return padBetween("git log -p", "● HEAD", width)
}
func (gitTheme) Footer(width int, status string) string {
	return padBetween("3 files changed · "+status, "? help", width)
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
