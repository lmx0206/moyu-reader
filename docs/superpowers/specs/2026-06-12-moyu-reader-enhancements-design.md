# 摸鱼阅读器 · 增强设计文档（v0.2）

- 日期：2026-06-12
- 状态：已通过 brainstorming，待用户复核 spec
- 基线：v0.1（核心引擎 + 前端已合并到 master）

## 概述

在已上线的 v0.1 上做四项增强：
1. **阅读界面重设计**（外壳/A 模式）→ "日志查看器"风格（顶部标题栏 + 分隔线 + 留白正文 + 分隔线 + 底部状态栏）
2. **目录跳转**（`g` 键）→ 弹出"代码大纲"样式、显示真实章节名的章节列表，回车跳章
3. **帮助键**（`?` 键）→ 弹出伪装成 `reader --help` 的快捷键说明
4. **GitHub Actions** → CI（push/PR 跑 test）+ Release（打 tag 自动出 exe）

不改动：内联（B 模式）渲染、老板键假屏、EPUB 解析、持久化格式。

---

## 1. 阅读界面重设计（日志查看器风格）

### 现状
`disguise.RenderShell(th, body, width, status)` 只输出三段：`Header` + body + `Footer`，共 2 行装饰。

### 目标布局（5 段）
```
 app.log · tail -f · 14:23:01                    ● running   ← 顶部标题栏(1行)
 ─────────────────────────────────────────────────────────   ← 分隔线(1行)
                                                              ← 正文区(左缩进3格,
    林尘缓缓睁开眼睛，发现自己躺在一个陌生的房间里，             段间保留空行)
    四周一片漆黑，只有窗外透进一缕清冷的月光。
 ─────────────────────────────────────────────────────────   ← 分隔线(1行)
 INFO  142 passed · ch.3/57 · 21%                             ← 底部状态栏(1行)
```

### 改动
- **`internal/disguise` 包**
  - `RenderShell` 重写为 5 段：`topBar` + `separator` + `body(左缩进)` + `separator` + `bottomBar`。装饰行数从 2 增加到 **4**。
  - `Theme` 接口语义微调：`Header(width, status)` 产出**顶部标题栏**（含右对齐的 `● running` 指示），`Footer(width, status)` 产出**底部状态栏**。三种风格各自文案：
    - log：`app.log · tail -f · <ts>` ／ `INFO  142 passed · <status>`
    - build：`> gradle build · <ts>` ／ `BUILD SUCCESSFUL in 12s · <status>`
    - git：`git log -p · <ts>` ／ `3 files changed · <status>`
  - 新增辅助函数 `padBetween(left, right, width) string`：用 runewidth 计算，把 left 左对齐、right 右对齐、中间空格填充到 width 宽（宽度不足时优先保 left 并截断）。供顶部栏右对齐 `● running`、底部栏右对齐百分比用。
  - 分隔线：通用 `separatorLine(width)` 返回 `─` × width。
  - 正文左缩进常量 `bodyIndent = 3`（3 个空格）。`RenderShell` 给每行 body 前加缩进。
- **`internal/ui/reader.go`**
  - `bodyHeight()`：shell 模式由 `height-2` 改为 `height-4`（多两条分隔线）。inline 模式不变（`height`）。
  - `contentWidth()`：shell 模式由 `width` 改为 `width - bodyIndent - rightMargin`（`rightMargin = 1`），让正文两侧留白；inline 模式仍用 `width`。
  - `Render()` 调 `RenderShell` 的方式不变（仍传 body+width+status），缩进与分段在 disguise 层完成。
- **影响测试**：`disguise` 的 shell 渲染测试与 `ui` 的 `TestReaderRenderExactHeightShellMode` 需按"装饰 4 行 / 正文 height-4 行"更新断言（总行数仍 == height）。

---

## 2. 目录跳转（`g` 键，代码大纲样式）

