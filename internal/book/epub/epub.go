// Package epub parses EPUB files into plain-text chapters.
package epub

import (
	"archive/zip"
	"io"
	"path"

	"moyureader/internal/book"
)

func init() { book.Register(".epub", Parse) }

// Parse opens an EPUB file and returns the fully parsed Book.
func Parse(filename string) (*book.Book, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	containerData, err := readZipFile(files["META-INF/container.xml"])
	if err != nil {
		return nil, err
	}
	opfPath, err := parseContainer(containerData)
	if err != nil {
		return nil, err
	}
	opfData, err := readZipFile(files[opfPath])
	if err != nil {
		return nil, err
	}
	meta, hrefs, err := parseOPF(opfData, path.Dir(opfPath))
	if err != nil {
		return nil, err
	}

	b := &book.Book{Title: meta.Title, Author: meta.Author}
	for _, href := range hrefs {
		data, err := readZipFile(files[href])
		if err != nil {
			continue // skip unreadable chapter rather than failing whole book
		}
		paras := htmlToParagraphs(string(data))
		if len(paras) == 0 {
			continue
		}
		title := paras[0]
		b.Chapters = append(b.Chapters, book.Chapter{Title: title, Paragraphs: paras})
	}
	return b, nil
}

// readZipFile reads the full contents of a zip entry, erroring if it is nil.
func readZipFile(f *zip.File) ([]byte, error) {
	if f == nil {
		return nil, errMissingEntry
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
