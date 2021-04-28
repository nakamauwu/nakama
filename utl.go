package nakama

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/lib/pq"
)

const (
	minPageSize     = 1
	defaultPageSize = 10
	maxPageSize     = 99
)

var queriesCache sync.Map
var (
	reUUID                = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
	reMultiSpace          = regexp.MustCompile(`(\s)+`)
	reMoreThan2Linebreaks = regexp.MustCompile(`(\n){2,}`)
	reMentions            = regexp.MustCompile(`\B@([a-zA-Z][a-zA-Z0-9_-]{0,17})`)
)

func isUniqueViolation(err error) bool {
	pqerr, ok := err.(*pq.Error)
	return ok && pqerr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	pqerr, ok := err.(*pq.Error)
	return ok && pqerr.Code == "23503"
}

func buildQuery(text string, data map[string]interface{}) (string, []interface{}, error) {
	var t *template.Template
	v, ok := queriesCache.Load(text)
	if !ok {
		var err error
		t, err = template.New("query").Parse(text)
		if err != nil {
			return "", nil, fmt.Errorf("could not parse sql query template: %w", err)
		}

		queriesCache.Store(text, t)
	} else {
		t = v.(*template.Template)
	}

	var wr bytes.Buffer
	if err := t.Execute(&wr, data); err != nil {
		return "", nil, fmt.Errorf("could not apply sql query data: %w", err)
	}

	query := wr.String()
	args := []interface{}{}
	for key, val := range data {
		if !strings.Contains(query, "@"+key) {
			continue
		}

		args = append(args, val)
		query = strings.Replace(query, "@"+key, fmt.Sprintf("$%d", len(args)), -1)
	}
	return query, args, nil
}

func normalizePageSize(i int) int {
	if i == 0 {
		return defaultPageSize
	}
	if i < minPageSize {
		return minPageSize
	}
	if i > maxPageSize {
		return maxPageSize
	}
	return i
}

func smartTrim(s string) string {
	oldLines := strings.Split(s, "\n")
	newLines := []string{}
	for _, line := range oldLines {
		line = strings.TrimSpace(reMultiSpace.ReplaceAllString(line, "$1"))
		newLines = append(newLines, line)
	}
	s = strings.Join(newLines, "\n")
	s = reMoreThan2Linebreaks.ReplaceAllString(s, "$1$1")
	return strings.TrimSpace(s)
}

func collectMentions(s string) []string {
	m := map[string]struct{}{}
	u := []string{}
	for _, submatch := range reMentions.FindAllStringSubmatch(s, -1) {
		val := submatch[1]
		if _, ok := m[val]; !ok {
			m[val] = struct{}{}
			u = append(u, val)
		}
	}
	return u
}

func cloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}
	u2 := new(url.URL)
	*u2 = *u
	if u.User != nil {
		u2.User = new(url.Userinfo)
		*u2.User = *u.User
	}
	return u2
}
