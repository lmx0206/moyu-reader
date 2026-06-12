package disguise

import (
	"strings"
	"testing"
)

func TestBossScreenContainsNoNovelText(t *testing.T) {
	th := ThemeByName("log")
	novel := "林尘睁开眼"
	lines := BossScreen(th, 0, 20)
	if len(lines) != 20 {
		t.Fatalf("want 20 lines got %d", len(lines))
	}
	for _, l := range lines {
		if strings.Contains(l, novel) {
			t.Fatalf("boss screen leaked novel text: %q", l)
		}
	}
}

func TestBossScreenScrollsByTick(t *testing.T) {
	th := ThemeByName("log")
	a := BossScreen(th, 0, 5)
	b := BossScreen(th, 1, 5)
	if a[0] == b[0] {
		t.Fatal("boss screen should scroll between ticks")
	}
}
