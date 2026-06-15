# 摸鱼阅读器 v0.4 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 v0.4：引入 Go 标准库式「格式驱动注册」架构，新增 TXT 格式支持（UTF-8/GB18030 自动识别 + 行锚定分章），并新增 docker/npm/pytest 三个伪装风格。

**Architecture:** 新建中立入口包 `internal/book`（拥有 `Book`/`Chapter` 类型 + 扩展名注册表 + `Open`）；`epub`（迁入 `internal/book/epub`）与新 `txt`（`internal/book/txt`）作为后端在 `init()` 中自注册；`internal/book/formats` 空导入各后端触发注册；`ui`/`stream`/`cmd` 只依赖 `book.Open`。伪装主题按现有 `Theme` 接口扩展。

**Tech Stack:** Go、现有依赖；GB18030 解码用 `golang.org/x/text/encoding/simplifiedchinese`（go.mod 已含 x/text，仅需 `go mod tidy` 提升为直接依赖）。

**运行 Go：** 命令前加 `export PATH="/d/develop/Go/bin:$PATH"`。

---

## 文件结构

```
internal/book/book.go              + Book/Chapter/Parser/Register/Open（新建）
internal/book/book_test.go         + 注册/分发测试（新建）
internal/book/epub/                ← 由 internal/epub 整目录迁入（git mv）
internal/book/epub/epub.go         改：Parse 返回 *book.Book + init() 注册（修改）
internal/book/txt/txt.go           + TXT 解析后端 + init() 注册（新建）
internal/book/txt/txt_test.go      + 编码/分章/兜底测试（新建）
internal/book/formats/formats.go   + 空导入各后端（新建）
internal/store/store.go            Import 保留源扩展名（修改）
internal/store/store_import_test.go + 扩展名保留测试（新建）
internal/disguise/theme_extra.go   + docker/npm/pytest 三主题（新建）
internal/disguise/theme.go         styleOrder/registry 纳入三主题（修改）
internal/disguise/theme_extra_test.go + 三主题测试（新建）
internal/ui/reader.go              epub.Book→book.Book（修改）
internal/ui/toc.go                 epub.Book→book.Book（修改）
internal/ui/model.go               epub→book、openBook 用 book.Open（修改）
internal/stream/stream.go          epub.Book→book.Book（修改）
cmd/reader/main.go                 epub.Parse→book.Open + 空导入 formats（修改）
cmd/reader/txt_integration_test.go + .txt 端到端测试（新建）
```

---

## Task 1: `internal/book` 核心（类型 + 注册表 + Open）

**Files:**
- Create: `internal/book/book.go`
- Create: `internal/book/book_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/book/book_test.go`:
```go
package book

import (
	"strings"
	"testing"
)

func TestOpenRoutesByExtensionCaseInsensitive(t *testing.T) {
	Register(".zz1", func(path string) (*Book, error) {
		return &Book{Title: "routed:" + path}, nil
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/book/`
Expected: FAIL（`Book`/`Register`/`Open` undefined）

- [ ] **Step 3: 实现**

Create `internal/book/book.go`:
```go
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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/book/`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/book/book.go internal/book/book_test.go
git commit -m "feat(book): format-agnostic Book model + extension registry"
```

---

## Task 2: 迁移 epub 后端 + 改造所有调用方（原子改动）

> 纯重构：把类型上移到 `book`，`epub` 变成后端，所有调用方改用 `book.Open`/`book.Book`。一次提交内完成以保持编译与测试通过。无新行为，验证手段是既有测试全绿。

**Files:**
- Move: `internal/epub/` → `internal/book/epub/`（`git mv`）
- Modify: `internal/book/epub/epub.go`
- Create: `internal/book/formats/formats.go`
- Modify: `internal/ui/reader.go`, `internal/ui/toc.go`, `internal/ui/model.go`
- Modify: `internal/stream/stream.go`
- Modify: `cmd/reader/main.go`

- [ ] **Step 1: 迁移 epub 目录**

Run:
```bash
git mv internal/epub internal/book/epub
```

- [ ] **Step 2: 改 `internal/book/epub/epub.go`**

Replace the import block + type defs + `Parse` head/body. Specifically:

Change the imports from:
```go
import (
	"archive/zip"
	"io"
	"path"
)
```
to:
```go
import (
	"archive/zip"
	"io"
	"path"

	"moyureader/internal/book"
)

