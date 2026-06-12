package main

import (
	"fmt"
	"os"
	"path/filepath"

	"moyureader/internal/epub"
	"moyureader/internal/store"
	"moyureader/internal/stream"
	"moyureader/internal/ui"
)

func main() {
	cmd := parseArgs(os.Args[1:])

	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}
	dataDir := resolveDataDir(exe, os.Getenv("MOYU_DATA"))
	st := store.New(dataDir)
	lib, err := st.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法读取书架:", err)
		os.Exit(1)
	}

	switch cmd.Mode {
	case "list":
		runList(lib)
	case "import":
		runImport(st, lib, cmd.Arg)
	case "open":
		runOpen(st, lib, cmd.Arg)
	case "stream":
		runStream(st, lib, cmd.Arg)
	default:
		if err := ui.Run(st, lib, ""); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runList(lib *store.Library) {
	if len(lib.Books) == 0 {
		fmt.Println("书架为空。用 reader <某本书.epub> 导入。")
		return
	}
	for _, b := range lib.Books {
		mark := " "
		if b.ID == lib.LastBookID {
			mark = "*"
		}
		fmt.Printf("%s %-8s  %s — %s\n", mark, b.ID, b.Title, b.Author)
	}
}

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

func runImport(st *store.Store, lib *store.Library, path string) {
	if path == "" {
		fmt.Fprintln(os.Stderr, "用法: reader import <某本书.epub>")
		os.Exit(2)
	}
	entry, err := importPath(st, lib, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "导入失败:", err)
		os.Exit(1)
	}
	fmt.Printf("已导入: %s — %s (id=%s)\n", entry.Title, entry.Author, entry.ID)
}

func runOpen(st *store.Store, lib *store.Library, path string) {
	entry, err := importPath(st, lib, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "打不开:", err)
		os.Exit(1)
	}
	if err := ui.Run(st, lib, entry.ID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runStream(st *store.Store, lib *store.Library, idOrEmpty string) {
	id := idOrEmpty
	if id == "" {
		id = lib.LastBookID
	}
	entry := lib.FindByID(id)
	if entry == nil {
		fmt.Fprintln(os.Stderr, "没有可续读的书。先用 reader <某本书.epub> 导入。")
		os.Exit(1)
	}
	book, err := epub.Parse(filepath.Join(st.Dir(), filepath.FromSlash(entry.File)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "解析失败:", err)
		os.Exit(1)
	}
	s := stream.NewStreamer(book, entry.Progress, entry.Prefs.Style, 18)
	stream.Run(s, os.Stdin, os.Stdout, func(p store.Progress, style string) {
		store.UpdateProgress(lib, entry.ID, p, store.Prefs{Style: style, Mode: "inline"})
		_ = st.Save(lib)
	})
}
