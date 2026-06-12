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
