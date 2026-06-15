package stream

import (
	"bytes"
	"strings"
	"testing"

	"moyureader/internal/book"
	"moyureader/internal/store"
)

func sampleBook() *book.Book {
	mk := func(title string, n int) book.Chapter {
		ps := []string{title}
		for i := 0; i < n; i++ {
			ps = append(ps, strings.Repeat("文", 20))
		}
		return book.Chapter{Title: title, Paragraphs: ps}
	}
	return &book.Book{Title: "样书", Chapters: []book.Chapter{mk("第一章", 30)}}
}

func TestStreamerNextAdvances(t *testing.T) {
	s := NewStreamer(sampleBook(), store.Progress{}, "log", 5)
	first := s.Next()
	if len(first) != 5 {
		t.Fatalf("chunk should be height(5) lines, got %d", len(first))
	}
	p1 := s.Progress()
	_ = s.Next()
	p2 := s.Progress()
	if p2.Line <= p1.Line && p2.Chapter == p1.Chapter {
		t.Fatalf("position should advance: %+v -> %+v", p1, p2)
	}
	// inline 行应带日志前缀
	if !strings.Contains(first[0], " - ") {
		t.Fatalf("inline stream line should have log prefix: %q", first[0])
	}
}

func TestStreamRunEnterAndQuit(t *testing.T) {
	s := NewStreamer(sampleBook(), store.Progress{}, "log", 3)
	in := strings.NewReader("\n\nq\n") // two enters then quit
	var out bytes.Buffer
	var savedCalled bool
	Run(s, in, &out, func(p store.Progress, style string) { savedCalled = true })
	if !savedCalled {
		t.Fatal("Run should call onExit to save progress")
	}
	if out.Len() == 0 {
		t.Fatal("Run should print chunks")
	}
}