### 组件
- 新增 `internal/ui/toc.go` 的 **`TOCView`**（纯逻辑、可测）：
  - 字段：`chapters []string`（章节标题）、`cursor int`、`top int`（滚动视窗顶部索引）。
  - `NewTOCView(book *epub.Book, current int) *TOCView`：光标定位到当前章。
  - `MoveUp() / MoveDown()`：移动光标并维护 `top` 使光标始终可见。
  - `Selected() int`：返回选中章节索引。
  - `Render(width, height int) []string`：渲染为"代码大纲"样式，显示真实章节名，高亮当前光标行。例：
    ```
    outline · book.go                        57 symbols
       ▸ func ch01()   第一章 觉醒
     ▸ ▸ func ch02()   第二章 迷雾      ← 光标行高亮
       ▸ func ch03()   第三章 抉择
    ```
    （章节多于可视高度时按 `top` 窗口滚动。）

### 接入 Model
- 新增屏幕状态 `screenTOC`。
- 阅读界面按 `g` → `m.toc = NewTOCView(book, 当前章)`，`m.screen = screenTOC`。
- TOC 界面按键：`↑/k` `↓/j` 移动；`Enter` → 跳到 `Selected()` 章、`line=0`、回 `screenReader` 并存盘；`Esc/q` → 取消回 `screenReader`。
- Model 需持有当前打开的 `*epub.Book`（现在 `openBook` 解析后只存进 `ReaderView`，需要在 Model 上也留一份引用 `book`，供 TOC 用），并提供 `reader.JumpTo(chapter)` 方法重置位置。

---

## 3. 帮助键（`?`，伪装成 --help）

### 组件
- 新增屏幕状态 `screenHelp`。
- 在书架 / 阅读 / TOC 界面按 `?` → `m.screen = screenHelp`（记住来源 `helpReturn screen` 以便返回）。
- 任意键 / `Esc` → 回到来源界面。
- `View()` 渲染**静态帮助文本**，伪装成 `reader --help` 用法输出，内容覆盖全部快捷键：
  ```
  reader - a tail-style log viewer (v0.2)

  USAGE:
    reader [command]

  KEYBINDINGS (shelf):
    ↑/↓  select    enter  open    i  import    d  delete    q  quit
  KEYBINDINGS (reader):
    space/→/pgdn  next page     ↑/↓  scroll line
    tab  switch profile         m    toggle view
    g    goto section           `/b  minimize
    ?    help                   esc  back to list
  KEYBINDINGS (stream/CLI):
    enter  next   b  back   t  switch profile   q  quit
  ```
  （措辞刻意像真工具：profile=风格、minimize=老板键、section=章节、log viewer=阅读器。）
- **发现性**：底部状态栏 / 书架标题角落加不起眼的 `· ? help` 提示。

---

## 4. GitHub Actions

### `.github/workflows/ci.yml`
- 触发：`push`（所有分支）与 `pull_request`。
- 步骤：checkout → `actions/setup-go`（固定 Go 版本，如 `1.26.x`）→ `go vet ./...` → `go test ./...` → `go build ./...`。
- 作用：PR/提交自动验证，坏代码显示红叉。

### `.github/workflows/release.yml`
- 触发：`push` tag 匹配 `v*`（如 `v0.2.0`）。
- 步骤：checkout → setup-go → `go build -ldflags "-s -w" -o reader.exe ./cmd/reader` → 用 `softprops/action-gh-release`（或 `gh release create`）把 `reader.exe` 作为 Release 附件上传。
- 作用：打 tag 即得可下载的单文件 exe。

---

## 测试策略

- **disguise**：`RenderShell` 新 5 段布局（总行数、顶/底栏内容、正文缩进）、`padBetween`（左右对齐与截断）、`separatorLine` 的单元测试。
- **ui/reader**：更新 `TestReaderRenderExactHeightShellMode`（装饰 4 行），新增 `JumpTo` 测试。
- **ui/toc**：`NewTOCView` 定位当前章、`MoveUp/MoveDown` 边界与滚动窗口、`Selected`、`Render` 高亮与滚动的单元测试。
- **ui/model**：`g` 打开 TOC、TOC 内 `Enter` 跳章后回到 reader 且章节正确、`?` 打开帮助并任意键返回的 Update 测试。
- **GitHub Actions**：本地无法单测；首次 push 后在仓库 Actions 页确认绿勾。

## 明确不做（YAGNI）

- 不做书签、搜索、主题配色切换（本轮聚焦上述四项）
- 不重设计内联 B 模式与老板屏
- 不改持久化 JSON 结构（无需迁移）