func init() { book.Register(".epub", Parse) }
```

Delete the `Book` and `Chapter` type definitions (they now live in package `book`):
```go
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

Change `Parse`'s signature and the local variable (rename `book`→`b` to avoid shadowing the imported package). Replace:
```go
// Parse opens an EPUB file and returns the fully parsed Book.
func Parse(filename string) (*Book, error) {
```
with:
```go
// Parse opens an EPUB file and returns the fully parsed Book.
func Parse(filename string) (*book.Book, error) {
```

Replace:
```go
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
```
with:
```go
	b := &book.Book{Title: meta.Title, Author: meta.Author}
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
		b.Chapters = append(b.Chapters, book.Chapter{Title: title, Paragraphs: paras})
	}
	return b, nil
```

> `internal/book/epub/epub_test.go` 无需改动：它用局部变量 `book` 接收 `Parse` 返回值并只访问其字段，不引用包标识符。

- [ ] **Step 3: 新建组合根 `internal/book/formats/formats.go`**

```go
// Package formats wires all built-in book format backends. Import it (usually
// for side effects) to make every supported format available via book.Open.
package formats

import (
	_ "moyureader/internal/book/epub"
)
```

- [ ] **Step 4: 改 `internal/ui/reader.go`**

Change the import `"moyureader/internal/epub"` → `"moyureader/internal/book"`. Then replace the struct field type and constructor param:

Replace:
```go
type ReaderView struct {
	book    *epub.Book
```
with:
```go
type ReaderView struct {
	book    *book.Book
```

Replace:
```go
func NewReaderView(b *epub.Book, p store.Progress, prefs store.Prefs, width, height int) *ReaderView {
```
with:
```go
func NewReaderView(b *book.Book, p store.Progress, prefs store.Prefs, width, height int) *ReaderView {
```

- [ ] **Step 5: 改 `internal/ui/toc.go`**

Change import `"moyureader/internal/epub"` → `"moyureader/internal/book"`. Replace:
```go
func NewTOCView(b *epub.Book, current int) *TOCView {
```
with:
```go
func NewTOCView(b *book.Book, current int) *TOCView {
```

- [ ] **Step 6: 改 `internal/ui/model.go`**

Change import `"moyureader/internal/epub"` → `"moyureader/internal/book"`. Replace the struct field:
```go
	shelf  *ShelfView
	reader *ReaderView
	book   *epub.Book
	bookID string
```
with:
```go
	shelf  *ShelfView
	reader *ReaderView
	book   *book.Book
	bookID string
```

Replace the body of `openBook` that parses (rename local `book`→`bk`):
```go
	book, err := epub.Parse(filepath.Join(m.st.Dir(), filepath.FromSlash(e.File)))
	if err != nil {
		e.Broken = true
		_ = m.st.Save(m.lib)
		m.status = "这本书打不开（已标记损坏）"
		return
	}
	m.reader = NewReaderView(book, e.Progress, e.Prefs, m.width, m.height)
	m.book = book
```
with:
```go
	bk, err := book.Open(filepath.Join(m.st.Dir(), filepath.FromSlash(e.File)))
	if err != nil {
		e.Broken = true
		_ = m.st.Save(m.lib)
		m.status = "这本书打不开（已标记损坏）"
		return
	}
	m.reader = NewReaderView(bk, e.Progress, e.Prefs, m.width, m.height)
	m.book = bk
```

- [ ] **Step 7: 改 `internal/stream/stream.go`**

