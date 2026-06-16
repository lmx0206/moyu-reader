package store

import "testing"

func TestAddAndDeleteAnnotation(t *testing.T) {
	lib := NewLibrary()
	lib.Books = append(lib.Books, BookEntry{ID: "a"})
	AddAnnotation(lib, "a", Annotation{Chapter: 1, Para: 3, Note: "hi"})
	AddAnnotation(lib, "a", Annotation{Chapter: 2, Para: 0})
	e := lib.FindByID("a")
	if len(e.Annotations) != 2 {
		t.Fatalf("want 2 annotations, got %d", len(e.Annotations))
	}
	DeleteAnnotation(lib, "a", 0)
	if len(e.Annotations) != 1 || e.Annotations[0].Chapter != 2 {
		t.Fatalf("delete wrong: %+v", e.Annotations)
	}
	DeleteAnnotation(lib, "a", 99)  // out of range -> no-op
	DeleteAnnotation(lib, "zzz", 0) // unknown id -> no-op
	if len(e.Annotations) != 1 {
		t.Fatalf("no-op delete changed slice: %+v", e.Annotations)
	}
}

func TestAddAnnotationUnknownID(t *testing.T) {
	lib := NewLibrary()
	AddAnnotation(lib, "nope", Annotation{}) // must not panic
}
