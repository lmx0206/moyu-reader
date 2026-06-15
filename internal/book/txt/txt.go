// Package txt parses plain-text books, auto-detecting UTF-8 / GB18030 and
// splitting chapters by line-anchored headings.
package txt

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"moyureader/internal/book"
	"moyureader/internal/render"
)

func init() { book.Register(".txt", Parse) }

var bom = []byte{0xEF, 0xBB, 0xBF}

// headingRE matches typical Chinese chapter headings at the start of a line.
var headingRE = regexp.MustCompile(`^第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章回卷节话篇集部]|^(序章|序言|楔子|引子|前言|后记|尾声|终章|番外|附录)`)

// headingMaxWidth guards against ordinary sentences that merely contain a
// chapter-like phrase: a real heading line is short.
const headingMaxWidth = 30

// decode turns raw bytes into a UTF-8 string, handling a UTF-8 BOM and
// falling back to GB18030 (a GBK superset) for non-UTF-8 input.
func decode(data []byte) (string, error) {
	if bytes.HasPrefix(data, bom) {
		return string(data[len(bom):]), nil
	}
	if utf8.Valid(data) {
		return string(data), nil
	}
	out, _, err := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func isHeading(line string) bool {
	return render.StringWidth(line) <= headingMaxWidth && headingRE.MatchString(line)
}

// Parse reads a .txt file into a Book.
func Parse(path string) (*book.Book, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text, err := decode(data)
	if err != nil {
		return nil, err
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	b := &book.Book{Title: title}

	cur := -1
	start := func(t string) {
		b.Chapters = append(b.Chapters, book.Chapter{Title: t})
		cur = len(b.Chapters) - 1
	}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, " \t　")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isHeading(trimmed) {
			start(trimmed)
			continue
		}
		if cur == -1 {
			start(title) // pre-heading body goes under a book-titled chapter
		}
		b.Chapters[cur].Paragraphs = append(b.Chapters[cur].Paragraphs, line)
	}
	if len(b.Chapters) == 0 {
		b.Chapters = []book.Chapter{{Title: title}}
	}
	return b, nil
}
