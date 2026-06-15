// Package book is the format-agnostic entry point: it owns the parsed Book
// model and dispatches a file to a registered format backend by extension.
package book

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Book is a fully parsed book ready for rendering.
type Book struct {
	Title    string
	Author   string
	Chapters []Chapter
}

// Chapter is one section reduced to plain-text paragraphs.
type Chapter struct {
	Title      string
	Paragraphs []string
}

// Parser turns a file into a Book. Backends register one per extension.
type Parser func(path string) (*Book, error)

var (
	mu       sync.RWMutex
	registry = map[string]Parser{} // key: lowercase extension incl. dot, e.g. ".epub"
)

// Register records a parser for a file extension (e.g. ".txt"). Backends call
// this from init(). The last registration for an extension wins.
func Register(ext string, p Parser) {
	mu.Lock()
	registry[strings.ToLower(ext)] = p
	mu.Unlock()
}

// Open parses path using the parser registered for its extension.
func Open(path string) (*Book, error) {
	ext := strings.ToLower(filepath.Ext(path))
	mu.RLock()
	p, ok := registry[ext]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unsupported format %q (supported: %s)", ext, supported())
	}
	return p(path)
}

// supported returns a sorted, space-joined list of registered extensions.
func supported() string {
	mu.RLock()
	defer mu.RUnlock()
	exts := make([]string, 0, len(registry))
	for e := range registry {
		exts = append(exts, e)
	}
	sort.Strings(exts)
	return strings.Join(exts, " ")
}
