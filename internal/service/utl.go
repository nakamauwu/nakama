package service

import (
	"github.com/jackc/pgx"
)

func isUniqueViolation(err error) bool {
	pgerr, ok := err.(pgx.PgError)
	return ok && pgerr.Code == "23505"
}
