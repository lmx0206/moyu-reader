package render

import (
	"reflect"
	"testing"
)

func TestWrapASCIIWord(t *testing.T) {
	got := WrapParagraph("hello world foo", 11)
	want := []string{"hello world", "foo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestWrapCJKBreaksAnywhere(t *testing.T) {
	// 6 双宽字 = 12 cells; maxWidth 6 cells = 3 chars per line.
	got := WrapParagraph("一二三四五六", 6)
	want := []string{"一二三", "四五六"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestWrapMixed(t *testing.T) {
	// "ab一二cd" widths: a1 b1 一2 二2 c1 d1 = 8; maxWidth 4.
	got := WrapParagraph("ab一二cd", 4)
	want := []string{"ab一", "二cd"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestWrapEmpty(t *testing.T) {
	got := WrapParagraph("", 10)
	want := []string{""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
