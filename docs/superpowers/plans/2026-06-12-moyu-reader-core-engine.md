# 摸鱼阅读器 · 计划一：核心引擎 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建摸鱼 EPUB 阅读器的无界面核心：EPUB 解析、CJK 感知排版分页、伪装渲染、书架与进度持久化，全部单元测试覆盖。

**Architecture:** 纯 Go 库代码，按职责分包：`internal/epub`（解析）、`internal/render`（排版分页）、`internal/disguise`（伪装文本生成）、`internal/store`（JSON 持久化）。无终端/UI 依赖，便于 golden 测试。计划二的 TUI 与 CLI 前端将消费这些包。

**Tech Stack:** Go 1.22+、`github.com/mattn/go-runewidth`（CJK 宽度）、`golang.org/x/net/html`（XHTML 解析）、标准库 `archive/zip` / `encoding/xml` / `encoding/json`。本计划不引入 bubbletea/lipgloss（留给计划二）。

---

## 模块文件结构

```
go.mod                         模块定义 (module moyureader)
internal/epub/epub.go          Book/Chapter 类型 + Parse 入口
internal/epub/container.go     container.xml → OPF 路径
internal/epub/opf.go           OPF manifest+spine → 有序章节文件列表
internal/epub/html.go          XHTML → 段落纯文本
internal/epub/epub_test.go     解析测试
internal/epub/testdata/        fixture epub
internal/render/width.go       CJK 宽度条件 + 宽度函数
internal/render/wrap.go        段落按宽度换行
internal/render/page.go        行序列分页 + 位置坐标
internal/render/*_test.go      排版分页测试
internal/disguise/theme.go     Theme 接口 + 三种风格的元数据生成
internal/disguise/shell.go     模式A：外壳伪装渲染
internal/disguise/inline.go    模式B：正文藏日志行渲染
internal/disguise/boss.go      老板键假屏行生成
internal/disguise/*_test.go    伪装 golden 测试
internal/store/model.go        Library/BookEntry/Progress 类型
internal/store/store.go        加载/原子保存/导入/进度更新/损坏恢复
internal/store/store_test.go   持久化测试
```

---

## Task 0: 项目脚手架

**Files:**
- Create: `go.mod`
- Create: `internal/version/version.go`
- Test: `internal/version/version_test.go`

- [ ] **Step 1: 初始化 go module**

Run:
```bash
cd /d/StudioProjects/reader
go mod init moyureader
go get github.com/mattn/go-runewidth@latest
go get golang.org/x/net/html@latest
```
Expected: `go.mod` 创建，含两个 require。

- [ ] **Step 2: 写一个最小包 + 失败测试**

Create `internal/version/version.go`:
```go
// Package version exposes the application version string.
package version

// Version is the current build version.
const Version = "0.1.0"
```

Create `internal/version/version_test.go`:
```go
package version

import "testing"

func TestVersionNotEmpty(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
}
```

- [ ] **Step 3: 运行测试确认通过（验证工具链可用）**

Run: `go test ./internal/version/...`
Expected: `ok  moyureader/internal/version`

- [ ] **Step 4: 提交**

```bash
git add go.mod go.sum internal/version
git commit -m "chore: scaffold go module and toolchain smoke test"
```

---

## Task 1: EPUB 类型与 HTML 提取

**Files:**
- Create: `internal/epub/epub.go`
- Create: `internal/epub/html.go`
- Test: `internal/epub/html_test.go`

- [ ] **Step 1: 写失败测试（HTML → 段落）**

Create `internal/epub/html_test.go`:
```go
package epub

import (
	"reflect"
	"testing"
)

func TestHTMLToParagraphs(t *testing.T) {
	html := `<html><body>
		<h1>第一章</h1>
		<p>林尘睁开了眼。</p>
		<p>他看见<em>窗外</em>的月光。</p>
		<div>另一段。<br/>换行后。</div>
		<script>ignore();</script>
	</body></html>`
	got := htmlToParagraphs(html)
	want := []string{
		"第一章",
		"林尘睁开了眼。",
		"他看见窗外的月光。",
		"另一段。",
		"换行后。",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestHTMLToParagraphsCollapsesWhitespace(t *testing.T) {
	got := htmlToParagraphs("<p>  hello   world  </p>")
	want := []string{"hello world"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/epub/ -run TestHTMLToParagraphs`
Expected: FAIL（`htmlToParagraphs` undefined）

- [ ] **Step 3: 实现 HTML 提取**

