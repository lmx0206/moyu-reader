package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

const libraryFile = "library.json"

// Store reads and writes the library under a data directory.
type Store struct {
	dir string
}

// New returns a Store rooted at the given data directory.
func New(dir string) *Store { return &Store{dir: dir} }

// Dir returns the data directory path.
func (s *Store) Dir() string { return s.dir }

// BooksDir returns the directory where imported epub files are copied.
func (s *Store) BooksDir() string { return filepath.Join(s.dir, "books") }

func (s *Store) path() string { return filepath.Join(s.dir, libraryFile) }

// Load reads the library. A missing file yields a fresh empty library. A
// corrupt file is backed up to library.json.bak and replaced with an empty one.
func (s *Store) Load() (*Library, error) {
	data, err := os.ReadFile(s.path())
	if os.IsNotExist(err) {
		return NewLibrary(), nil
	}
	if err != nil {
		return nil, err
	}
	var lib Library
	if err := json.Unmarshal(data, &lib); err != nil {
		// back up the corrupt file, then recover
		_ = os.Rename(s.path(), s.path()+".bak")
		return NewLibrary(), nil
	}
	if lib.Global.Style == "" {
		lib.Global.Style = "log"
	}
	if lib.Global.Mode == "" {
		lib.Global.Mode = "shell"
	}
	return &lib, nil
}

// Save writes the library atomically (temp file + rename) so a crash mid-write
// never corrupts the existing file.
func (s *Store) Save(lib *Library) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lib, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, libraryFile+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path())
}

// newID returns a short random hex id for a book.
func newID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Import copies the epub at srcPath into <data>/books/<id>.epub and appends a
// new BookEntry to lib (caller is responsible for Save). Title/author come from
// the parsed book (passed in to keep store free of an epub dependency).
func (s *Store) Import(lib *Library, srcPath, title, author string) (*BookEntry, error) {
	if err := os.MkdirAll(s.BooksDir(), 0o755); err != nil {
		return nil, err
	}
	id := newID()
	rel := filepath.Join("books", id+".epub")
	dst := filepath.Join(s.dir, rel)
	if err := copyFile(srcPath, dst); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	entry := BookEntry{
		ID:           id,
		Title:        title,
		Author:       author,
		File:         filepath.ToSlash(rel),
		AddedAt:      now,
		LastOpenedAt: now,
		Prefs:        lib.Global,
	}
	lib.Books = append(lib.Books, entry)
	return lib.FindByID(id), nil
}

// copyFile copies src to dst, creating/truncating dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// UpdateProgress records reading position + prefs for a book, sets it as the
// last-read book, and stamps lastOpenedAt. No-op if the id is unknown.
func UpdateProgress(lib *Library, id string, p Progress, prefs Prefs) {
	e := lib.FindByID(id)
	if e == nil {
		return
	}
	e.Progress = p
	e.Prefs = prefs
	e.LastOpenedAt = time.Now().UTC().Format(time.RFC3339)
	lib.LastBookID = id
}
