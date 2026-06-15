# 摸鱼阅读器 · v0.5 增强设计文档

- 日期：2026-06-15
- 状态：已通过 brainstorming，待用户复核
- 基线：v0.4（已合并并发布 Release v0.4.0）
- 主题：**标注（书签 + 笔记）** + **修复 resize 内容显示不全 bug**（同根因，一并解决）

## 概述

把「书签」和「笔记」统一为一个**标注（Annotation）**概念：每条标注 = 一个位置 + 可选批注文字（留空即纯书签）。标注与阅读进度都改用**排版无关的段落锚点**（章 + 段落序号），从根本上解决「换窗口宽度后位置漂移 / 内容显示不全」的问题。

核心改动：
1. **段落锚点基础设施**：`render` 新增「行↔段落」映射纯函数。
2. **修复 resize bug**：`ReaderView.SetSize` 按段落重锚顶部位置（会话内改窗口大小后停在原处、内容完整）。
3. **进度持久化改段落锚点**：`store.Progress` 由 `{Chapter, Line}` 改为 `{Chapter, Para}`（跨终端宽度重开也不漂）。
4. **标注**：`store` 增数据结构与增删；`ui` 新增「加标注」输入与「标注列表」覆盖层（伪装成调试器断点面板）。

不改动：EPUB/TXT 解析、伪装主题、翻页/滚动、TOC、帮助、老板键等既有能力。

---

## 1. 数据模型与持久化（`internal/store`）

### Annotation
```go
// Annotation marks a reading position with an optional note. An empty Note is a
// plain bookmark; a non-empty Note is an annotation.
type Annotation struct {
	Chapter   int    `json:"chapter"`
	Para      int    `json:"para"`           // paragraph index within the chapter (layout-independent)
	Note      string `json:"note,omitempty"` // empty = plain bookmark
	CreatedAt string `json:"createdAt"`
}
```

`BookEntry` 增加字段：
```go
	Annotations []Annotation `json:"annotations,omitempty"`
```

### Progress 改段落锚点（破坏性、可接受）
```go
// Progress is a stable reading position: chapter index + paragraph index.
type Progress struct {
	Chapter int `json:"chapter"`
	Para    int `json:"para"`
}
```
- 由 `{Chapter, Line}` 改为 `{Chapter, Para}`。旧存档的 `"line"` 键不再被读取，`Para` 默认 0 → 已在读的书一次性回到**所在章开头**。本地单文件、影响很小，可接受。
- 全仓库对 `Progress.Line` 的引用都要改（`ui`、`stream`、`cmd`、相关测试）。

### store 纯函数（调用方负责 Save）
```go
// AddAnnotation appends an annotation to the book and returns it.
func AddAnnotation(lib *Library, id string, a Annotation)
// DeleteAnnotation removes the annotation at index i (no-op if out of range).
func DeleteAnnotation(lib *Library, id string, i int)
```
标注按**创建顺序**存储与展示（不排序），`DeleteAnnotation` 按列表下标删除，避免下标错位。

---

## 2. render 行↔段落映射（`internal/render`）

新增文件 `internal/render/anchor.go`，三个纯函数，排版规则与 `LayoutChapter` 完全一致（段落 `i>0` 前有一个空行）：

```go
// ParagraphStartLines returns, for each paragraph, the display-line index where
// its content begins at the given width (matching LayoutChapter's layout).
func ParagraphStartLines(paragraphs []string, width int) []int

// ParaStartLine returns the start display-line of paragraph para (clamped).
func ParaStartLine(paragraphs []string, width, para int) int

// LineToPara returns the index of the paragraph that owns display-line line
// (the last paragraph whose start <= line; clamped to [0, len-1]).
func LineToPara(paragraphs []string, width, line int) int
```

实现要点：
- `ParagraphStartLines`：累加 `line`，`i>0` 时先 `line++`（段间空行），记 `starts[i]=line`，再 `line += len(WrapParagraph(p, width))`。
- `ParaStartLine`：`para` 夹到 `[0, len-1]`，空段落集返回 0。
- `LineToPara`：遍历 `starts` 取最后一个 `<= line` 的下标；空段落集返回 0。

---

## 3. ReaderView 段落锚定（`internal/ui/reader.go`）

`line` 仍是渲染用的顶部行号，但**段落是规范锚点**。

- `NewReaderView`：`r.line = render.ParaStartLine(章节段落, r.contentWidth(), p.Para)`，再 `clampLine`。
- `SetSize(w, h)`：
  ```go
  paras := r.book.Chapters[r.chapter].Paragraphs // 经 chapter 边界检查
  topPara := render.LineToPara(paras, r.contentWidth(), r.line) // 旧宽度
  r.width, r.height = w, h
  r.line = clampLine(render.ParaStartLine(paras, r.contentWidth(), topPara), r.chapterLineCount())
  ```
  → resize 后顶部仍是同一段落，内容完整（**修复 resize bug**）。
