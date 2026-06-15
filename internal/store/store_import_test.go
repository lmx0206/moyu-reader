package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportPreservesExtension(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "novel.TXT")
	if err := os.WriteFile(src, []byte("正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := New(filepath.Join(dir, "data"))
	lib := NewLibrary()
	entry, err := st.Import(lib, src, "标题", "作者")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(entry.File, ".txt") {
		t.Fatalf("File should keep lowercased source extension, got %q", entry.File)
	}
	if _, err := os.Stat(filepath.Join(st.Dir(), filepath.FromSlash(entry.File))); err != nil {
		t.Fatalf("copied file missing: %v", err)
	}
}