Change import `"moyureader/internal/epub"` → `"moyureader/internal/book"`. Replace:
```go
type Streamer struct {
	book    *epub.Book
```
with:
```go
type Streamer struct {
	book    *book.Book
```

Replace:
```go
func NewStreamer(b *epub.Book, p store.Progress, style string, height int) *Streamer {
```
with:
```go
func NewStreamer(b *book.Book, p store.Progress, style string, height int) *Streamer {
```

- [ ] **Step 8: 改 `cmd/reader/main.go`**

Change import `"moyureader/internal/epub"` → `"moyureader/internal/book"` and add a blank import for the format wiring. The import block becomes:
```go
import (
	"fmt"
	"os"
	"path/filepath"

	"moyureader/internal/book"
	_ "moyureader/internal/book/formats"
	"moyureader/internal/store"
	"moyureader/internal/stream"
	"moyureader/internal/ui"
)
```

Replace `importPath` (rename local `book`→`bk`):
```go
func importPath(st *store.Store, lib *store.Library, path string) (*store.BookEntry, error) {
	book, err := epub.Parse(path)
	if err != nil {
		return nil, err
	}
	entry, err := st.Import(lib, path, book.Title, book.Author)
	if err != nil {
		return nil, err
	}
	if err := st.Save(lib); err != nil {
		return nil, err
	}
	return entry, nil
}
```
with:
```go
func importPath(st *store.Store, lib *store.Library, path string) (*store.BookEntry, error) {
	bk, err := book.Open(path)
	if err != nil {
		return nil, err
	}
	entry, err := st.Import(lib, path, bk.Title, bk.Author)
	if err != nil {
		return nil, err
	}
	if err := st.Save(lib); err != nil {
		return nil, err
	}
	return entry, nil
}
```

Replace in `runStream` (rename local `book`→`bk`):
```go
	book, err := epub.Parse(filepath.Join(st.Dir(), filepath.FromSlash(entry.File)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "解析失败:", err)
		os.Exit(1)
	}
	s := stream.NewStreamer(book, entry.Progress, entry.Prefs.Style, 18)
```
with:
```go
	bk, err := book.Open(filepath.Join(st.Dir(), filepath.FromSlash(entry.File)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "解析失败:", err)
		os.Exit(1)
	}
	s := stream.NewStreamer(bk, entry.Progress, entry.Prefs.Style, 18)
```

- [ ] **Step 9: 全量构建 + 测试确认全绿**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go build ./...
go test ./...
```
Expected: 全部 PASS（含迁移后的 `internal/book/epub` 测试）。

- [ ] **Step 10: 提交**

```bash
git add -A
git commit -m "refactor: introduce book package + dispatch; migrate epub to backend"
```

---

## Task 3: `store.Import` 保留源扩展名

**Files:**
- Modify: `internal/store/store.go`
- Create: `internal/store/store_import_test.go`

- [ ] **Step 1: 写失败测试**

Create `internal/store/store_import_test.go`:
```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/store/ -run TestImportPreservesExtension`
Expected: FAIL（当前固定写成 `.epub`）

- [ ] **Step 3: 实现**

In `internal/store/store.go`, add `"strings"` to the import block (alongside `"path/filepath"`).

Replace:
```go
	id := newID()
	rel := filepath.Join("books", id+".epub")
```
with:
```go
	id := newID()
	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext == "" {
		ext = ".epub"
	}
	rel := filepath.Join("books", id+ext)
```

- [ ] **Step 4: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/store/...`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/store/store.go internal/store/store_import_test.go
git commit -m "feat(store): Import preserves source file extension"
```

---

## Task 4: TXT 后端（编码识别 + 分章 + 兜底）

**Files:**
- Create: `internal/book/txt/txt.go`
- Create: `internal/book/txt/txt_test.go`
- Modify: `internal/book/formats/formats.go`

- [ ] **Step 1: 写失败测试**

Create `internal/book/txt/txt_test.go`:
```go
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
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/book/txt/`
Expected: FAIL（`Parse` undefined）

- [ ] **Step 3: 实现**

Create `internal/book/txt/txt.go`:
```go
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
```

- [ ] **Step 4: 提升 x/text 为直接依赖并运行测试**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go mod tidy
go test ./internal/book/txt/
```
Expected: PASS（`go mod tidy` 把 `golang.org/x/text` 从 indirect 提升为直接依赖）。

