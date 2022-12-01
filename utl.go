package nakama

import (
	"regexp"
	"strings"
)

var reMoreThanThreeWhitespaces = regexp.MustCompile(`(\s){3,}`)

func smartTrim(s string) string {
	s = strings.TrimSpace(s)
	s = reMoreThanThreeWhitespaces.ReplaceAllString(s, "$1$1")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func ptr[T any](v T) *T {
	return &v
}
