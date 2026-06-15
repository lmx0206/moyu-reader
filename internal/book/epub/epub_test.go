package epub

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// writeSampleEPUB builds a minimal valid EPUB at path for testing.
func writeSampleEPUB(t *testing.T, path string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	add("mimetype", "application/epub+zip")
	add("META-INF/container.xml", `<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`)
	add("OEBPS/content.opf", `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:title>摸鱼测试书</dc:title><dc:creator>佚名</dc:creator></metadata>
<manifest>
<item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
<item id="c2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
</manifest>
<spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`)
	add("OEBPS/ch1.xhtml", `<html><body><h1>第一章</h1><p>开头第一段。</p></body></html>`)
	add("OEBPS/ch2.xhtml", `<html><body><h1>第二章</h1><p>第二章内容。</p></body></html>`)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseFullBook(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sample.epub")
	writeSampleEPUB(t, p)

	book, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if book.Title != "摸鱼测试书" || book.Author != "佚名" {
		t.Fatalf("bad meta: %+v", book)
	}
	if len(book.Chapters) != 2 {
		t.Fatalf("want 2 chapters, got %d", len(book.Chapters))
	}
	if book.Chapters[0].Paragraphs[0] != "第一章" {
		t.Fatalf("ch1 p0 = %q", book.Chapters[0].Paragraphs[0])
	}
	if book.Chapters[1].Paragraphs[1] != "第二章内容。" {
		t.Fatalf("ch2 p1 = %q", book.Chapters[1].Paragraphs[1])
	}
}

func TestParseMissingFile(t *testing.T) {
	if _, err := Parse(filepath.Join(t.TempDir(), "nope.epub")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
