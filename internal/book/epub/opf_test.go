package epub

import (
	"reflect"
	"testing"
)

func TestParseContainerOPFPath(t *testing.T) {
	xml := `<?xml version="1.0"?>
	<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
	  <rootfiles>
	    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
	  </rootfiles>
	</container>`
	got, err := parseContainer([]byte(xml))
	if err != nil {
		t.Fatal(err)
	}
	if got != "OEBPS/content.opf" {
		t.Fatalf("got %q want OEBPS/content.opf", got)
	}
}

func TestParseOPFSpineOrder(t *testing.T) {
	xml := `<?xml version="1.0"?>
	<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
	  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
	    <dc:title>测试之书</dc:title>
	    <dc:creator>张三</dc:creator>
	  </metadata>
	  <manifest>
	    <item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
	    <item id="c2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
	    <item id="css" href="style.css" media-type="text/css"/>
	  </manifest>
	  <spine>
	    <itemref idref="c2"/>
	    <itemref idref="c1"/>
	  </spine>
	</package>`
	meta, hrefs, err := parseOPF([]byte(xml), "OEBPS")
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "测试之书" || meta.Author != "张三" {
		t.Fatalf("bad meta: %+v", meta)
	}
	want := []string{"OEBPS/ch2.xhtml", "OEBPS/ch1.xhtml"}
	if !reflect.DeepEqual(hrefs, want) {
		t.Fatalf("got %#v want %#v", hrefs, want)
	}
}

// Real-world EPUBs percent-encode hrefs in the OPF (spaces, CJK, etc.) while
// the zip entry name is the decoded form. parseOPF must decode hrefs so the
// later files[href] lookup succeeds; otherwise chapters are silently dropped.
func TestParseOPFDecodesPercentEncodedHref(t *testing.T) {
	xml := `<?xml version="1.0"?>
	<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
	  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
	    <dc:title>编码之书</dc:title>
	  </metadata>
	  <manifest>
	    <item id="c1" href="Text/chapter%201.xhtml" media-type="application/xhtml+xml"/>
	    <item id="c2" href="Text/%E7%AC%AC%E4%B8%80%E7%AB%A0.xhtml" media-type="application/xhtml+xml"/>
	  </manifest>
	  <spine>
	    <itemref idref="c1"/>
	    <itemref idref="c2"/>
	  </spine>
	</package>`
	_, hrefs, err := parseOPF([]byte(xml), "OEBPS")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"OEBPS/Text/chapter 1.xhtml", "OEBPS/Text/第一章.xhtml"}
	if !reflect.DeepEqual(hrefs, want) {
		t.Fatalf("hrefs not percent-decoded:\ngot  %#v\nwant %#v", hrefs, want)
	}
}
