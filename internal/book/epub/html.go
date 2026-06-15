package epub

import (
	"strings"

	"golang.org/x/net/html"
)

// blockTags trigger a paragraph break before and after their content.
var blockTags = map[string]bool{
	"p": true, "div": true, "h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true, "li": true, "blockquote": true,
	"section": true, "article": true, "tr": true,
}

// skipTags and their content are dropped entirely.
var skipTags = map[string]bool{
	"script": true, "style": true, "head": true,
}

// htmlToParagraphs parses an XHTML document body into trimmed text paragraphs.
// Block-level elements and <br> introduce paragraph boundaries; inline text is
// concatenated; runs of whitespace collapse to a single space.
func htmlToParagraphs(doc string) []string {
	node, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return nil
	}
	var paras []string
	var cur strings.Builder

	flush := func() {
		text := collapseSpaces(cur.String())
		if text != "" {
			paras = append(paras, text)
		}
		cur.Reset()
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skipTags[n.Data] {
			return
		}
		if n.Type == html.ElementNode && n.Data == "br" {
			flush()
			return
		}
		isBlock := n.Type == html.ElementNode && blockTags[n.Data]
		if isBlock {
			flush()
		}
		if n.Type == html.TextNode {
			cur.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if isBlock {
			flush()
		}
	}
	walk(node)
	flush()
	return paras
}

// collapseSpaces trims and collapses internal whitespace to single spaces.
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
