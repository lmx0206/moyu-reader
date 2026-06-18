package book

import "testing"

func charBook() *Book {
	return &Book{
		Title: "T",
		Chapters: []Chapter{
			{Title: "c1", Paragraphs: []string{"ab", "cde"}},   // 2 + 3 = 5
			{Title: "c2", Paragraphs: []string{"文文文", "x"}}, // 3 + 1 = 4
		},
	}
}

func TestTotalChars(t *testing.T) {
	if got := TotalChars(charBook()); got != 9 {
		t.Fatalf("TotalChars = %d, want 9", got)
	}
}

func TestCharsUpTo(t *testing.T) {
	b := charBook()
	// before (0,0): nothing read
	if got := CharsUpTo(b, 0, 0); got != 0 {
		t.Fatalf("CharsUpTo(0,0) = %d, want 0", got)
	}
	// before (0,1): paragraph 0 of chapter 0 -> "ab" = 2
	if got := CharsUpTo(b, 0, 1); got != 2 {
		t.Fatalf("CharsUpTo(0,1) = %d, want 2", got)
	}
	// before (1,1): all of chapter 0 (5) + paragraph 0 of chapter 1 ("文文文"=3) = 8
	if got := CharsUpTo(b, 1, 1); got != 8 {
		t.Fatalf("CharsUpTo(1,1) = %d, want 8", got)
	}
	// out-of-range clamps to the end (everything) = 9
	if got := CharsUpTo(b, 9, 9); got != 9 {
		t.Fatalf("CharsUpTo(9,9) = %d, want 9", got)
	}
}
