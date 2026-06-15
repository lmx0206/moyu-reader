package disguise

import (
	"strings"
	"testing"
)

func TestStylesRegistered(t *testing.T) {
	for _, name := range []string{"log", "build", "git"} {
		if ThemeByName(name) == nil {
			t.Fatalf("theme %q not registered", name)
		}
	}
}

func TestThemeByNameFallback(t *testing.T) {
	if ThemeByName("nonsense").Name() != "log" {
		t.Fatal("unknown theme should fall back to log")
	}
}

func TestNextStyleCycles(t *testing.T) {
	if NextStyle("log") != "build" || NextStyle("build") != "git" || NextStyle("git") != "docker" {
		t.Fatal("style cycle wrong")
	}
}

// LogLine for a deterministic seed must be stable and contain the payload.
func TestLogThemePrefixDeterministic(t *testing.T) {
	th := ThemeByName("log")
	a := th.LinePrefix(42)
	b := th.LinePrefix(42)
	if a != b {
		t.Fatalf("prefix not deterministic: %q vs %q", a, b)
	}
	if !strings.Contains(a, "INFO") && !strings.Contains(a, "DEBUG") && !strings.Contains(a, "WARN") {
		t.Fatalf("log prefix missing level: %q", a)
	}
}
