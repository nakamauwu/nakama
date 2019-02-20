package service

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/lib/pq"
)

const (
	minPageSize     = 1
	defaultPageSize = 10
	maxPageSize     = 99
)

var queriesCache = map[string]*template.Template{}

func isUniqueViolation(err error) bool {
	pqerr, ok := err.(*pq.Error)
	return ok && pqerr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	pqerr, ok := err.(*pq.Error)
	return ok && pqerr.Code == "23503"
}

func buildQuery(text string, data map[string]interface{}) (string, []interface{}, error) {
	t, ok := queriesCache[text]
	if !ok {
		var err error
		t, err = template.New("query").Parse(text)
		if err != nil {
			return "", nil, fmt.Errorf("could not parse sql query template: %v", err)
		}

		queriesCache[text] = t
	}

	var wr bytes.Buffer
	if err := t.Execute(&wr, data); err != nil {
		return "", nil, fmt.Errorf("could not apply sql query data: %v", err)
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
