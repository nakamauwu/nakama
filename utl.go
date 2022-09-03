package nakama

import (
	"errors"
	"regexp"
	"strings"

	"github.com/lib/pq"
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

func isPqError(err error, codeName string, columns ...string) bool {
	var e *pq.Error
	if !errors.As(err, &e) {
		return false
	}

	if e.Code.Name() != codeName {
		return false
	}

	if len(columns) == 0 {
		return true
	}

	for _, col := range columns {
		if strings.Contains(strings.ToLower(e.Error()), strings.ToLower(col)) {
			return true
		}
	}

	return false
}

// func isPqNotNullViolationError(err error, columns ...string) bool {
// 	return isPqError(err, "not_null_violation", columns...)
// }

func isPqForeignKeyViolationError(err error, columns ...string) bool {
	return isPqError(err, "foreign_key_violation", columns...)
}

func isPqUniqueViolationError(err error, columns ...string) bool {
	return isPqError(err, "unique_violation", columns...)
}