- `Progress()`：
  ```go
  return store.Progress{Chapter: r.chapter, Para: render.LineToPara(当前章段落, r.contentWidth(), r.line)}
  ```
- 新增 `JumpToPara(chapter, para int)`：跳到指定章并把 `line` 设为该段起始行（章、段都夹紧）。供书签跳转。
- 现有 `JumpTo(chapter)`（TOC 用）保持：跳到章开头（`para=0` 等价），可改为内部调用 `JumpToPara(chapter, 0)`。

> 注意：`SetSize`/`Progress`/构造里取段落要先判 `chapter` 合法（空书或越界返回安全值），复用现有 `clamp`/边界习惯，不 panic。

---

## 4. UI 交互（`internal/ui`）

### 4.1 加标注：`a`
- 阅读界面按 `a` → 进入 `screenAnnotate`：复用 import 式单行输入（提示「批注（可留空=书签），回车保存，Esc 取消」）。
- 回车：以当前**顶部段落**为锚点（`m.reader.Progress()` 的 Chapter/Para）构造 `Annotation{Chapter, Para, Note: 输入文字, CreatedAt: now}`，`store.AddAnnotation` + `Save`，回阅读界面并提示「已加标注」。
- `Esc`：取消，回阅读界面。

### 4.2 标注列表：`l`
- 阅读界面按 `l` → 进入 `screenAnnotList`，新建 `AnnotationView`（持有当前书 + 该书 `Annotations` 副本）。
- 渲染伪装成调试器断点面板：
  - 顶栏：`breakpoints (N)`
  - 每行：`● brk <i+1>  ch<chapter+1>:¶<para+1>  <摘要>`，摘要 = 该段开头文字截断（有 Note 用 Note，否则用段落首句），CJK 宽度安全截断（复用 `render.StringWidth`/`PadRight`）。
  - 空列表显示一行 `breakpoints (0)  — none set —`。
- 按键：`↑↓`/`kj` 移动；`enter` → `m.reader.JumpToPara(chapter, para)` + 保存进度 + 回阅读；`d` 删除选中（`store.DeleteAnnotation` + `Save` + 重建视图，光标夹紧）；`esc`/`q` 返回阅读。
- 用 `paintDim` 着色（与 TOC 一致）。

### 4.3 model 接线
- `screen` 枚举新增 `screenAnnotate`、`screenAnnotList`。
- `handleReaderKey` 加 `case "a"`（进 annotate）、`case "l"`（进 annot list，建视图）。
- 新增 `handleAnnotateKey`（输入：runes/backspace/space/enter/esc，仿 `handleImportKey`）与 `handleAnnotListKey`。
- `View()` 增两个 case：annotate 显示输入提示 + 缓冲；annotList 显示 `AnnotationView.Render(width,height)`。
- `helpText` 的 reader 段补：`a 加标注（书签/笔记）`、`l 标注列表`。

---

## 5. 调用方迁移（`Progress.Line` → `Progress.Para`）

- `internal/stream/stream.go`：`Streamer` 用 `Progress.Para`；`NewStreamer` 按 para 设起始（stream 用固定 `wrapWidth=80` 计算 `ParaStartLine`）；`Progress()` 返回 para（用 `LineToPara`，width=80）。
- `cmd/reader/main.go`：`runStream` 的 `UpdateProgress` 仍传 `s.Progress()`，类型已是新 Progress，无需特殊处理。
- 所有构造 `store.Progress{...}` 的测试改用 `Para`。

---

## 6. 测试策略

- **render**：`ParagraphStartLines` 对 1/多段落、空段落集；`ParaStartLine` 越界夹紧；`LineToPara` 段中/段间空行/越界；**往返**：`LineToPara(ParaStartLine(p))==p`。
- **store**：`AddAnnotation` 追加、`DeleteAnnotation` 删除与越界 no-op；Progress JSON 用 `para` 键。
- **ui/reader**：
  - resize 守恒：构造后 PageDown 几页 → `SetSize` 改宽 → 顶部段落（`Progress().Para`）不变，且 `Render()` 行数仍 == height、首个正文行非空（内容完整、不再是末尾空白页）。
  - `JumpToPara` 跳转正确；`Progress()` 段落往返。
- **ui/model**：`a` 后该书 `Annotations` +1；`l` 打开列表；列表 `enter` 跳到对应段落；`d` 删除后 -1。
- **stream**：Progress 段落往返不 panic。

---

## 7. 明确不做（YAGNI）

- 不做划选区间 / 高亮某段（无选区交互，成本高，留后续）。
- 不做标注导出（markdown）、跨设备同步、标注排序/搜索（留后续）。
- 不做老存档 `line→para` 的智能迁移（一次性回到章首，可接受）。
- 标注列表不做分页滚动以外的花样（复用 TOC 式滚动窗口即可）。
