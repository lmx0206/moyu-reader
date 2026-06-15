package main

import (
	"os"
	"path/filepath"
	"testing"

	"moyureader/internal/book"
)

// Importing package formats via main.go's blank import means .txt is registered.
func TestTxtOpensEndToEnd(t *testing.T) {
	p := filepath.Join(t.TempDir(), "novel.txt")
	if err := os.WriteFile(p, []byte("第一章\n正文一段\n第二章\n正文二段\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := book.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("want 2 chapters from .txt, got %d", len(b.Chapters))
	}
}
