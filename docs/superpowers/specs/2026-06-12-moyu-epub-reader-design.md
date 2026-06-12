# 摸鱼看书神器 —— 终端 EPUB 阅读器设计文档

- 日期：2026-06-12
- 状态：已通过 brainstorming，待用户复核 spec
- 目标平台：Windows（终端 / Android Studio 集成终端 / PowerShell / cmd / Windows Terminal）

## 1. 一句话定位

一个用 Go 编译成**单文件 `.exe`**、零依赖、可放 U 盘随身带的终端 EPUB 阅读器。
核心卖点是**伪装**：远看像在跑日志 / 编译 / 看代码 diff，近看才是小说。支持
全屏 TUI 与内联流式 CLI 两种启动方式。

> 现有终端阅读器（epr / epy / baca / lue）都能记忆进度，但没有任何一个有
> "把阅读伪装成工作内容"的能力——这是本项目的差异化核心。

## 2. 两种启动方式

| 方式 | 启动 | 场景 |
|------|------|------|
| **全屏 TUI** | `reader.exe`（无参数） | 默认，完整体验：书架、导入、阅读、老板键 |
| **内联流式 CLI** | `reader.exe stream [书名/id]` | 在 Android Studio 等集成终端里，像一条不断吐日志的命令，回车出下一段。不接管屏幕、保留滚动区 |

> 小贴士：exe 文件名可自行改成 `tail.exe` / `gradlew.exe` / `build.exe`，
> 进程列表里更不显眼。

## 3. 三层伪装体系（灵魂）

### 3.1 阅读模式（全屏 TUI 内，按 `m` 切换）

- **模式 A —— 外壳伪装**（默认）：正文正常排版、舒服阅读；顶/底状态栏与边框
  伪装成日志工具 / IDE。状态栏示例：`build: OK 73% · 142 tests passed · ch.3/57 · 21%`
- **模式 B —— 正文藏日志行**：每行正文加上假时间戳 + 级别 + 类名前缀，正文作为
  日志 message 体，最高拟真，有人靠近时切到此模式。

### 3.2 伪装风格（A/B 模式与老板键屏都套用，按 `Tab` 循环，记住上次选择）

- **log**：`[14:23:01] INFO  OrderSvc - <正文>`，带时间戳/级别/类名
- **build**：模拟 Gradle/Maven/webpack/npm 编译输出，`Compiling module... BUILD SUCCESSFUL in 12s`
- **git**：模拟 `git log` / `git diff` / 一屏源码，正文藏在注释或字符串里

### 3.3 老板键（全屏 TUI 内，按 `` ` `` 或 `b`）

满屏**自动滚动**的假日志/编译输出，**完全不含小说内容**，看着像程序在跑。
按任意键秒回原阅读位置。风格沿用当前选择的伪装风格。

### 3.4 内联流式 CLI 的伪装

内联模式本身就是"一条命令在持续输出日志"，天然伪装。固定采用**模式 B**的渲染
（正文藏在日志/编译/git 行里），风格同样支持 log/build/git。

## 4. 技术架构（Go + Bubbletea TUI）

模块按单一职责拆分，各自可独立测试：

```
cmd/reader/main.go        装配、解析子命令/参数、配置、终端初始化
internal/epub/            EPUB 解析
internal/render/          CJK 感知的排版与分页引擎
internal/disguise/        三种风格 × 两种模式的渲染器 + 老板键假屏生成器
internal/store/           书架与进度持久化
internal/ui/              Bubbletea 全屏 TUI（书架 / 阅读 / 老板键）
internal/stream/          内联流式 CLI 模式
```

### 4.1 `internal/epub` —— EPUB 解析

- 用标准库 `archive/zip` 打开 epub；`encoding/xml` 解析 `META-INF/container.xml`
  → 定位 OPF → 读 `manifest` + `spine` 得到章节顺序
- 用 `golang.org/x/net/html` 把每个 XHTML 章节剥离为纯文本段落（保留段落边界，
  丢弃样式/脚本；图片/表格降级为占位或忽略）
- 输出：
  ```go
  type Book struct {
      Title, Author string
      Chapters []Chapter
  }
  type Chapter struct {
      Title string
      Paragraphs []string
  }
  ```
- 依赖：仅标准库 + `golang.org/x/net/html`，保证单文件可静态编译

### 4.2 `internal/render` —— 排版与分页引擎

