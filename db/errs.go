package db

import (
	"errors"
	"strings"

	"github.com/lib/pq"
)

func IsPqError(err error, codeName string, columns ...string) bool {
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

func IsPqNotNullViolationError(err error, columns ...string) bool {
	return IsPqError(err, "not_null_violation", columns...)
}

func IsPqForeignKeyViolationError(err error, columns ...string) bool {
	return IsPqError(err, "foreign_key_violation", columns...)
}

func IsPqUniqueViolationError(err error, columns ...string) bool {
	return IsPqError(err, "unique_violation", columns...)
}
