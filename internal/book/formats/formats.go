// Package formats wires all built-in book format backends. Import it (usually
// for side effects) to make every supported format available via book.Open.
package formats

import (
	_ "moyureader/internal/book/epub"
)
