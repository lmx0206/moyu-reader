// Package store persists the bookshelf and reading progress as JSON.
package store

// Prefs holds disguise style ("log"/"build"/"git") and reading mode
// ("shell"/"inline").
type Prefs struct {
	Style string `json:"style"`
	Mode  string `json:"mode"`
}

// Progress is a stable reading position: chapter index + paragraph index
// (layout-independent, so it survives window-width changes).
type Progress struct {
	Chapter int `json:"chapter"`
	Para    int `json:"para"`
}

// Annotation marks a reading position with an optional note. An empty Note is a
// plain bookmark; a non-empty Note is an annotation.
type Annotation struct {
	Chapter   int    `json:"chapter"`
	Para      int    `json:"para"`
	Note      string `json:"note,omitempty"`
	CreatedAt string `json:"createdAt"`
}

// BookEntry is one imported book.
type BookEntry struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Author       string       `json:"author"`
	File         string       `json:"file"` // relative to data dir, e.g. "books/<id>.epub"
	AddedAt      string       `json:"addedAt"`
	LastOpenedAt string       `json:"lastOpenedAt"`
	Progress     Progress     `json:"progress"`
	Prefs        Prefs        `json:"prefs"`
	Annotations  []Annotation `json:"annotations,omitempty"`
	Broken       bool         `json:"broken,omitempty"`
}

// Library is the whole bookshelf plus global defaults.
type Library struct {
	LastBookID string      `json:"lastBookId"`
	Global     Prefs       `json:"global"`
	Books      []BookEntry `json:"books"`
}

// NewLibrary returns an empty library with sane default prefs.
func NewLibrary() *Library {
	return &Library{Global: Prefs{Style: "log", Mode: "shell"}}
}

// FindByID returns a pointer to the matching book entry, or nil.
func (l *Library) FindByID(id string) *BookEntry {
	for i := range l.Books {
		if l.Books[i].ID == id {
			return &l.Books[i]
		}
	}
	return nil
}
