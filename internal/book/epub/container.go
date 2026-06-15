package epub

import "encoding/xml"

// parseContainer returns the OPF rootfile path from META-INF/container.xml.
func parseContainer(data []byte) (string, error) {
	var c struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", err
	}
	if len(c.Rootfiles) == 0 {
		return "", errNoRootfile
	}
	return c.Rootfiles[0].FullPath, nil
}
