# moyu-reader · 摸鱼终端阅读器

在终端里读 EPUB 小说，远看像在跑日志/编译/看 diff 的摸鱼神器。

## 构建

需要 Go（本机在 `D:\develop\Go`）：

    go build -ldflags "-s -w" -o reader.exe ./cmd/reader

产物是单文件 `reader.exe`，零依赖，拷到任意 Windows 机器即用。数据存在 exe 同级的 `data/` 文件夹（可用环境变量 `MOYU_DATA` 改）。

## 用法

    reader.exe                 打开全屏 TUI（书架）
    reader.exe 某本书.epub      导入并直接阅读
    reader.exe import 路径.epub  仅导入到书架
    reader.exe list            列出书架
    reader.exe stream [id]      内联流式模式（在集成终端里像一条吐日志的命令）

## 阅读快捷键（全屏 TUI）

- `Space`/`→`/`PgDn` 翻页 · `↑↓` 行滚动
- `Tab` 切伪装风格（log/build/git）
- `m` 切阅读模式（外壳伪装 / 正文藏日志行）
- `` ` `` 或 `b` 老板键（满屏假日志，任意键返回）
- `i` 导入 · `d` 删除 · `Esc` 回书架 · `q` 退出

## 内联流式模式

    reader.exe stream

回车出下一段；`b` 回退；`t` 切风格；`q` 退出。退出自动存进度。
