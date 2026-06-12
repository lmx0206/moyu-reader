package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsEmptyLibrary(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	lib, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if lib == nil || len(lib.Books) != 0 || lib.Global.Style != "log" {
		t.Fatalf("expected empty default library, got %+v", lib)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	lib := NewLibrary()
	lib.Books = append(lib.Books, BookEntry{ID: "x", Title: "书"})
	lib.LastBookID = "x"
	if err := s.Save(lib); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.LastBookID != "x" || len(got.Books) != 1 || got.Books[0].Title != "书" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestLoadCorruptBacksUpAndRecovers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, libraryFile), []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(dir)
	lib, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Books) != 0 {
		t.Fatal("corrupt file should recover to empty library")
	}
	if _, err := os.Stat(filepath.Join(dir, libraryFile+".bak")); err != nil {
		t.Fatal("corrupt file should be backed up to .bak")
	}
}