- 用 `github.com/mattn/go-runewidth` 处理 **CJK 双宽字符**的中英混排换行
- 输入：纯文本段落 + 终端宽/高 → 输出：换行后的显示行序列，再按高度分页
- **进度定位**用稳定坐标 `(章节索引, 行索引)`，不受窗口大小变化影响时尽量保持
  在同一章节起点附近恢复
- 纯函数式、无副作用，便于 golden 测试

### 4.3 `internal/disguise` —— 伪装渲染层

- `Theme` 接口：`log` / `build` / `git` 三个实现
- 两种正文渲染：`RenderShell`（外壳伪装，正文正常）与 `RenderInline`（正文藏行里）
- `BossScreen`：按风格生成无尽的自动滚动假行（不含正文）
- 全部纯函数，golden 测试

### 4.4 `internal/store` —— 持久化

- 数据目录：**exe 同级的 `data/` 文件夹**（便携，易删易藏，可随 U 盘带走）
- 导入时把 epub **复制进 `data/books/<id>.epub`**，自包含
- `data/library.json`：
  ```json
  {
    "lastBookId": "abc123",
    "global": { "style": "log", "mode": "shell" },
    "books": [
      {
        "id": "abc123",
        "title": "...", "author": "...",
        "file": "books/abc123.epub",
        "addedAt": "2026-06-12T14:00:00Z",
        "lastOpenedAt": "2026-06-12T14:30:00Z",
        "progress": { "chapter": 3, "line": 128 },
        "prefs": { "style": "log", "mode": "shell" }
      }
    ]
  }
  ```
- 进度在翻页/退出时持续写回；写入采用临时文件 + rename 原子替换，防止损坏

### 4.5 `internal/ui` —— 全屏 TUI（Bubbletea）

三个界面（Bubbletea model 状态机）：

- **书架**：列表，最近读的高亮置顶；导入 / 删除 / 打开
- **阅读**：当前阅读模式 + 风格的渲染视图
- **老板键覆盖层**：自动滚动假屏

### 4.6 `internal/stream` —— 内联流式 CLI

- 不使用 alt-screen，直接写 stdout，保留终端滚动区
- 启动续读上次的书（或指定书）；打印一屏（约满终端高度）的伪装行后等待
- 交互：**回车** = 下一段；输入 `q` 回车 = 退出；输入 `b` 回车 = 回退一段；
  输入 `t` 回车 = 切换风格（行式输入，跨终端最稳）
- 退出/翻页时持续写回进度，与全屏 TUI 共享同一 `library.json`

## 5. 按键设计（全屏 TUI）

- **书架**：`↑↓` 选择 · `Enter` 打开 · `i` 导入(输入路径) · `d` 删除 · `q` 退出
- **阅读**：`Space`/`→`/`PgDn` 翻页 · `↑↓` 行滚动 · `Tab` 切风格 · `m` 切 A/B 模式
  · `` ` ``/`b` 老板键 · `g` 目录跳转 · `Esc` 回书架
- **老板键屏**：任意键返回

## 6. 命令行接口

```
reader.exe                      启动全屏 TUI（书架）
reader.exe <某本书.epub>        导入并直接打开该 epub（全屏）
reader.exe stream [书名/id]     内联流式 CLI 模式（续读，或指定书）
reader.exe import <路径.epub>   仅导入到书架，不打开
reader.exe list                 列出书架（可脚本化）
```

## 7. 健壮性与错误处理

- 坏 epub / 解析失败 / 文件丢失 → 书架中标记错误条目，不崩溃
- 终端过小 → 降级布局，至少保证可读
- `library.json` 损坏 → 备份旧文件并以空书架重建，提示用户
- 中英混排 → 严格用 runewidth 处理，避免错位换行

## 8. 测试策略

- `epub`：用一两本 fixture epub 验证解析（章节顺序、纯文本提取）
- `render`：CJK 中英混排换行与分页的 golden 测试
- `disguise`：三风格 × 两模式 + 老板键的 golden 输出
- `store`：library.json 读写、原子替换、损坏恢复
- `ui`：Bubbletea model 的 update 逻辑（按键 → 状态转移）
- `stream`：分段输出与进度推进逻辑

## 9. 明确不做（YAGNI）

- 不做 TTS、字典、注释/书签同步、网络书城、PDF/mobi 等其它格式（首版只 EPUB）
- 不做图片/表格的精细渲染（降级为占位或忽略）
- 不做多端云同步