- [ ] **Step 5: 把 txt 接入组合根**

In `internal/book/formats/formats.go`, replace:
```go
import (
	_ "moyureader/internal/book/epub"
)
```
with:
```go
import (
	_ "moyureader/internal/book/epub"
	_ "moyureader/internal/book/txt"
)
```

- [ ] **Step 6: 全量测试**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./...`
Expected: PASS

- [ ] **Step 7: 提交**

```bash
git add internal/book/txt/ internal/book/formats/formats.go go.mod go.sum
git commit -m "feat(txt): TXT backend with UTF-8/GB18030 detection and chapter splitting"
```

---

## Task 5: 三个伪装主题（docker/npm/pytest）

**Files:**
- Create: `internal/disguise/theme_extra.go`
- Create: `internal/disguise/theme_extra_test.go`
- Modify: `internal/disguise/theme.go`

- [ ] **Step 1: 写失败测试**

Create `internal/disguise/theme_extra_test.go`:
```go
package disguise

import (
	"strings"
	"testing"
)

func TestNewThemesRegistered(t *testing.T) {
	for _, name := range []string{"docker", "npm", "pytest"} {
		if got := ThemeByName(name).Name(); got != name {
			t.Fatalf("ThemeByName(%q).Name() = %q", name, got)
		}
	}
}

func TestStyleCycleIncludesNewThemes(t *testing.T) {
	seen := map[string]bool{}
	s := "log"
	for i := 0; i < 6; i++ {
		seen[s] = true
		s = NextStyle(s)
	}
	for _, n := range []string{"log", "build", "git", "docker", "npm", "pytest"} {
		if !seen[n] {
			t.Fatalf("cycle should include %q; seen=%v", n, seen)
		}
	}
	if NextStyle("pytest") != "log" {
		t.Fatalf("cycle should wrap pytest->log, got %q", NextStyle("pytest"))
	}
}

