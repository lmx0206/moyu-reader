package render

// WrapParagraph wraps a single paragraph to lines no wider than maxWidth cells.
// CJK (width-2) runes may break between any two characters; ASCII words break on
// spaces and are not split unless a single word exceeds maxWidth. An empty
// paragraph yields a single empty line so paragraph spacing is preserved.
func WrapParagraph(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	var lines []string
	var line []rune
	lineW := 0
	flush := func() {
		// trim trailing spaces before breaking
		for len(line) > 0 && line[len(line)-1] == ' ' {
			line = line[:len(line)-1]
		}
		lines = append(lines, string(line))
		line = line[:0]
		lineW = 0
	}

	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == ' ' {
			if lineW == 0 {
				i++ // skip leading space
				continue
			}
			if lineW+1 > maxWidth {
				flush()
				i++
				continue
			}
			line = append(line, ' ')
			lineW++
			i++
			continue
		}

		// Build the next token: one wide rune, or a run of narrow non-space runes.
		var tok []rune
		tokW := 0
		if RuneWidth(r) == 2 {
			tok = append(tok, r)
			tokW = 2
			i++
		} else {
			for i < len(runes) && runes[i] != ' ' && RuneWidth(runes[i]) != 2 {
				tok = append(tok, runes[i])
				tokW += RuneWidth(runes[i])
				i++
			}
		}

		if lineW > 0 && lineW+tokW > maxWidth {
			flush()
		}
		if tokW > maxWidth {
			// token longer than a full line: hard-split by rune
			for _, rr := range tok {
				w := RuneWidth(rr)
				if lineW > 0 && lineW+w > maxWidth {
					flush()
				}
				line = append(line, rr)
				lineW += w
			}
		} else {
			line = append(line, tok...)
			lineW += tokW
		}
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
