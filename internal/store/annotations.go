package store

// AddAnnotation appends an annotation to the book with the given id. It is a
// no-op if the id is unknown. The caller is responsible for Save.
func AddAnnotation(lib *Library, id string, a Annotation) {
	e := lib.FindByID(id)
	if e == nil {
		return
	}
	e.Annotations = append(e.Annotations, a)
}

// DeleteAnnotation removes the annotation at index i for the book with the given
// id. It is a no-op if the id is unknown or i is out of range. The caller is
// responsible for Save.
func DeleteAnnotation(lib *Library, id string, i int) {
	e := lib.FindByID(id)
	if e == nil || i < 0 || i >= len(e.Annotations) {
		return
	}
	e.Annotations = append(e.Annotations[:i], e.Annotations[i+1:]...)
}
