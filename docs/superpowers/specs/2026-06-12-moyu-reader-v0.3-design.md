# 摸鱼阅读器 · v0.3 增强设计文档

- 日期：2026-06-12
- 状态：已通过 brainstorming，待用户复核 spec
- 基线：v0.2（已合并并发布 Release v0.2.0）

## 概述

四项改动：两个修复 + 两个阅读增强。
1. **修复 log 时间**：时间戳基于真实当前时间，而非从 08:00:00 按行号硬编码递增。
2. **修复老板键退出**：假屏下只有老板键本身（`` ` `` / `b`）能退出，其他键无效。
3. **翻页 / 滚动双模式**：按 `s` 切换；滚动模式右侧带竖向滚动条。
4. **状态栏显示本章页码**：`ch.3/57 · 本章 2/7页 · 21%`。

不改动：EPUB 解析、持久化 JSON 结构、内联（B 模式）与老板屏的整体形态、TOC/帮助覆盖层。

---

## 1. 修复 log 时间

### 现状
`logTheme.LinePrefix(seed)` 用 `8+(seed/3600)%10` 计算小时，永远从 08 点开始，且纯随行号变化，显得假。

### 设计
- `internal/disguise/theme.go` 新增包级变量 `clockBase = time.Now()`（程序启动时刻）。
- 新增纯函数 `logClock(base time.Time, seed int) string`：返回 `base.Add(time.Duration(seed)*time.Second).Format("15:04:05")`。
- `logTheme.LinePrefix(seed)` 改用 `logClock(clockBase, seed)` 生成时间戳，其余（级别、类名）不变。
- 可测性：`logClock` 是纯函数，用固定 base 单测；`TestLogThemePrefixDeterministic`（同 seed 两次相等、含级别）仍成立。

---

## 2. 修复老板键退出

### 现状
`internal/ui/model.go` `handleKey` 开头：`if m.bossActive { m.bossActive = false; return }` —— 任意键退出。

### 设计
改为只有老板键能退出：
```go
if m.bossActive {
    if key == "`" || key == "b" {
        m.bossActive = false
    }
    return m, nil // 其他键一律吞掉，停留在假屏
}
```
（`key := msg.String()` 需移到该判断之前。）

---

## 3. 翻页 / 滚动双模式（含滚动条）

### 导航模式
- `ReaderView` 新增字段 `nav string`（`"page"` | `"scroll"`，默认 `"page"`）与方法 `ToggleNav()`。
- `internal/ui/model.go` 阅读界面新增按键 `s` → `m.reader.ToggleNav()`。
- 两套移动键在**两种模式下都始终可用**（`Space`/`PgDn` 整页翻、`↑↓` 逐行滚动），不互相禁用——`nav` 只决定**是否绘制右侧滚动条**：
  - **翻页模式**（默认）：不画滚动条，画面更干净。
  - **滚动模式**：正文右侧显示滚动条，配合 `↑↓` 逐行滚动时能直观看到在本章的位置。

### 滚动条与右侧合成
- `internal/render` 新增两个纯函数：
  - `PadRight(s string, width int) string`：用空格把 s 右填充到 width 显示列（超出则按 cell 截断），CJK 宽度感知。
  - `Scrollbar(total, top, viewport int) []string`：返回 `viewport` 个字形，表示在 total 行中、从 top 起、viewport 高的视窗位置；滑块 `█`、轨道 `░`。total<=viewport 时全为轨道（无需滚动）。
- **正文缩进职责上移**：`disguise.RenderShell` 不再给正文加左缩进，只负责顶栏/分隔线/正文原样/分隔线/底栏（仍是 4 行装饰）。左缩进改由 `ReaderView` 负责。
  - `internal/ui/reader.go` 新增常量 `BodyIndent = 3`（原在 disguise，迁移至此）。
  - `disguise.BodyIndent` 删除。

### ReaderView.Render() 新逻辑
1. 取本章 `lines`、`bh = bodyHeight()`、窗口 `[top, top+bh)`（`top = r.line`）。
2. 构造 `page`（窗口内行，补空行到 bh 行）。
3. 组装 body：
   - 翻页模式（shell）：每行 `indent + line`。
   - 滚动模式（shell）：先算 `bars := render.Scrollbar(len(lines), top, bh)`；每行 `render.PadRight(indent+line, width-1) + bars[rowIdx]`。
   - inline 模式：不缩进、不滚动条（保持现状），且 inline 模式 `s` 切换无视觉差异（仍逐行/整页由 nav 决定移动量，但不画滚动条）。
4. shell → `disguise.RenderShell(th, body, width, StatusText())`；inline → `disguise.RenderInline(th, page, width)`。
- `bodyHeight()`、`contentWidth()` 维持 v0.2（shell 减 4 行装饰、减 `BodyIndent+rightMargin`）。滚动条占用最右 1 列，已被 `rightMargin=1` 覆盖，无需额外缩减。

---

## 4. 状态栏显示本章页码

- `ReaderView.StatusText()` 改为：`fmt.Sprintf("ch.%d/%d · 本章 %d/%d页 · %d%%", chapter+1, totalCh, page, totalPages, pct)`
  - `totalPages = ceil(chapterLineCount / bodyHeight)`（至少 1）
  - `page = top/bodyHeight + 1`（夹到 `[1, totalPages]`）
  - `pct` 维持现有（章节进度）。
- 翻页、滚动模式都显示页码（滚动模式下页码随 top 实时变化，仍有意义）。

---

## 帮助文本更新

`helpText()` 的 reader 段补充 `s  toggle scroll/page`，并把 `` `/b `` 的说明改为"minimize (press again to restore)"。

---

## 测试策略

- **render**：`PadRight`（ASCII/CJK 补齐与截断到精确 cell 宽）、`Scrollbar`（total<=viewport 全轨道；滑块位置随 top 移动；输出长度==viewport）单测。
- **disguise**：`logClock(固定base, seed)` 格式与递增；更新 `TestRenderShellFiveSectionLayout` —— RenderShell 不再缩进，正文行应原样（断言 `out[2]=="正文一"`，分隔线仍在 out[1]/out[4]）。
- **ui/reader**：
  - 翻页模式 `StatusText` 含"本章 1/N页"。
  - 滚动模式 Render 每行末尾含滚动条字形（`█` 或 `░`），且总行数仍 == height。
  - `ToggleNav` 在 page/scroll 间切换。
  - shell 正文缩进（`TestReaderRenderExactHeightShellMode` 更新：正文行以 3 空格起，footer 含 `? help`）。
- **ui/model**：`s` 切换 nav；老板键激活后按任意普通键仍在假屏、按 `` ` `` 退出。

## 明确不做（YAGNI）

- 不做鼠标拖拽滚动条、不做平滑动画滚动（逐行即可）。
- 内联（B）模式不画滚动条（它本就是流式日志形态）。
- 不改持久化结构（nav 模式不入库，默认每次启动为翻页模式）。
