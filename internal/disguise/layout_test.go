package disguise

import (
	"strings"
	"testing"

	"moyureader/internal/render"
)

func TestPadBetween(t *testing.T) {
	got := padBetween("ab", "cd", 6) // 2 + gap(2) + 2
	if got != "ab  cd" {
		t.Fatalf("padBetween = %q want %q", got, "ab  cd")
	}
}

func TestPadBetweenCJKWidth(t *testing.T) {
	// "你好"=4 cells, "x"=1 cell, width 8 -> gap 3 spaces
	got := padBetween("你好", "x", 8)
	if render.StringWidth(got) != 8 {
		t.Fatalf("width = %d want 8 (%q)", render.StringWidth(got), got)
	}
}

func TestPadBetweenTooNarrowTruncates(t *testing.T) {
	got := padBetween("abcdef", "xyz", 4)
	if render.StringWidth(got) > 4 {
		t.Fatalf("should truncate to <=4, got width %d (%q)", render.StringWidth(got), got)
	}
}

// The footer puts the reading progress at the END of `left` (after a fake
// "INFO 142 passed ·" prefix). On a narrow terminal padBetween must keep that
// suffix (the only progress cue) plus the right marker, dropping the decorative
// prefix first — not the other way around.
func TestPadBetweenKeepsLeftSuffixWhenNarrow(t *testing.T) {
	got := padBetween("INFO 142 passed · 50%", "? help", 14)
	if render.StringWidth(got) > 14 {
		t.Fatalf("must not exceed width 14, got %d (%q)", render.StringWidth(got), got)
	}
	if !strings.Contains(got, "50%") {
		t.Fatalf("narrow padBetween dropped the progress suffix: %q", got)
	}
	if !strings.Contains(got, "? help") {
		t.Fatalf("must still keep the right marker: %q", got)
	}
}

func TestSeparatorLineWidth(t *testing.T) {
	got := separatorLine(10)
	if render.StringWidth(got) != 10 {
		t.Fatalf("separator width = %d want 10", render.StringWidth(got))
	}
}
