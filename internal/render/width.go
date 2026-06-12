// Package render handles CJK-aware text wrapping and pagination.
package render

import "github.com/mattn/go-runewidth"

// cjk measures display width with ambiguous-width characters treated as narrow,
// which matches how Windows Terminal and most modern terminals render them.
// CJK ideographs and full-width punctuation remain width 2.
var cjk = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// RuneWidth returns the terminal cell width of r (1 or 2).
func RuneWidth(r rune) int { return cjk.RuneWidth(r) }

// StringWidth returns the total terminal cell width of s.
func StringWidth(s string) int { return cjk.StringWidth(s) }
