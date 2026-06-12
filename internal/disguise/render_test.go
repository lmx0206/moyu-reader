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

func TestRenderShellWrapsBodyWithChrome(t *testing.T) {
	th := ThemeByName("build")
	body := []string{"正文一", "正文二"}
	out := RenderShell(th, body, 40, "ch.1/10")
	if len(out) < 4 {
		t.Fatalf("shell output too short: %v", out)
	}
	if !strings.Contains(out[0], "gradle") {
		t.Fatalf("first line should be build header: %q", out[0])
	}
	if out[1] != "正文一" || out[2] != "正文二" {
		t.Fatalf("body not preserved verbatim: %#v", out)
	}
	last := out[len(out)-1]
	if !strings.Contains(last, "SUCCESSFUL") {
		t.Fatalf("last line should be build footer: %q", last)
	}
}
