package ui

import (
	"github.com/microcosm-cc/bluemonday"
)

func htmlSanitize(input string) string {
	p := bluemonday.NewPolicy()
	p.AllowStandardURLs()
	p.AllowAttrs("color", "data-mx-bg-color").OnElements("font")
	p.AllowElements("b", "i")
	return p.Sanitize(input)
}

func htmlStrip(input string) string {
	p := bluemonday.StrictPolicy()
	return p.Sanitize(input)
}
