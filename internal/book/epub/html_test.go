package epub

import (
	"reflect"
	"testing"
)

func TestHTMLToParagraphs(t *testing.T) {
	html := `<html><body>
		<h1>第一章</h1>
		<p>林尘睁开了眼。</p>
		<p>他看见<em>窗外</em>的月光。</p>
		<div>另一段。<br/>换行后。</div>
		<script>ignore();</script>
	</body></html>`
	got := htmlToParagraphs(html)
	want := []string{
		"第一章",
		"林尘睁开了眼。",
		"他看见窗外的月光。",
		"另一段。",
		"换行后。",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestHTMLToParagraphsCollapsesWhitespace(t *testing.T) {
	got := htmlToParagraphs("<p>  hello   world  </p>")
	want := []string{"hello world"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
