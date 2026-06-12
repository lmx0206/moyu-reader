package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportCopiesFileAndAddsEntry(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(t.TempDir(), "novel.epub")
	if err := os.WriteFile(src, []byte("FAKEEPUBDATA"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(dir)
	lib := NewLibrary()

	entry, err := s.Import(lib, src, "我的小说", "作者甲")
	if err != nil {
		t.Fatal(err)
	}
	if entry.ID == "" || entry.Title != "我的小说" || entry.Author != "作者甲" {
		t.Fatalf("bad entry: %+v", entry)
	}
	copied := filepath.Join(dir, entry.File)
	if data, err := os.ReadFile(copied); err != nil || string(data) != "FAKEEPUBDATA" {
		t.Fatalf("epub not copied into data dir: err=%v", err)
	}
	if lib.FindByID(entry.ID) == nil {
		t.Fatal("entry not added to library")
	}
}

func TestUpdateProgress(t *testing.T) {
	lib := NewLibrary()
	lib.Books = append(lib.Books, BookEntry{ID: "a"})
	UpdateProgress(lib, "a", Progress{Chapter: 2, Line: 40}, Prefs{Style: "git", Mode: "inline"})
	e := lib.FindByID("a")
	if e.Progress.Chapter != 2 || e.Progress.Line != 40 {
		t.Fatalf("progress not updated: %+v", e.Progress)
	}
	if e.Prefs.Style != "git" || lib.LastBookID != "a" {
		t.Fatalf("prefs/lastBook not updated: %+v", e)
	}
	if e.LastOpenedAt == "" {
		t.Fatal("lastOpenedAt should be set")
	}
}
