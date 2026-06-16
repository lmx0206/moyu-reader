package render

import "testing"

func TestParagraphStartLines(t *testing.T) {
	// 两个单行段落：段间一个空行 -> 起始行 0 和 2
	got := ParagraphStartLines([]string{"abc", "def"}, 80)
	if len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("starts = %v want [0 2]", got)
	}
}

func TestParaStartLineClamp(t *testing.T) {
	ps := []string{"a", "b"}
	if ParaStartLine(ps, 80, 99) != ParagraphStartLines(ps, 80)[1] {
		t.Fatalf("para should clamp to last")
	}
	if ParaStartLine(nil, 80, 0) != 0 {
		t.Fatalf("empty -> 0")
	}
}

func TestLineToParaRoundTrip(t *testing.T) {
	ps := []string{"一二三四五六七八九十", "甲乙丙丁", "x"}
	for p := range ps {
		start := ParaStartLine(ps, 6, p)
		if got := LineToPara(ps, 6, start); got != p {
			t.Fatalf("round trip para %d -> line %d -> para %d", p, start, got)
		}
	}
}

func TestLineToParaBlankAndOOB(t *testing.T) {
	ps := []string{"abc", "def"}
	if LineToPara(ps, 80, 1) != 0 { // 段间空行归前一段
		t.Fatalf("blank separator should map to preceding para")
	}
	if LineToPara(ps, 80, 999) != 1 { // 越界行归最后一段
		t.Fatalf("oob line should map to last para")
	}
	if LineToPara(nil, 80, 0) != 0 {
		t.Fatalf("empty -> 0")
	}
}
