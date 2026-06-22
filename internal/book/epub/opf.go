package epub

import (
	"encoding/xml"
	"errors"
	"net/url"
	"path"
)

var (
	errNoRootfile   = errors.New("epub: no rootfile in container.xml")
	errNoSpine      = errors.New("epub: empty spine")
	errMissingEntry = errors.New("epub: required archive entry missing")
	errNoChapters   = errors.New("epub: no readable chapters (spine hrefs did not match archive entries)")
)

// Meta holds book-level metadata extracted from the OPF.
type Meta struct {
	Title  string
	Author string
}

// parseOPF parses the OPF package document. opfDir is the directory of the OPF
// file inside the archive, used to resolve relative hrefs to archive paths.
// It returns metadata and the ordered list of chapter file paths (from spine).
func parseOPF(data []byte, opfDir string) (Meta, []string, error) {
	var pkg struct {
		Metadata struct {
			Title   string `xml:"title"`
			Creator string `xml:"creator"`
		} `xml:"metadata"`
		Manifest []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return Meta{}, nil, err
	}
	if len(pkg.Spine) == 0 {
		return Meta{}, nil, errNoSpine
	}
	hrefByID := make(map[string]string, len(pkg.Manifest))
	for _, it := range pkg.Manifest {
		hrefByID[it.ID] = it.Href
	}
	var hrefs []string
	for _, ref := range pkg.Spine {
		href, ok := hrefByID[ref.IDRef]
		if !ok {
			continue
		}
		hrefs = append(hrefs, joinArchive(opfDir, href))
	}
	meta := Meta{Title: pkg.Metadata.Title, Author: pkg.Metadata.Creator}
	return meta, hrefs, nil
}

// joinArchive joins a directory and a relative href into a clean archive path
// using forward slashes (zip archives always use "/"). The href is
// percent-decoded first: OPF hrefs are URL-encoded (spaces as %20, CJK as %XX
// bytes) while zip entry names are the decoded form, so decoding is required
// for the later files[href] lookup to match.
func joinArchive(dir, href string) string {
	if decoded, err := url.PathUnescape(href); err == nil {
		href = decoded
	}
	if dir == "" || dir == "." {
		return path.Clean(href)
	}
	return path.Clean(dir + "/" + href)
}
