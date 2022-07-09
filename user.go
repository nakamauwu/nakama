package nakama

import (
	"context"
	"database/sql"
	"errors"
	"regexp"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrUserNotFound    = errs.NotFoundError("user not found")
	ErrUsernameTaken   = errs.ConflictError("username taken")
	ErrInvalidEmail    = errs.InvalidArgumentError("invalid email")
	ErrInvalidUsername = errs.InvalidArgumentError("invalid username")
)

var (
	reEmail    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,17}$`)
)

func (svc *Service) UserByUsername(ctx context.Context, username string) (User, error) {
	var out User

	if !isUsername(username) {
		return out, ErrInvalidUsername
	}

	out, err := svc.Queries.UserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrUserNotFound
	}

	return out, err
}

func isEmail(s string) bool {
	return reEmail.MatchString(s)
}

func isUsername(s string) bool {
	return reUsername.MatchString(s)
}
