package render

import (
	"strings"
	"testing"
)

func TestPadRightASCII(t *testing.T) {
	if got := PadRight("ab", 5); got != "ab   " {
		t.Fatalf("PadRight = %q want %q", got, "ab   ")
	}
}

func TestPadRightCJKExactWidth(t *testing.T) {
	// "你好"=4 cells, pad to 6 -> 2 trailing spaces
	got := PadRight("你好", 6)
	if StringWidth(got) != 6 {
		t.Fatalf("width = %d want 6 (%q)", StringWidth(got), got)
	}
}

func TestPadRightTruncates(t *testing.T) {
	got := PadRight("abcdef", 3)
	if got != "abc" {
		t.Fatalf("PadRight truncate = %q want abc", got)
	}
}

func TestScrollbarLengthAndGlyphs(t *testing.T) {
	bars := Scrollbar(100, 0, 10)
	if len(bars) != 10 {
		t.Fatalf("len = %d want 10", len(bars))
	}
	joined := strings.Join(bars, "")
	if !strings.Contains(joined, "█") {
		t.Fatalf("should contain a thumb glyph: %q", joined)
	}
	if !strings.Contains(joined, "░") {
		t.Fatalf("should contain track glyph: %q", joined)
	}
}

func TestScrollbarThumbMovesDown(t *testing.T) {
	top := Scrollbar(100, 0, 10)
	bottom := Scrollbar(100, 90, 10)
	// thumb at top: first glyph is thumb; at bottom: last glyph is thumb
	if top[0] != "█" {
		t.Fatalf("thumb should be at top: %v", top)
	}
	if bottom[len(bottom)-1] != "█" {
		t.Fatalf("thumb should be at bottom: %v", bottom)
	}
}

func TestScrollbarNoScrollAllTrack(t *testing.T) {
	bars := Scrollbar(5, 0, 10) // content fits viewport
	for _, b := range bars {
		if b != "░" {
			t.Fatalf("no-scroll should be all track, got %v", bars)
		}
	}
}
