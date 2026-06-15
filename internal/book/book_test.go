package book

import (
	"strings"
	"testing"
)

func TestOpenRoutesByExtensionCaseInsensitive(t *testing.T) {
	Register(".zz1", func(path string) (*Book, error) {
		return &Book{Title: "routed:" + path}, nil
	})
	t.Cleanup(func() {
		mu.Lock()
		delete(registry, ".zz1")
		mu.Unlock()
	})
	b, err := Open("dir/sample.ZZ1") // 大写扩展名也应命中
	if err != nil {
		t.Fatal(err)
	}
	if b.Title != "routed:dir/sample.ZZ1" {
		t.Fatalf("Open routed to wrong parser: %q", b.Title)
	}
}

func TestOpenUnknownFormat(t *testing.T) {
	_, err := Open("a.nosuchext")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "supported") {
		t.Fatalf("error should list supported formats: %v", err)
	}
}
