package disguise

import (
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

func TestSeparatorLineWidth(t *testing.T) {
	got := separatorLine(10)
	if render.StringWidth(got) != 10 {
		t.Fatalf("separator width = %d want 10", render.StringWidth(got))
	}
}
