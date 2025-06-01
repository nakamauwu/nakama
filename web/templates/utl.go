package templates

import (
	"html/template"

	"mvdan.cc/xurls/v2"
)

var reURL = xurls.Relaxed()

func linkify(s string) string {
	s = template.HTMLEscapeString(s)
	s = reURL.ReplaceAllString(s,
		`<a href="$0" target="_blank" rel="noopener noreferrer">$0</a>`,
	)
	return s
}
