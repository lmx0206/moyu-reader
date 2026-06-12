package store

import "testing"

func TestFindByIDAndPromote(t *testing.T) {
	lib := &Library{Books: []BookEntry{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
	}}
	e := lib.FindByID("b")
	if e == nil || e.Title != "B" {
		t.Fatalf("FindByID failed: %v", e)
	}
	if lib.FindByID("zzz") != nil {
		t.Fatal("missing id should return nil")
	}
}

func TestDefaultPrefs(t *testing.T) {
	lib := NewLibrary()
	if lib.Global.Style != "log" || lib.Global.Mode != "shell" {
		t.Fatalf("bad defaults: %+v", lib.Global)
	}
}