Create `internal/epub/html.go`:
```go
package epub

import (
	"strings"

	"golang.org/x/net/html"
)

// blockTags trigger a paragraph break before and after their content.
var blockTags = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "li": true, "blockquote": true,
	"section": true, "article": true, "tr": true,
}

// skipTags and their content are dropped entirely.
var skipTags = map[string]bool{
	"script": true, "style": true, "head": true,
}

// htmlToParagraphs parses an XHTML document body into trimmed text paragraphs.
// Block-level elements and <br> introduce paragraph boundaries; inline text is
// concatenated; runs of whitespace collapse to a single space.
func htmlToParagraphs(doc string) []string {
	node, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return nil
	}
	var paras []string
	var cur strings.Builder

	flush := func() {
		text := collapseSpaces(cur.String())
		if text != "" {
			paras = append(paras, text)
		}
		cur.Reset()
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skipTags[n.Data] {
			return
		}
		if n.Type == html.ElementNode && n.Data == "br" {
			flush()
			return
		}
		isBlock := n.Type == html.ElementNode && blockTags[n.Data]
		if isBlock {
			flush()
		}
		if n.Type == html.TextNode {
			cur.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if isBlock {
			flush()
		}
	}
	walk(node)
	flush()
	return paras
}

// collapseSpaces trims and collapses internal whitespace to single spaces.
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
```

Create `internal/epub/epub.go`:
```go
// Package epub parses EPUB files into plain-text chapters.
package epub

// Book is a fully parsed EPUB ready for rendering.
type Book struct {
	Title    string
	Author   string
	Chapters []Chapter
}

// Chapter is one spine item reduced to plain-text paragraphs.
type Chapter struct {
	Title      string
	Paragraphs []string
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/epub/ -run TestHTMLToParagraphs`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/epub
git commit -m "feat(epub): xhtml to plain-text paragraph extraction"
```

---

## Task 2: container.xml 与 OPF 解析

**Files:**
- Create: `internal/epub/container.go`
- Create: `internal/epub/opf.go`
- Test: `internal/epub/opf_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/epub/opf_test.go`:
```go
package epub

import (
	"reflect"
	"testing"
)

