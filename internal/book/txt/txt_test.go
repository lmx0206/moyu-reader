package txt

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func write(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseUTF8Chapters(t *testing.T) {
	p := write(t, "a.txt", []byte("第一章\n正文一\n第二章\n正文二\n"))
	b, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("want 2 chapters, got %d: %+v", len(b.Chapters), b.Chapters)
	}
	if b.Chapters[0].Title != "第一章" || b.Chapters[0].Paragraphs[0] != "正文一" {
		t.Fatalf("ch0 wrong: %+v", b.Chapters[0])
	}
}

func TestParseUTF8BOM(t *testing.T) {
	p := write(t, "b.txt", append([]byte{0xEF, 0xBB, 0xBF}, []byte("第一章\n正文\n")...))
	b, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if b.Chapters[0].Title != "第一章" {
		t.Fatalf("BOM not stripped / title wrong: %q", b.Chapters[0].Title)
	}
}

func TestParseGB18030(t *testing.T) {
	gbk, _, err := transform.Bytes(simplifiedchinese.GB18030.NewEncoder(), []byte("第一章\n甲乙丙丁\n"))
	if err != nil {
		t.Fatal(err)
	}
	p := write(t, "c.txt", gbk)
	b, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if b.Chapters[0].Title != "第一章" || b.Chapters[0].Paragraphs[0] != "甲乙丙丁" {
		t.Fatalf("GB18030 decode wrong: %+v", b.Chapters[0])
	}
}

func TestParseShortLineGuard(t *testing.T) {
	long := "他翻到第三章的时候发现这是一个非常非常长的句子超过了三十个显示宽度的护栏阈值继续"
	p := write(t, "d.txt", []byte("第一章\n"+long+"\n"))
	b, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 1 {
		t.Fatalf("long line containing 第三章 must NOT start a chapter: %d chapters", len(b.Chapters))
	}
	if b.Chapters[0].Paragraphs[0] != long {
		t.Fatalf("long line should be a paragraph: %+v", b.Chapters[0])
	}
}

func TestParseFallbackSingleChapter(t *testing.T) {
	p := write(t, "mybook.txt", []byte("没有任何标题的一段\n第二段文字\n"))
	b, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 1 {
		t.Fatalf("no headings -> single chapter, got %d", len(b.Chapters))
	}
	if b.Chapters[0].Title != "mybook" {
		t.Fatalf("fallback title should be filename base, got %q", b.Chapters[0].Title)
	}
	if b.Title != "mybook" {
		t.Fatalf("book title should be filename base, got %q", b.Title)
	}
}

func TestParseCRLF(t *testing.T) {
	p := write(t, "e.txt", []byte("第一章\r\n正文\r\n"))
	b, err := Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if b.Chapters[0].Title != "第一章" || b.Chapters[0].Paragraphs[0] != "正文" {
		t.Fatalf("CRLF not normalized: %+v", b.Chapters[0])
	}
}
