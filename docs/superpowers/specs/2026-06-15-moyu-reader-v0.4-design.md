# 摸鱼阅读器 · v0.4 增强设计文档

- 日期：2026-06-15
- 状态：已通过 brainstorming，待用户复核
- 基线：v0.3（已合并并发布 Release v0.3.0）
- 主题：**内容覆盖** —— TXT 格式支持 + 三个新伪装风格

## 概述

两块互相独立的增强：

1. **TXT 格式支持**：导入 `.txt`，自动识别 UTF-8 / GBK(GB18030) 编码，行锚定分章，无标题时整本作单章。
2. **三个新伪装风格**：`docker` / `npm` / `pytest`，`Tab` 循环纳入。

为支撑 TXT 与后续多格式，引入 Go 标准库式的「格式驱动注册」架构：中立入口包 `internal/book` + 各格式后端子包 + 自注册。

不改动：渲染分页、持久化 JSON 字段（除 `BookEntry.File` 现在保留源扩展名外）、阅读/内联/老板屏形态、TOC/帮助/滚动条等 v0.3 既有能力。

---

## 1. 架构：格式驱动注册（`internal/book`）

仿 stdlib `image`（`image.RegisterFormat` + `image/png` 自注册）与 `database/sql`（driver 注册）的可插拔模式。

### 包结构

```
internal/book/            中立入口：类型 + 注册表 + Open
  book.go                 Book, Chapter, Parser, Register, Open
internal/book/epub/       EPUB 后端（由现 internal/epub 迁入）
internal/book/txt/        TXT 后端（新建）
internal/book/formats/    组合根：空导入各后端触发自注册
```

### 依赖方向（无环）

- 后端 `epub`/`txt` 依赖 `book`（拿类型 + 注册自身）。
- `book` 不依赖任何后端。
- `ui` / `stream` / `cmd` 只依赖 `book`（认 `book.Open` 一个入口）。
- 新增格式 = 加一个后端子包 + 在 `formats` 加一行空导入，其它代码零改动。

### `internal/book/book.go`（核心契约）

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
// this from init(). Last registration for an extension wins.
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

### 后端自注册（epub 示例）

```go
package epub

import "moyureader/internal/book"

func init() { book.Register(".epub", Parse) }

// Parse 现有逻辑不变，仅把返回类型从 *Book 改为 *book.Book，
// Chapter 构造改为 book.Chapter。
func Parse(path string) (*book.Book, error) { /* ... */ }
```

### 组合根 `internal/book/formats`

```go
// Package formats wires all built-in book format backends. Import it (usually
// for side effects) to make every supported format available via book.Open.
package formats

import (
	_ "moyureader/internal/book/epub"
	_ "moyureader/internal/book/txt"
)
```

`cmd/reader` 导入 `formats`（空导入）触发注册；集成测试同样导入它即可获得全格式。

---

## 2. TXT 解析（`internal/book/txt`）

`Parse(path string) (*book.Book, error)` 流程：

### 2.1 读取 + 编码识别（自动 UTF-8 / GB18030）

1. 读全部字节。
2. 有 UTF-8 BOM（`EF BB BF`）→ 去 BOM，按 UTF-8 处理。
3. 否则 `utf8.Valid(data)` 为真 → 直接当 UTF-8。
4. 否则 → 用 `golang.org/x/text/encoding/simplifiedchinese.GB18030.NewDecoder()` 转成 UTF-8（GB18030 是 GBK 超集，覆盖更全）。

### 2.2 规整

- 换行统一：`\r\n`、`\r` → `\n`。
- 按 `\n` 切行；逐行 `strings.TrimRight(line, " \t　\r")` 去尾随空白（含全角空格）。

### 2.3 分章（行锚定 + 短行护栏）

一行被认作章节标题，需**同时**满足：

- 去首尾空白后匹配标题正则：
  ```
  ^第\s*[0-9零一二三四五六七八九十百千万两]+\s*[章回卷节话篇集部]|^(序章|序言|楔子|引子|前言|后记|尾声|终章|番外|附录)
  ```
- 该行 `render.StringWidth` ≤ 30（避免「他翻到第三章时……」这类正文句被误判）。

匹配到标题行 → 开新章，`Title` 取该行去空白后的文本。其余非空行作为当前章段落（**一行一段**，符合中文 txt 习惯）；空行跳过。

### 2.4 兜底

整篇无任何匹配标题 → 整本作单章。

### 2.5 元数据

- `Title` = 文件名去扩展名（`filepath.Base` 去 `.txt`）。
- `Author` = 空字符串（txt 无内置信息）。
- 若第一个标题行之前存在正文，归入一个 `Title` = 书名的「前置章」（保证开头正文不丢）。

### 2.6 边界

- 空文件 / 全空行 → 返回单章、段落为空（调用方 `ReaderView` 已能处理空章不 panic，见 v0.3 测试）。
- 解析过程不因单行异常失败；编码解码出错才返回 error。

