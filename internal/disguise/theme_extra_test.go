package disguise

import (
	"strings"
	"testing"
)

func TestNewThemesRegistered(t *testing.T) {
	for _, name := range []string{"docker", "npm", "pytest"} {
		if got := ThemeByName(name).Name(); got != name {
			t.Fatalf("ThemeByName(%q).Name() = %q", name, got)
		}
	}
}

func TestStyleCycleIncludesNewThemes(t *testing.T) {
	seen := map[string]bool{}
	s := "log"
	for i := 0; i < 6; i++ {
		seen[s] = true
		s = NextStyle(s)
	}
	for _, n := range []string{"log", "build", "git", "docker", "npm", "pytest"} {
		if !seen[n] {
			t.Fatalf("cycle should include %q; seen=%v", n, seen)
		}
	}
	if NextStyle("pytest") != "log" {
		t.Fatalf("cycle should wrap pytest->log, got %q", NextStyle("pytest"))
	}
}

func TestNewThemesHeaderFooter(t *testing.T) {
	cases := map[string]string{
		"docker": "docker compose up",
		"npm":    "npm install",
		"pytest": "pytest -v",
	}
	for name, headerMark := range cases {
		th := ThemeByName(name)
		if !strings.Contains(th.Header(60, "st"), headerMark) {
			t.Fatalf("%s header missing %q: %q", name, headerMark, th.Header(60, "st"))
		}
		f := th.Footer(60, "STATUSMARK")
		if !strings.Contains(f, "? help") || !strings.Contains(f, "STATUSMARK") {
			t.Fatalf("%s footer should embed status + help: %q", name, f)
		}
		if a, b := th.LinePrefix(3), th.LinePrefix(3); a != b {
			t.Fatalf("%s LinePrefix not deterministic: %q vs %q", name, a, b)
		}
		if strings.Contains(th.BossLine(2), "STATUSMARK") {
			t.Fatalf("%s BossLine must not contain status/novel", name)
		}
	}
}

// The pytest theme's test numbers must not increment 0,1,2,… with the line
// index (a clean ramp reads as fake). The number is the trailing token after
// the last "test_"; consecutive seeds must not produce consecutive numbers.
func TestPytestTestNumbersNotSequential(t *testing.T) {
	th := pytestTheme{}
	num := func(seed int) string {
		p := th.LinePrefix(seed)
		i := strings.LastIndex(p, "test_")
		return strings.TrimSpace(p[i+len("test_"):])
	}
	if num(0) == "0" && num(1) == "1" && num(2) == "2" {
		t.Fatalf("pytest numbers ramp sequentially: %s,%s,%s", num(0), num(1), num(2))
	}
}
