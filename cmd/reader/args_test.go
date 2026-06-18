package main

import (
	"path/filepath"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		in   []string
		mode string
		arg  string
	}{
		{[]string{}, "tui", ""},
		{[]string{"list"}, "list", ""},
		{[]string{"stream"}, "stream", ""},
		{[]string{"stream", "abc"}, "stream", "abc"},
		{[]string{"import", "x.epub"}, "import", "x.epub"},
		{[]string{"version"}, "version", ""},
		{[]string{"--version"}, "version", ""},
		{[]string{"-v"}, "version", ""},
		{[]string{"book.epub"}, "open", "book.epub"},
		{[]string{"C:\\x\\My Book.epub"}, "open", "C:\\x\\My Book.epub"},
	}
	for _, c := range cases {
		cmd := parseArgs(c.in)
		if cmd.Mode != c.mode || cmd.Arg != c.arg {
			t.Fatalf("parseArgs(%v) = {%q,%q}, want {%q,%q}", c.in, cmd.Mode, cmd.Arg, c.mode, c.arg)
		}
	}
}

func TestResolveDataDirEnvOverride(t *testing.T) {
	got := resolveDataDir("D:\\develop\\reader.exe", "D:\\mydata")
	if got != "D:\\mydata" {
		t.Fatalf("env override should win, got %q", got)
	}
}

func TestResolveDataDirExeAdjacent(t *testing.T) {
	got := resolveDataDir(filepath.Join("D:\\app", "reader.exe"), "")
	want := filepath.Join("D:\\app", "data")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
