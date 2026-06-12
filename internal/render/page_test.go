package render

import (
	"reflect"
	"testing"
)

func TestLayoutChapterInsertsBlankBetweenParagraphs(t *testing.T) {
	paras := []string{"一二三四", "abcd"}
	got := LayoutChapter(paras, 4)
	// "一二三四" -> ["一二","三四"]; blank; "abcd" -> ["abcd"]
	want := []string{"一二", "三四", "", "abcd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestPaginate(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got := Paginate(lines, 2)
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestLineToPage(t *testing.T) {
	if got := LineToPage(5, 2); got != 2 {
		t.Fatalf("LineToPage(5,2)=%d want 2", got)
	}
	if got := LineToPage(0, 2); got != 0 {
		t.Fatalf("LineToPage(0,2)=%d want 0", got)
	}
}
