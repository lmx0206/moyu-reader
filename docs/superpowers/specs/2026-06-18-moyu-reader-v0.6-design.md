# moyu-reader v0.6 设计：假命令行（REPL 阅读模式）+ 摸鱼统计

日期：2026-06-18
状态：已确认，待写实现计划

## 目标

v0.6 增加两个**相互独立**的组件，合在一个版本、一个 spec、一个实现计划里（计划内部分两批任务，各自可独立提交/审查）：

- **A. 假命令行**：第三种 TUI 阅读模式——一个可输入的伪 shell / REPL，正文作为「命令输出」打印。
- **B. 摸鱼统计**：阅读数据采集 + 伪装成 coverage 报告的统计面板。

非目标（YAGNI，本版不做）：命令行模式下未知命令吐假输出、统计导出、跨设备同步、命令自动补全。

---

## A. 假命令行（REPL 阅读模式）

### 形态

`ReaderView.mode` 增加第三种取值 `"repl"`。`m` 键循环顺序：`shell → inline → repl → shell`。

进入 repl 后是全屏「终端」：
- 上方是**回显区（scrollback）**：命令回显行 + 正文输出行交错，**底部对齐**（新内容在下，像真终端往下滚；超出高度则顶部行被顶掉）。
- 最后一行是**提示符** `PS C:\proj> ` 加当前输入缓冲（末尾光标用一个可见字符或下划线表示）。

### 交互模型

与外壳/内联的「被动翻页」根本不同——repl 是行编辑 + 命令执行：

- 可打印字符（除反引号）→ 追加到输入缓冲
- `Enter` → 执行当前缓冲的命令，清空缓冲
- `Backspace` → 删末字符
- `↑` / `↓` → 翻命令历史（执行过的非空命令入历史）
- `` ` ``（反引号）→ 触发**老板键**（全模式保留；命令词汇里永不需要反引号，故安全占用为 panic 键）
- `Esc` → 保存进度、回书架
- `Ctrl+C` → 退出程序

每次 `next` 吐**一个段落**（段长就多吐几行），按内联日志风格（`disguise.RenderInline`）渲染成输出行；进度前进一段。`prev` 打印上一段、进度回退一段。进度按**段落锚点**（`store.Progress{Chapter, Para}`），与现有模型一致。

### 命令词汇

精简阅读命令 + dev 风格别名；未知命令 → `command not found`。

| 输入 | 作用 |
|------|------|
| `Enter`（空）/ `n` / `next` / `git log` | 下一段 |
| `p` / `prev` | 上一段 |
| `toc` / `ls` | 打印章节列表（1 基序号 + 标题，伪装成目录列表） |
| `cd <章号>` | 跳到第 N 章章首并打印章标题；无效序号 → `cd: no such chapter` |
| `status` / `git status` | 打印进度行（如 `on chapter 4/15 · 23%`，伪装） |
| `clear` / `cls` | 清空回显区 |
| `help` / `?` | 打印命令清单（伪装成 usage 帮助） |
| `q` / `exit` | 保存进度、回书架 |
| 其他非空输入 | 打印 `command not found` |

命令解析大小写不敏感；`cd` 后跟空格 + 数字。`cd`/`status`/`toc` 的 N、进度等数字对用户显示为 1 基。

### 实现落点

- 新建 `internal/ui/repl.go`：`ReplView`，持有
  - `book *book.Book`、当前 `chapter, para int`（段落锚点）
  - `scroll []string`（回显区已渲染行）、`input string`（缓冲）、`history []string` + 历史游标
  - `width, height, style string`
  - 方法：`NewReplView(b, p, prefs, w, h)`、`SetSize`、`Render() []string`（恰好 height 行）、`Key(msg) `（处理一次按键，内部 `run(cmd)` 分发命令）、`Progress() store.Progress`、`Prefs()`。
- `ReplView` 复用 `disguise.RenderInline` 渲染段落为输出行、复用 `render.ParaStartLine`/`LineToPara` 与段落推进逻辑。
- `Model`：当阅读模式为 repl 时，`View()` 委托 `ReplView.Render()`，按键委托 `handleReplKey`（除 `` ` `` 老板键、`Esc`、`Ctrl+C` 由 `Model` 统一拦截外，其余交给 `ReplView`）。`m` 切到/切出 repl 时，在 `ReaderView` 与 `ReplView` 间同步段落进度。

### 测试策略

- 命令解析：`next`/`n`/`git log`/空 → 推进一段；`prev` 回退；`toc`/`cd N`/非法 `cd`；`status`；`clear` 清空 scrollback；未知 → `command not found`。
- 推进到章尾跨章；`Render()` 恒为 height 行、提示符在最后一行。
- 命令历史上下翻。
- `Progress()` 段落锚点正确；`SetSize` 后不丢内容（复用段落锚点）。

