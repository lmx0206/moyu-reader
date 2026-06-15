package disguise

import (
	"strings"
	"testing"
)

func TestRenderInlinePrefixesEachLine(t *testing.T) {
	th := ThemeByName("log")
	body := []string{"林尘睁开眼", "看见月光"}
	out := RenderInline(th, body, 100)
	if len(out) != 2 {
		t.Fatalf("want 2 lines got %d", len(out))
	}
	if !strings.HasSuffix(out[0], "林尘睁开眼") {
		t.Fatalf("line0 should end with payload: %q", out[0])
	}
	if !strings.Contains(out[0], " - ") {
		t.Fatalf("line0 should contain log prefix: %q", out[0])
	}
}

func TestRenderShellFiveSectionLayout(t *testing.T) {
	th := ThemeByName("build")
	body := []string{"正文一", "正文二"}
	out := RenderShell(th, body, 40, "ch.1/10 · 本章 1/1页 · 0%")
	// 4 装饰行(顶栏/分隔/分隔/底栏) + 2 正文 = 6
	if len(out) != 6 {
		t.Fatalf("want 6 lines, got %d: %#v", len(out), out)
	}
	if !strings.Contains(out[0], "gradle") {
		t.Fatalf("top bar should be build theme: %q", out[0])
	}
	if !strings.Contains(out[1], "─") {
		t.Fatalf("line1 should be separator: %q", out[1])
	}
	// RenderShell no longer indents — body is placed verbatim (caller indents).
	if out[2] != "正文一" || out[3] != "正文二" {
		t.Fatalf("body must be verbatim: %#v", out)
	}
	if !strings.Contains(out[4], "─") {
		t.Fatalf("line4 should be separator: %q", out[4])
	}
	if !strings.Contains(out[5], "SUCCESSFUL") {
		t.Fatalf("bottom bar should be build theme: %q", out[5])
	}
}