func TestParseContainerOPFPath(t *testing.T) {
	xml := `<?xml version="1.0"?>
	<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	  <rootfiles>
	    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
	  </rootfiles>
	</container>`
	got, err := parseContainer([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if got != "OEBPS/content.opf" {
		t.Fatalf("got %q want OEBPS/content.opf", got)
	}
}

func TestParseOPFSpineOrder(t *testing.T) {
	xml := `<?xml version="1.0"?>
	<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
	  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
	    <dc:title>测试之书</dc:title>
	    <dc:creator>张三</dc:creator>
	  </metadata>
	  <manifest>
	    <item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
	    <item id="c2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
	    <item id="css" href="style.css" media-type="text/css"/>
	  </manifest>
	  <spine>
	    <itemref idref="c2"/>
	    <itemref idref="c1"/>
	  </spine>
	</package>`
	meta, hrefs, err := parseOPF([]byte(xml), "OEBPS")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "测试之书" || meta.Author != "张三" {
		t.Fatalf("bad meta: %+v", meta)
	}
	want := []string{"OEBPS/ch2.xhtml", "OEBPS/ch1.xhtml"}
	if !reflect.DeepEqual(hrefs, want) {
		t.Fatalf("got %#v want %#v", hrefs, want)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/epub/ -run "TestParseContainer|TestParseOPF"`
Expected: FAIL（`parseContainer` / `parseOPF` undefined）

- [ ] **Step 3: 实现 container.xml 解析**

Create `internal/epub/container.go`:
```go
package epub

import "encoding/xml"

// parseContainer returns the OPF rootfile path from META-INF/container.xml.
func parseContainer(data []byte) (string, error) {
	var c struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", err
	}
	if len(c.Rootfiles) == 0 {
		return "", errNoRootfile
	}
	return c.Rootfiles[0].FullPath, nil
}
```

- [ ] **Step 4: 实现 OPF 解析**

Create `internal/epub/opf.go`:
```go
package epub

import (
	"encoding/xml"
	"errors"
	"path"
)

var (
	errNoRootfile = errors.New("epub: no rootfile in container.xml")
	errNoSpine    = errors.New("epub: empty spine")
)

// Meta holds book-level metadata extracted from the OPF.
type Meta struct {
	Title  string
	Author string
}

// parseOPF parses the OPF package document. opfDir is the directory of the OPF
// file inside the archive, used to resolve relative hrefs to archive paths.
// It returns metadata and the ordered list of chapter file paths (from spine).
func parseOPF(data []byte, opfDir string) (Meta, []string, error) {
	var pkg struct {
		Metadata struct {
			Title   string `xml:"title"`
			Creator string `xml:"creator"`
		} `xml:"metadata"`
		Manifest []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return Meta{}, nil, err
	}
	if len(pkg.Spine) == 0 {
		return Meta{}, nil, errNoSpine
	}
	hrefByID := make(map[string]string, len(pkg.Manifest))
	for _, it := range pkg.Manifest {
		hrefByID[it.ID] = it.Href
	}
	var hrefs []string
	for _, ref := range pkg.Spine {
		href, ok := hrefByID[ref.IDRef]
		if !ok {
			continue
		}
		hrefs = append(hrefs, joinArchive(opfDir, href))
	}
	meta := Meta{Title: pkg.Metadata.Title, Author: pkg.Metadata.Creator}
	return meta, hrefs, nil
}

// joinArchive joins a directory and a relative href into a clean archive path
// using forward slashes (zip archives always use "/").
func joinArchive(dir, href string) string {
	if dir == "" || dir == "." {
		return path.Clean(href)
	}
	return path.Clean(dir + "/" + href)
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/epub/ -run "TestParseContainer|TestParseOPF"`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/epub
git commit -m "feat(epub): parse container.xml and OPF spine"
```

---

## Task 3: EPUB 解析入口（zip 装配）

**Files:**
- Modify: `internal/epub/epub.go`
- Create: `internal/epub/testdata/sample.epub`（用代码生成，见 Step 1）
- Test: `internal/epub/epub_test.go`

- [ ] **Step 1: 写一个生成 fixture epub 的辅助 + 失败测试**

Create `internal/epub/epub_test.go`:
```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/epub/ -run TestParseFull`
Expected: FAIL（`Parse` undefined）

- [ ] **Step 3: 实现 Parse 入口**

Append to `internal/epub/epub.go`:
```go
import (
	"archive/zip"
	"io"
	"path"
	"strings"
)

// Parse opens an EPUB file and returns the fully parsed Book.
func Parse(filename string) (*Book, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	containerData, err := readZipFile(files["META-INF/container.xml"])
	if err != nil {
		return nil, err
	}
	opfPath, err := parseContainer(containerData)
	if err != nil {
		return nil, err
	}
	opfData, err := readZipFile(files[opfPath])
	if err != nil {
		return nil, err
	}
	meta, hrefs, err := parseOPF(opfData, path.Dir(opfPath))
	if err != nil {
		return nil, err
	}

	book := &Book{Title: meta.Title, Author: meta.Author}
	for _, href := range hrefs {
		data, err := readZipFile(files[href])
		if err != nil {
			continue // skip unreadable chapter rather than failing whole book
		}
		paras := htmlToParagraphs(string(data))
		if len(paras) == 0 {
			continue
		}
		title := paras[0]
		book.Chapters = append(book.Chapters, Chapter{Title: title, Paragraphs: paras})
	}
	return book, nil
}

// readZipFile reads the full contents of a zip entry, erroring if it is nil.
func readZipFile(f *zip.File) ([]byte, error) {
	if f == nil {
		return nil, errMissingEntry
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
```

Add the sentinel error to `internal/epub/opf.go` var block:
```go
	errMissingEntry = errors.New("epub: required archive entry missing")
```
(Add `errMissingEntry` to the existing `var ( ... )` block alongside `errNoRootfile`, `errNoSpine`.)

Note: the `strings` import in epub.go is required by other helpers added later; if unused now, remove it to keep the build green. Keep only imports actually referenced (`archive/zip`, `io`, `path`).

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/epub/...`
Expected: PASS（全部 epub 测试）

- [ ] **Step 5: 提交**

```bash
git add internal/epub
git commit -m "feat(epub): full Parse pipeline assembling Book from zip"
```

---

## Task 4: CJK 宽度与段落换行

**Files:**
- Create: `internal/render/width.go`
- Create: `internal/render/wrap.go`
- Test: `internal/render/wrap_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/render/wrap_test.go`:
```go
package render

import (
	"reflect"
	"testing"
)

func TestWrapASCIIWord(t *testing.T) {
	got := WrapParagraph("hello world foo", 11)
	want := []string{"hello world", "foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestWrapCJKBreaksAnywhere(t *testing.T) {
	// 6 双宽字 = 12 cells; maxWidth 6 cells = 3 chars per line.
	got := WrapParagraph("一二三四五六", 6)
	want := []string{"一二三", "四五六"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestWrapMixed(t *testing.T) {
	// "ab一二cd" widths: a1 b1 一2 二2 c1 d1 = 8; maxWidth 4.
	got := WrapParagraph("ab一二cd", 4)
	want := []string{"ab一", "二cd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestWrapEmpty(t *testing.T) {
	got := WrapParagraph("", 10)
	want := []string{""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/render/ -run TestWrap`
Expected: FAIL（`WrapParagraph` undefined）

- [ ] **Step 3: 实现宽度条件**

Create `internal/render/width.go`:
```go
// Package render handles CJK-aware text wrapping and pagination.
package render

import "github.com/mattn/go-runewidth"

// cjk measures display width with ambiguous-width characters treated as narrow,
// which matches how Windows Terminal and most modern terminals render them.
// CJK ideographs and full-width punctuation remain width 2.
var cjk = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// RuneWidth returns the terminal cell width of r (1 or 2).
func RuneWidth(r rune) int { return cjk.RuneWidth(r) }

// StringWidth returns the total terminal cell width of s.
func StringWidth(s string) int { return cjk.StringWidth(s) }
```

- [ ] **Step 4: 实现换行**

Create `internal/render/wrap.go`:
```go
package render

// WrapParagraph wraps a single paragraph to lines no wider than maxWidth cells.
// CJK (width-2) runes may break between any two characters; ASCII words break on
// spaces and are not split unless a single word exceeds maxWidth. An empty
// paragraph yields a single empty line so paragraph spacing is preserved.
func WrapParagraph(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	var lines []string
	var line []rune
	lineW := 0
	flush := func() {
		// trim trailing spaces before breaking
		for len(line) > 0 && line[len(line)-1] == ' ' {
			line = line[:len(line)-1]
		}
		lines = append(lines, string(line))
		line = line[:0]
		lineW = 0
	}

	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == ' ' {
			if lineW == 0 {
				i++ // skip leading space
				continue
			}
			if lineW+1 > maxWidth {
				flush()
				i++
				continue
			}
			line = append(line, ' ')
			lineW++
			i++
			continue
		}

		// Build the next token: one wide rune, or a run of narrow non-space runes.
		var tok []rune
		tokW := 0
		if RuneWidth(r) == 2 {
			tok = append(tok, r)
			tokW = 2
			i++
		} else {
			for i < len(runes) && runes[i] != ' ' && RuneWidth(runes[i]) != 2 {
				tok = append(tok, runes[i])
				tokW += RuneWidth(runes[i])
				i++
			}
		}

		if lineW > 0 && lineW+tokW > maxWidth {
			flush()
		}
		if tokW > maxWidth {
			// token longer than a full line: hard-split by rune
			for _, rr := range tok {
				w := RuneWidth(rr)
				if lineW > 0 && lineW+w > maxWidth {
					flush()
				}
				line = append(line, rr)
				lineW += w
			}
		} else {
			line = append(line, tok...)
			lineW += tokW
		}
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/render/ -run TestWrap`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/render
git commit -m "feat(render): CJK-aware paragraph wrapping"
```

---

## Task 5: 章节排版为行序列 + 分页

**Files:**
- Create: `internal/render/page.go`
- Test: `internal/render/page_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/render/page_test.go`:
```go
package render

import (
	"reflect"
	"testing"
)

func TestLayoutChapterInsertsBlankBetweenParagraphs(t *testing.T) {
	paras := []string{"一二三四", "abcd"}
	got := LayoutChapter(paras, 4)
	// "一二三四" -> ["一二","三四"]; blank; "abcd" -> ["abcd"]
	want := []string{"一二", "三四", "", "abcd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestPaginate(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got := Paginate(lines, 2)
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestLineToPage(t *testing.T) {
	if got := LineToPage(5, 2); got != 2 {
		t.Fatalf("LineToPage(5,2)=%d want 2", got)
	}
	if got := LineToPage(0, 2); got != 0 {
		t.Fatalf("LineToPage(0,2)=%d want 0", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/render/ -run "TestLayout|TestPaginate|TestLineToPage"`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现排版与分页**

Create `internal/render/page.go`:
```go
package render

// LayoutChapter wraps every paragraph to width and joins them into a single
// flat slice of display lines, inserting one blank line between paragraphs.
func LayoutChapter(paragraphs []string, width int) []string {
	var lines []string
	for idx, p := range paragraphs {
		if idx > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, WrapParagraph(p, width)...)
	}
	if lines == nil {
		return []string{}
	}
	return lines
}

// Paginate splits lines into pages of at most height lines each.
func Paginate(lines []string, height int) [][]string {
	if height <= 0 {
		return [][]string{lines}
	}
	var pages [][]string
	for i := 0; i < len(lines); i += height {
		end := i + height
		if end > len(lines) {
			end = len(lines)
		}
		pages = append(pages, lines[i:end])
	}
	return pages
}

// LineToPage returns the zero-based page index containing the given line index.
func LineToPage(line, height int) int {
	if height <= 0 {
		return 0
	}
	return line / height
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/render/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/render
git commit -m "feat(render): chapter layout and pagination with stable line index"
```

---

## Task 6: 伪装风格接口与三种风格

**Files:**
- Create: `internal/disguise/theme.go`
- Test: `internal/disguise/theme_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/disguise/theme_test.go`:
```go
package disguise

import (
	"strings"
	"testing"
)

func TestStylesRegistered(t *testing.T) {
	for _, name := range []string{"log", "build", "git"} {
		if ThemeByName(name) == nil {
			t.Fatalf("theme %q not registered", name)
		}
	}
}

func TestThemeByNameFallback(t *testing.T) {
	if ThemeByName("nonsense").Name() != "log" {
		t.Fatal("unknown theme should fall back to log")
	}
}

func TestNextStyleCycles(t *testing.T) {
	if NextStyle("log") != "build" || NextStyle("build") != "git" || NextStyle("git") != "log" {
		t.Fatal("style cycle wrong")
	}
}

// LogLine for a deterministic seed must be stable and contain the payload.
func TestLogThemePrefixDeterministic(t *testing.T) {
	th := ThemeByName("log")
	a := th.LinePrefix(42)
	b := th.LinePrefix(42)
	if a != b {
		t.Fatalf("prefix not deterministic: %q vs %q", a, b)
	}
	if !strings.Contains(a, "INFO") && !strings.Contains(a, "DEBUG") && !strings.Contains(a, "WARN") {
		t.Fatalf("log prefix missing level: %q", a)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/disguise/ -run "TestStyles|TestTheme|TestNext|TestLog"`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现 Theme 接口与三风格**

Create `internal/disguise/theme.go`:
```go
// Package disguise renders novel text so it resembles developer tool output.
package disguise

import "fmt"

// Theme produces fake "work output" decoration for a given style.
type Theme interface {
	// Name returns the style id ("log"/"build"/"git").
	Name() string
	// LinePrefix returns a deterministic decoration placed before a body line.
	// seed makes the decoration vary per line while staying reproducible.
	LinePrefix(seed int) string
	// Header / Footer return chrome lines for shell-disguise mode.
	Header(width int, status string) string
	Footer(width int, status string) string
	// BossLine returns a pure fake-output line (no novel content) for seed.
	BossLine(seed int) string
}

// styleOrder defines the Tab cycle order.
var styleOrder = []string{"log", "build", "git"}

var registry = map[string]Theme{
	"log":   logTheme{},
	"build": buildTheme{},
	"git":   gitTheme{},
}

// ThemeByName returns the theme for name, falling back to "log" if unknown.
func ThemeByName(name string) Theme {
	if th, ok := registry[name]; ok {
		return th
	}
	return registry["log"]
}

// NextStyle returns the next style id in the Tab cycle.
func NextStyle(name string) string {
	for i, s := range styleOrder {
		if s == name {
			return styleOrder[(i+1)%len(styleOrder)]
		}
	}
	return styleOrder[0]
}

// --- log theme ---

type logTheme struct{}

func (logTheme) Name() string { return "log" }

var logLevels = []string{"INFO", "DEBUG", "INFO", "WARN", "INFO"}
var logClasses = []string{"OrderSvc", "CacheManager", "HttpPool", "AuthFilter", "TaskRunner"}

func (logTheme) LinePrefix(seed int) string {
	ts := fmt.Sprintf("%02d:%02d:%02d", 8+(seed/3600)%10, (seed/60)%60, seed%60)
	lvl := logLevels[seed%len(logLevels)]
	cls := logClasses[(seed/7)%len(logClasses)]
	return fmt.Sprintf("[%s] %-5s %s - ", ts, lvl, cls)
}

func (logTheme) Header(width int, status string) string {
	return fitLine("app.log — tail -f   "+status, width)
}
func (logTheme) Footer(width int, status string) string {
	return fitLine("INFO  watching for changes   "+status, width)
}
func (logTheme) BossLine(seed int) string {
	return fitLine(logTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- build theme ---

type buildTheme struct{}

func (buildTheme) Name() string { return "build" }

var buildVerbs = []string{"Compiling", "Linking", "Bundling", "Transpiling", "Resolving"}
var buildMods = []string{"core", "ui", "data", "net", "auth", "render"}

func (buildTheme) LinePrefix(seed int) string {
	return fmt.Sprintf("> %s %s … ", buildVerbs[seed%len(buildVerbs)], buildMods[(seed/3)%len(buildMods)])
}
func (buildTheme) Header(width int, status string) string {
	return fitLine("> gradle build   "+status, width)
}
func (buildTheme) Footer(width int, status string) string {
	return fitLine("BUILD SUCCESSFUL in 12s   "+status, width)
}
func (buildTheme) BossLine(seed int) string {
	return fitLine(buildTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- git theme ---

type gitTheme struct{}

func (gitTheme) Name() string { return "git" }

func (gitTheme) LinePrefix(seed int) string {
	if seed%4 == 0 {
		return "+ // "
	}
	return "  // "
}
func (gitTheme) Header(width int, status string) string {
	return fitLine("git diff --stat   "+status, width)
}
func (gitTheme) Footer(width int, status string) string {
	return fitLine("3 files changed, 128 insertions(+)   "+status, width)
}
func (gitTheme) BossLine(seed int) string {
	return fitLine(fmt.Sprintf("commit %08x  %s", seed*2654435761, bossPayload[seed%len(bossPayload)]), 0)
}

// bossPayload are neutral fake messages used only in boss-screen lines.
var bossPayload = []string{
	"processing batch 0x1f", "cache hit ratio 0.92", "retry attempt 1/3",
	"flushing write buffer", "connection pool resized", "gc pause 4ms",
	"index rebuilt", "checkpoint committed", "heartbeat ok",
}
```

(Note: `fitLine` is defined in Task 7 / shell.go. To keep this task self-contained and the build green, also create `internal/disguise/util.go` now — see Step 4.)

- [ ] **Step 4: 创建共享工具 fitLine（被多文件使用）**

Create `internal/disguise/util.go`:
```go
package disguise

import "moyureader/internal/render"

// fitLine truncates or returns s. When width<=0 it returns s unchanged; when
// width>0 it truncates to width display cells (never pads, to keep golden tests
// stable and let the UI layer own padding/coloring).
func fitLine(s string, width int) string {
	if width <= 0 || render.StringWidth(s) <= width {
		return s
	}
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		rw := render.RuneWidth(r)
		if w+rw > width {
			return string(runes[:i])
		}
		w += rw
	}
	return s
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/disguise/ -run "TestStyles|TestTheme|TestNext|TestLog"`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/disguise
git commit -m "feat(disguise): theme interface with log/build/git styles"
```

---

## Task 7: 两种正文渲染模式（shell / inline）

**Files:**
- Create: `internal/disguise/shell.go`
- Create: `internal/disguise/inline.go`
- Test: `internal/disguise/render_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/disguise/render_test.go`:
```go
package disguise

import (
	"strings"
	"testing"
)

func TestRenderInlinePrefixesEachLine(t *testing.T) {
	th := ThemeByName("log")
	body := []string{"林尘睁开眼", "看见月光"}
	out := RenderInline(th, body, 100)
	if len(out) != 2 {
		t.Fatalf("want 2 lines got %d", len(out))
	}
	if !strings.HasSuffix(out[0], "林尘睁开眼") {
		t.Fatalf("line0 should end with payload: %q", out[0])
	}
	if !strings.Contains(out[0], " - ") {
		t.Fatalf("line0 should contain log prefix: %q", out[0])
	}
}

func TestRenderShellWrapsBodyWithChrome(t *testing.T) {
	th := ThemeByName("build")
	body := []string{"正文一", "正文二"}
	out := RenderShell(th, body, 40, "ch.1/10")
	if len(out) < 4 {
		t.Fatalf("shell output too short: %v", out)
	}
	if !strings.Contains(out[0], "gradle") {
		t.Fatalf("first line should be build header: %q", out[0])
	}
	if out[1] != "正文一" || out[2] != "正文二" {
		t.Fatalf("body not preserved verbatim: %#v", out)
	}
	last := out[len(out)-1]
	if !strings.Contains(last, "SUCCESSFUL") {
		t.Fatalf("last line should be build footer: %q", last)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/disguise/ -run "TestRender"`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现 shell 模式**

Create `internal/disguise/shell.go`:
```go
package disguise

// RenderShell wraps already-wrapped body lines with a theme header and footer.
// The body is returned verbatim (the UI layer applies width/padding/color); the
// status string is embedded into header/footer chrome. This is reading mode A.
func RenderShell(th Theme, body []string, width int, status string) []string {
	out := make([]string, 0, len(body)+2)
	out = append(out, th.Header(width, status))
	out = append(out, body...)
	out = append(out, th.Footer(width, status))
	return out
}
```

- [ ] **Step 4: 实现 inline 模式**

Create `internal/disguise/inline.go`:
```go
package disguise

// RenderInline turns each body line into a fake log/build/git line by prefixing
// it with theme decoration. The novel text becomes the "message" payload. This
// is reading mode B (highest stealth). width truncates the whole line; pass 0
// to leave untruncated (callers that own padding should pass 0).
func RenderInline(th Theme, body []string, width int) []string {
	out := make([]string, len(body))
	for i, line := range body {
		out[i] = fitLine(th.LinePrefix(i)+line, width)
	}
	return out
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/disguise/...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/disguise
git commit -m "feat(disguise): shell and inline body render modes"
```

---

## Task 8: 老板键假屏生成

**Files:**
- Create: `internal/disguise/boss.go`
- Test: `internal/disguise/boss_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/disguise/boss_test.go`:
```go
package disguise

import (
	"strings"
	"testing"
)

func TestBossScreenContainsNoNovelText(t *testing.T) {
	th := ThemeByName("log")
	novel := "林尘睁开眼"
	lines := BossScreen(th, 0, 20)
	if len(lines) != 20 {
		t.Fatalf("want 20 lines got %d", len(lines))
	}
	for _, l := range lines {
		if strings.Contains(l, novel) {
			t.Fatalf("boss screen leaked novel text: %q", l)
		}
	}
}

func TestBossScreenScrollsByTick(t *testing.T) {
	th := ThemeByName("log")
	a := BossScreen(th, 0, 5)
	b := BossScreen(th, 1, 5)
	if a[0] == b[0] {
		t.Fatal("boss screen should scroll between ticks")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/disguise/ -run TestBoss`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现老板屏**

Create `internal/disguise/boss.go`:
```go
package disguise

// BossScreen returns `height` fake-output lines for the given theme. `tick`
// advances the stream so repeated calls with increasing tick scroll the
// content, simulating a live, running process. No novel text is ever included.
func BossScreen(th Theme, tick, height int) []string {
	lines := make([]string, height)
	for i := 0; i < height; i++ {
		lines[i] = th.BossLine(tick + i)
	}
	return lines
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/disguise/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/disguise
git commit -m "feat(disguise): scrolling boss-key fake screen"
```

---

## Task 9: 持久化数据模型

**Files:**
- Create: `internal/store/model.go`
- Test: `internal/store/model_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/store/model_test.go`:
```go
package store

import "testing"

func TestFindByIDAndPromote(t *testing.T) {
	lib := &Library{Books: []BookEntry{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
	}}
	e := lib.FindByID("b")
	if e == nil || e.Title != "B" {
		t.Fatalf("FindByID failed: %v", e)
	}
	if lib.FindByID("zzz") != nil {
		t.Fatal("missing id should return nil")
	}
}

func TestDefaultPrefs(t *testing.T) {
	lib := NewLibrary()
	if lib.Global.Style != "log" || lib.Global.Mode != "shell" {
		t.Fatalf("bad defaults: %+v", lib.Global)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/store/ -run "TestFindByID|TestDefaultPrefs"`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现模型**

Create `internal/store/model.go`:
```go
// Package store persists the bookshelf and reading progress as JSON.
package store

// Prefs holds disguise style ("log"/"build"/"git") and reading mode
// ("shell"/"inline").
type Prefs struct {
	Style string `json:"style"`
	Mode  string `json:"mode"`
}

// Progress is a stable reading position: chapter index + display-line index.
type Progress struct {
	Chapter int `json:"chapter"`
	Line    int `json:"line"`
}

// BookEntry is one imported book.
type BookEntry struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Author       string   `json:"author"`
	File         string   `json:"file"` // relative to data dir, e.g. "books/<id>.epub"
	AddedAt      string   `json:"addedAt"`
	LastOpenedAt string   `json:"lastOpenedAt"`
	Progress     Progress `json:"progress"`
	Prefs        Prefs    `json:"prefs"`
	Broken       bool     `json:"broken,omitempty"`
}

// Library is the whole bookshelf plus global defaults.
type Library struct {
	LastBookID string      `json:"lastBookId"`
	Global     Prefs       `json:"global"`
	Books      []BookEntry `json:"books"`
}

// NewLibrary returns an empty library with sane default prefs.
func NewLibrary() *Library {
	return &Library{Global: Prefs{Style: "log", Mode: "shell"}}
}

// FindByID returns a pointer to the matching book entry, or nil.
func (l *Library) FindByID(id string) *BookEntry {
	for i := range l.Books {
		if l.Books[i].ID == id {
			return &l.Books[i]
		}
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/store/ -run "TestFindByID|TestDefaultPrefs"`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/store
git commit -m "feat(store): library data model"
```

---

## Task 10: 加载 / 原子保存 / 损坏恢复

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/store/store_test.go`:
```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/store/ -run "TestLoad|TestSave"`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现 store**

Create `internal/store/store.go`:
```go
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/store/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/store
git commit -m "feat(store): atomic load/save with corruption recovery"
```

---

## Task 11: 导入 epub 与进度更新

**Files:**
- Modify: `internal/store/store.go`
- Test: `internal/store/import_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/store/import_test.go`:
```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/store/ -run "TestImport|TestUpdateProgress"`
Expected: FAIL（未定义）

- [ ] **Step 3: 实现 Import 与 UpdateProgress**

Append to `internal/store/store.go`:
```go
import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"time"
)

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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/store/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/store
git commit -m "feat(store): import epub copy and progress updates"
```

---

## Task 12: 核心引擎全量验证

**Files:**
- Create: `internal/render/doc.go`（占位，确保包注释；可选）

- [ ] **Step 1: 运行全部测试 + vet**

Run:
```bash
go vet ./...
go test ./...
```
Expected: 全部 PASS，无 vet 报错。

- [ ] **Step 2: 确认依赖整洁**

Run: `go mod tidy && git diff --stat go.mod go.sum`
Expected: 仅 `go-runewidth` 与 `x/net/html`（及其传递依赖）出现。

- [ ] **Step 3: 提交**

```bash
git add -A
git commit -m "chore: tidy modules and verify core engine green"
```

---

## Self-Review（计划作者已核对）

- **Spec 覆盖**：epub 解析(Task1-3) ✓ / CJK 排版分页(Task4-5) ✓ / 三风格×两模式(Task6-7) ✓ / 老板屏(Task8) ✓ / data/ 持久化+导入+进度+损坏恢复(Task9-11) ✓ / 健壮性：坏章节跳过、缺文件报错、library 损坏恢复 ✓。
- **类型一致性**：`Book/Chapter`、`Theme` 接口方法（`Name/LinePrefix/Header/Footer/BossLine`）、`Library/BookEntry/Progress/Prefs`、`Store.{Load,Save,Import,Dir,BooksDir}`、`UpdateProgress`、`render.{WrapParagraph,LayoutChapter,Paginate,LineToPage,StringWidth,RuneWidth}`、`disguise.{ThemeByName,NextStyle,RenderShell,RenderInline,BossScreen,fitLine}` 在各任务间签名一致。
- **占位符扫描**：无 TBD/TODO；每个代码步骤含完整可编译代码。
- **遗留给计划二**：内联流式 CLI、全屏 TUI（书架/阅读/老板键覆盖层）、main 子命令装配、lipgloss 着色、目录跳转 `g`。这些都消费本计划导出的 API。

## 计划二预告（核心跑通后再写）

- `internal/ui`：bubbletea Model，三界面状态机，按键路由，调用 `disguise.RenderShell/RenderInline`、`render.LayoutChapter/Paginate`、`store.UpdateProgress`
- `internal/stream`：内联流式输出，复用 `disguise.RenderInline`
- `cmd/reader/main.go`：解析 `reader / <book.epub> / stream / import / list` 子命令，定位 exe 同级 `data/`