---

## B. 摸鱼统计

### 采集

阅读时累计，持久化到 `data/library.json`：

- **时长**：维护「上次活动时间戳」`lastActivity`。每次阅读动作时 `delta = now - lastActivity`；若 `delta ≤ 300s`（5 分钟空闲上限，超过算离开不计）则累加到全局 `TotalSeconds` 与 `TodaySeconds`；随后更新 `lastActivity`。老板键、空闲不计。
- **字数（高水位、不重复计）**：每本书记最远到达位置（`FurthestChapter, FurthestPara`）。进度推进到**新的更远**位置时，把新跨过段落的字符数增量加到该书 `CharsRead`；回看/重读不重复计。
- **进度 / 读完**：每本书 coverage% = `CharsRead / 总字数`（总字数由已加载的 book 现算）；100% 即「读完」。
- **streak**：全局记 `LastReadDate`（YYYY-MM-DD，本地时区）+ `StreakDays`。当天首次阅读：若 `今天 == LastReadDate + 1` 则 `StreakDays++`，否则（含首次/断签）重置为 1；同时把 `TodaySeconds` 清零；更新 `LastReadDate = 今天`。

### 展示

新建 `internal/ui/stats.go`：`StatsView`，伪装成 coverage 报告。新增屏幕 `screenStats`，按 **`c`**（coverage）从书架或阅读界面打开，`Esc` / `q` 关闭。

```
Name              Stmts   Miss  Cover
-----------------------------------------
三体.epub          1203    240   80%
活着.txt            512      0  100%
-----------------------------------------
TOTAL             1715    240   86%

elapsed 12h34m · today 1h12m · 142k chars · streak 7d
```

- 每本书一行：`Name`=书文件名/标题、`Stmts`=总字数（以「千」或原值，渲染时定）、`Miss`=未读字数、`Cover`=进度%。
- `TOTAL` 行汇总。
- 末尾汇总块：累计时长 `elapsed`、今日 `today`、总字数 `chars`、`streak`。
- 表格用现有 CJK 宽度对齐工具（`render.StringWidth`/`PadRight`）保证列对齐。

### 数据落点（store）

- 全局 `Stats`：`TotalSeconds int`、`TodaySeconds int`、`LastReadDate string`、`StreakDays int`。挂在 `Library`（如 `Library.Stats Stats`，`json:"stats,omitempty"`）。
- 每本书统计字段加到 `BookEntry`：`CharsRead int`、`FurthestChapter int`、`FurthestPara int`（均 `omitempty`）。
- 新增 `store` 纯函数（可单测、不依赖 UI）：
  - `RecordActivity(lib, now)`：更新 streak/日期、返回是否应计时；或 `AddReadingTime(lib, seconds)` + `TouchDay(lib, today)` 拆分。
  - `RecordProgress(lib, id, chapter, para, charsBetween)`：推进高水位并累加 `CharsRead`（仅当更远）。
- 向后兼容：所有新字段缺省零值；旧 `library.json` 不需迁移。

### 测试策略

- streak：连续两天 +1；隔天断签重置为 1；同日多次不重复加。
- 时长：`delta ≤ 300` 累加、`> 300` 不计；`TodaySeconds` 跨日清零。
- 字数高水位：前进累加、回看不重复、跨章累加。
- `StatsView.Render`：列对齐、TOTAL 正确、汇总块格式、空书架/零数据不 panic。

---

## 文档与发布

- 更新 `README.md`：
  - 特性加「命令行阅读模式」「摸鱼统计」；
  - 新增「命令行模式」一节，列出全部 REPL 命令；
  - 快捷键阅读表加 `c`（统计）；`m` 说明改为三模式循环；
  - 路线图把 v0.6 打钩。
- 发布：合并后打 `v0.6.0` tag，Release 工作流产出 `moyu-reader_v0.6.0.exe`（版本号注入已就绪）。

## 已知取舍

- repl 模式下，专用老板屏仍可达（`` ` ``）；`Esc` 回书架（注意书架会显示书名，非绝对安全，但与其它模式一致）。
- **字数高水位规则（统一、无歧义）**：每本书只记一个最远位置（章, 段）。当新位置 > 高水位时，把「旧高水位 → 新位置」之间所有段落的字符数一次性加到 `CharsRead`，并把高水位更新为新位置；新位置 ≤ 高水位时不变。因此 `cd` 向后跳章会把跳过的中间段落计为「已读」——这是有意的简化（摸鱼趣味统计，非精确），跨度计算与逐段推进等价。
- 统计为「摸鱼」趣味用途，非精确计时，允许近似。
