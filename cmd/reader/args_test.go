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
	// An already-absolute override is returned as-is. Use an OS-absolute path
	// (t.TempDir) so the test is portable: filepath.Abs is a no-op on it, on
	// both Windows and the Linux CI runner.
	abs := t.TempDir()
	got := resolveDataDir(filepath.Join("D:\\develop", "reader.exe"), abs)
	if got != abs {
		t.Fatalf("absolute env override should win as-is, got %q want %q", got, abs)
	}
}

func TestResolveDataDirExeAdjacent(t *testing.T) {
	got := resolveDataDir(filepath.Join("D:\\app", "reader.exe"), "")
	want := filepath.Join("D:\\app", "data")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// A relative MOYU_DATA must be absolutized so the same value resolves to the
// same library regardless of the process's working directory (double-click vs
// shell launch).
func TestResolveDataDirEnvAbsolutized(t *testing.T) {
	got := resolveDataDir("D:\\app\\reader.exe", "rel/data")
	if !filepath.IsAbs(got) {
		t.Fatalf("relative env override should be absolutized, got %q", got)
	}
}