---

## 3. 三个新伪装主题（`internal/disguise`）

各实现 `Theme` 接口（`Name / LinePrefix / Header / Footer / BossLine`），加入 `registry` 与 `styleOrder`。循环顺序变为 `log→build→git→docker→npm→pytest→log`。底栏统一右侧 `? help` 且嵌入 `status`，与现有主题一致（经 `padBetween`）。

### docker（容器日志流）

- `Name() = "docker"`
- 服务名集合：`["web","api","worker","redis","db"]`
- `LinePrefix(seed)`：`fmt.Sprintf("moyu-%s-1| ", svc[seed%len])`
- `Header`：`padBetween("docker compose up", "● running", width)`
- `Footer`：`padBetween("[+] Running 5/5 · "+status, "? help", width)`
- `BossLine(seed)`：`fitLine(LinePrefix(seed)+bossPayload[seed%len], 0)`

### npm（装包刷屏）

- `Name() = "npm"`
- 前缀集合：`["npm WARN deprecated ","npm http fetch GET 200 ","npm timing build:run ","npm info run "]`
- `LinePrefix(seed)`：`prefixes[seed%len]`
- `Header`：`padBetween("npm install", "⠹", width)`
- `Footer`：`padBetween("added 1287 packages in 14s · "+status, "? help", width)`
- `BossLine(seed)`：同上套路。

### pytest（测试输出）

- `Name() = "pytest"`
- 模块集合：`["core","api","auth","models","utils","cache"]`
- `LinePrefix(seed)`：`fmt.Sprintf("tests/test_%s.py::test_%d ", mods[seed%len], seed%97)`
- `Header`：`padBetween("pytest -v", "● running", width)`
- `Footer`：`padBetween("== 142 passed in 3.21s == · "+status, "? help", width)`
- `BossLine(seed)`：同上套路。

确切字符串以上面为准；`BossLine` 不含任何小说正文（沿用现有约定）。

---

## 4. 附带改动：`store.Import` 保留扩展名

现状：`Import` 把源文件拷成 `books/<id>.epub`（固定扩展名）。

改为：保留**源文件扩展名**，`BookEntry.File` 变为 `books/<id><ext>`（`ext` 为源文件小写扩展名）。这样重开时 `book.Open` 才能按扩展名分发到正确后端。已有 epub 条目的 `File` 字段不受影响（仍是历史写入值）。

---

## 5. 调用方迁移

机械替换，全程靠测试兜底：

- `internal/ui/reader.go`、`toc.go`、`model.go`：`epub.Book→book.Book`、`epub.Chapter→book.Chapter`、`epub.Parse→book.Open`，import 改为 `internal/book`。
- `internal/stream`：同上。
- `cmd/reader`：`epub.Parse→book.Open`，并空导入 `internal/book/formats` 触发注册。
- `internal/epub` 目录移动到 `internal/book/epub`（import path 变化；因类型已上移，外部不再直接引用 epub 包，仅 `formats` 空导入它）。

---

## 6. 测试策略

- **book**：注册一个假 parser，验证 `Open` 按扩展名路由；扩展名大小写不敏感（`.TXT` 命中 `.txt`）；未知格式返回含「supported」的清晰错误。
- **txt**：
  - UTF-8（带 BOM / 不带 BOM）解析出正确章节。
  - **GB18030 往返**：测试内用 `simplifiedchinese.GB18030.NewEncoder()` 把中文编成字节，再 `Parse` 解回，断言中文正确。
  - 分章：`第一章…\n第二章…` → 2 章；标题取自标题行。
  - 短行护栏：含「第三章」的长正文句**不**判为新章。
  - 兜底：无标题 → 单章，`Title` = 文件名。
  - CRLF 规整：`\r\n` 输入正常分行。
- **epub**：原有测试随包迁移，适配 `*book.Book`，行为不变。
- **disguise**：三新主题各测 `ThemeByName` 命中、`Header/Footer` 含特征文本 + `? help` + 内嵌 status、`LinePrefix` 同 seed 确定性、`NextStyle` 循环包含三新主题且能回到 `log`、`BossLine` 不含正文标记。
- **集成（cmd/reader）**：导入 `formats`；`.epub` 仍可开；新增一个 `.txt`（含 `第一章`）用例验证端到端导入 + 解析。

---

## 7. 依赖

仅新增官方扩展库 `golang.org/x/text`（用于 GB18030 解码）。`go.mod`/`go.sum` 更新；CI（vet+test+build）与 Release（交叉编译）无需特殊处理。

---

## 8. 明确不做（YAGNI）

- 不做 MOBI / AZW3 / PDF / Markdown（后续版本）。
- 不做内容嗅探（magic number）；仅按扩展名分发。
- 不做手动指定编码（只自动识别 UTF-8 / GB18030）。
- 不做按长度智能切章（无标题只单章兜底）。
- 主题不做动画 spinner、不引入新配色层（沿用现有 paint 层）。
