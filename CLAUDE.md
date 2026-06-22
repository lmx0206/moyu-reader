# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`moyu-reader` (摸鱼终端阅读器) is a Windows terminal EPUB/TXT novel reader written in Go (module `moyureader`). Its defining constraint: **everything on screen must look like developer tool output** (logs / build / git diff / docker / npm / pytest) so reading a novel at work looks like working. Any UI or disguise change must preserve this stealth — never leak the app name, book titles, or anything that reads as "a novel reader" into the disguised output.

## Commands

```bash
go test ./...                                   # all tests
go test ./internal/ui/ -run TestReaderJumpTo    # single test (pkg + -run regex)
go vet ./...                                     # static check (CI gate)
go build ./...                                   # compile everything

# one-click build with the version auto-derived from git (no manual version):
./scripts/build.ps1   # PowerShell (primary)
./scripts/build.sh    # Git Bash / WSL
# both produce moyu-reader_v<git-describe>.exe with version baked in via ldflags

# plain build (version shows as "dev")
go build -ldflags "-s -w" -o reader.exe ./cmd/reader
# manual injected version (what the scripts automate)
go build -ldflags "-s -w -X moyureader/internal/version.Version=0.6.2" -o moyu-reader_v0.6.2.exe ./cmd/reader
```

CI (`.github/workflows/ci.yml`) runs vet + test + build on every push/PR. `release.yml` triggers on a `v*` tag: it injects the tag (minus the leading `v`) into `version.Version` via ldflags and publishes `moyu-reader_v<tag>.exe` to Releases.

Note: gopls-lsp surfaces staticcheck + Go 1.22 "modernize" hints (rangeint/minmax) that `go vet`/CI do **not** run — those hints are advisory, not build failures.

## Architecture

Layered, dependency flows downward: `cmd/reader` → `ui`/`stream` → `book`/`render`/`disguise`/`store`.

- **`internal/book`** — format-agnostic `Book`/`Chapter` types plus a stdlib-style registry (`Register(ext, parser)` / `Open(path)`, keyed by extension, panics on misregistration). Backends live in `internal/book/epub` and `internal/book/txt` and self-register via `init()`; `internal/book/formats` blank-imports them and is the only thing `cmd/reader` imports for format support. **Adding a format = new backend package + one blank import**, nothing else.

- **`internal/render`** — CJK-aware text layout. `WrapParagraph`/`LayoutChapter` wrap to a display width; `StringWidth`/`RuneWidth` measure cells via go-runewidth (ambiguous-width treated as narrow to match Windows Terminal). `anchor.go` maps between paragraph index and display-line index (`ParaStartLine`/`LineToPara`) — this is what makes positions width-independent.

- **`internal/disguise`** — the stealth layer. `Theme` interface (log/build/git/docker/npm/pytest) provides per-line prefixes + chrome. Three render modes: **shell** (`RenderShell`, log-viewer chrome, body indented), **inline** (`RenderInline`, each body line prefixed with a fake log line — highest stealth), and the **boss screen** (pure fake output, no novel text). `PrefixWidth(th)` returns a theme's widest prefix so inline callers wrap to `width - prefix` and never clip the tail.

- **`internal/store`** — `data/library.json` persistence (`Library`/`BookEntry`/`Progress`/`Prefs`/`Annotation`). `FindByID` returns a pointer **into** the slice, so callers mutate in place then `Save`. Data dir defaults to `data/` next to the exe, overridable by `MOYU_DATA`.

- **`internal/ui`** — full-screen bubbletea TUI. `Model` is the root (screen state machine: shelf/reader/import/TOC/help/annotate/annotList); `ReaderView` owns reading position + disguised rendering; `ShelfView`/`TOCView`/`AnnotationView` are sub-views. `AnnotationView` is disguised as a debugger `breakpoints (N)` panel.

- **`internal/stream`** — the inline CLI reader (`reader stream`): emits disguised chunks to stdout and reads commands from stdin; renders with `RenderInline(..., 0)` (no truncation, terminal soft-wraps).

### Key cross-cutting decision: paragraph-anchored positions

Reading progress **and** annotations are anchored by `(chapter, paragraph)`, never by display-line index, because the line index is width-dependent. `ReaderView.SetSize` re-anchors to the top paragraph on resize so the view reflows instead of losing content. When touching navigation, persistence, or wrapping, keep positions paragraph-based and round-trip through `render.ParaStartLine`/`LineToPara`. (Consequence: pre-v0.5 saved `Progress.line` does not migrate — it resets to chapter start, by design.)

## Conventions

- **TDD**: write the failing test first, then implement; one logical change per commit.
- **bubbletea v1.3.x + lipgloss v1.1.x** (`github.com/charmbracelet/...`) — NOT the v2/`charm.land` packages.
- All width math goes through `render.StringWidth`/`RuneWidth` (CJK is double-width); never use `len()` for display width.
- Design specs and implementation plans live in `docs/superpowers/`.
- Before tagging a release, update `README.md` (features / keybindings / roadmap checkboxes) so docs match the implementation.