func TestNewThemesHeaderFooter(t *testing.T) {
	cases := map[string]string{
		"docker": "docker compose up",
		"npm":    "npm install",
		"pytest": "pytest -v",
	}
	for name, headerMark := range cases {
		th := ThemeByName(name)
		if !strings.Contains(th.Header(60, "st"), headerMark) {
			t.Fatalf("%s header missing %q: %q", name, headerMark, th.Header(60, "st"))
		}
		f := th.Footer(60, "STATUSMARK")
		if !strings.Contains(f, "? help") || !strings.Contains(f, "STATUSMARK") {
			t.Fatalf("%s footer should embed status + help: %q", name, f)
		}
		if th.LinePrefix(3) != th.LinePrefix(3) {
			t.Fatalf("%s LinePrefix not deterministic", name)
		}
		if strings.Contains(th.BossLine(2), "STATUSMARK") {
			t.Fatalf("%s BossLine must not contain status/novel", name)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/ -run "NewThemes|StyleCycle"`
Expected: FAIL（三主题未注册；循环不含它们）

- [ ] **Step 3: 实现主题**

Create `internal/disguise/theme_extra.go`:
```go
package disguise

import "fmt"

// --- docker theme ---

type dockerTheme struct{}

func (dockerTheme) Name() string { return "docker" }

var dockerSvcs = []string{"web", "api", "worker", "redis", "db"}

func (dockerTheme) LinePrefix(seed int) string {
	return fmt.Sprintf("moyu-%s-1| ", dockerSvcs[seed%len(dockerSvcs)])
}
func (dockerTheme) Header(width int, status string) string {
	return padBetween("docker compose up", "● running", width)
}
func (dockerTheme) Footer(width int, status string) string {
	return padBetween("[+] Running 5/5 · "+status, "? help", width)
}
func (dockerTheme) BossLine(seed int) string {
	return fitLine(dockerTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- npm theme ---

type npmTheme struct{}

func (npmTheme) Name() string { return "npm" }

var npmPrefixes = []string{"npm WARN deprecated ", "npm http fetch GET 200 ", "npm timing build:run ", "npm info run "}

func (npmTheme) LinePrefix(seed int) string { return npmPrefixes[seed%len(npmPrefixes)] }
func (npmTheme) Header(width int, status string) string {
	return padBetween("npm install", "⠹", width)
}
func (npmTheme) Footer(width int, status string) string {
	return padBetween("added 1287 packages in 14s · "+status, "? help", width)
}
func (npmTheme) BossLine(seed int) string {
	return fitLine(npmTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}

// --- pytest theme ---

type pytestTheme struct{}

func (pytestTheme) Name() string { return "pytest" }

var pytestMods = []string{"core", "api", "auth", "models", "utils", "cache"}

func (pytestTheme) LinePrefix(seed int) string {
	return fmt.Sprintf("tests/test_%s.py::test_%d ", pytestMods[seed%len(pytestMods)], seed%97)
}
func (pytestTheme) Header(width int, status string) string {
	return padBetween("pytest -v", "● running", width)
}
func (pytestTheme) Footer(width int, status string) string {
	return padBetween("== 142 passed in 3.21s == · "+status, "? help", width)
}
func (pytestTheme) BossLine(seed int) string {
	return fitLine(pytestTheme{}.LinePrefix(seed)+bossPayload[seed%len(bossPayload)], 0)
}
```

- [ ] **Step 4: 注册主题 + 纳入循环**

In `internal/disguise/theme.go`, replace:
```go
// styleOrder defines the Tab cycle order.
var styleOrder = []string{"log", "build", "git"}

var registry = map[string]Theme{
	"log":   logTheme{},
	"build": buildTheme{},
	"git":   gitTheme{},
}
```
with:
```go
// styleOrder defines the Tab cycle order.
var styleOrder = []string{"log", "build", "git", "docker", "npm", "pytest"}

var registry = map[string]Theme{
	"log":    logTheme{},
	"build":  buildTheme{},
	"git":    gitTheme{},
	"docker": dockerTheme{},
	"npm":    npmTheme{},
	"pytest": pytestTheme{},
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go test ./internal/disguise/...`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add internal/disguise/theme_extra.go internal/disguise/theme_extra_test.go internal/disguise/theme.go
git commit -m "feat(disguise): docker, npm, pytest disguise styles"
```

---

## Task 6: 端到端集成（.txt）+ 全量验证 + 跑给你看效果

**Files:**
- Create: `cmd/reader/txt_integration_test.go`
- 临时 `cmd/_frames/main.go`（验证后删除，不提交）

- [ ] **Step 1: 写 .txt 端到端测试**

Create `cmd/reader/txt_integration_test.go`:
```go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"moyureader/internal/book"
)

// Importing package formats via main.go's blank import means .txt is registered.
func TestTxtOpensEndToEnd(t *testing.T) {
	p := filepath.Join(t.TempDir(), "novel.txt")
	if err := os.WriteFile(p, []byte("第一章\n正文一段\n第二章\n正文二段\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := book.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Chapters) != 2 {
		t.Fatalf("want 2 chapters from .txt, got %d", len(b.Chapters))
	}
}
```

- [ ] **Step 2: 全量 vet + test**

Run:
```bash
export PATH="/d/develop/Go/bin:$PATH"
go vet ./...
go test ./...
```
Expected: 全部 PASS。

- [ ] **Step 3: 写临时帧渲染程序（看三种新主题 + txt）**

Create `cmd/_frames/main.go`:
```go
package main

import (
	"fmt"
	"os"
	"strings"

	"moyureader/internal/book"
	_ "moyureader/internal/book/formats"
	"moyureader/internal/store"
	"moyureader/internal/ui"
)

func main() {
	// 1) 临时造一个 txt 验证解析 + 三种新主题渲染
	tmp := "cmd/_frames/sample.txt"
	_ = os.WriteFile(tmp, []byte("第一章\n"+strings.Repeat("摸鱼摸鱼摸鱼摸鱼摸鱼\n", 30)+"第二章\n正文\n"), 0o644)
	b, err := book.Open(tmp)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	fmt.Printf("txt 解析: 书名=%q 章节数=%d 第一章标题=%q\n", b.Title, len(b.Chapters), b.Chapters[0].Title)

	for _, style := range []string{"docker", "npm", "pytest"} {
		r := ui.NewReaderView(b, store.Progress{Chapter: 0, Line: 0}, store.Prefs{Style: style, Mode: "shell"}, 64, 12)
		fmt.Println("\n===== " + style + " (shell) =====")
		fmt.Println(strings.Join(r.Render(), "\n"))
	}
	_ = os.Remove(tmp)
}
```

- [ ] **Step 4: 运行帧程序，肉眼检查效果**

Run: `export PATH="/d/develop/Go/bin:$PATH" && go run ./cmd/_frames/`
Expected 检查点：
- `txt 解析:` 行显示书名=`sample`、章节数=`2`、第一章标题=`第一章`。
- docker：顶栏 `docker compose up … ● running`，正文行前缀 `moyu-web-1| ` 等。
- npm：顶栏 `npm install … ⠹`，行前缀 `npm WARN deprecated ` 等。
- pytest：顶栏 `pytest -v … ● running`，行前缀 `tests/test_core.py::test_… `，底栏 `== 142 passed …`。

- [ ] **Step 5: 删除临时程序**

Run: `rm -rf cmd/_frames`

- [ ] **Step 6: 提交集成测试**

```bash
git add cmd/reader/txt_integration_test.go
git commit -m "test(cmd): .txt opens end-to-end via format registry"
```

---

## Self-Review（计划作者已核对）

- **Spec 覆盖**：① 格式驱动注册架构 `internal/book`(Task1) + 后端迁移 + formats(Task2) ✓ ② TXT 编码识别/分章/护栏/兜底/元数据(Task4) ✓ ③ docker/npm/pytest 三主题 + 注册 + 循环(Task5) ✓ ④ `store.Import` 保留扩展名(Task3) ✓ ⑤ 调用方迁移(Task2) ✓ ⑥ 测试策略：book 路由 / txt 编码往返 / 主题 / 集成(Task1,4,5,6) ✓ ⑦ 依赖 `go mod tidy`(Task4) ✓。
- **类型一致性**：`book.{Book,Chapter,Parser,Register,Open}`、`epub.Parse(*book.Book)+init`、`txt.Parse(*book.Book)+init`、`formats` 空导入、`ui.NewReaderView(*book.Book)`/`ReaderView.book`、`ui.NewTOCView(*book.Book)`、`Model.book *book.Book`、`stream.NewStreamer(*book.Book)`、`disguise.{dockerTheme,npmTheme,pytestTheme}` + `styleOrder`/`registry` 跨任务一致。局部变量与包名冲突处统一改名 `book→bk`/`book→b`。
- **占位符扫描**：无 TBD/TODO；每个代码步骤为完整代码或精确 old→new 替换。
- **构建顺序**：Task1 独立绿；Task2 原子重构后全绿；Task4 在 formats 接入 txt 前后均可编译（formats 先只引 epub，Task4 再加 txt）；其余各任务独立绿。
- **已知取舍**：`epub_test.go` 迁移后无需改动（仅用局部变量访问字段）；`go mod tidy` 仅提升既有 indirect `x/text` 为直接依赖，无新增下载；帧程序仅本地肉眼验证、不入库。
```
