package nakama

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrUserNotFound    = errs.NotFoundError("user not found")
	ErrUsernameTaken   = errs.ConflictError("username taken")
	ErrInvalidUserID   = errs.InvalidArgumentError("invalid user ID")
	ErrInvalidEmail    = errs.InvalidArgumentError("invalid email")
	ErrInvalidUsername = errs.InvalidArgumentError("invalid username")
)

var (
	reEmail    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,17}$`)
)

type User struct {
	ID             string
	Email          string
	Username       string
	PostsCount     int32
	FollowersCount int32
	FollowingCount int32
	Following      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserPreview struct {
	Username string
}

func (svc *Service) User(ctx context.Context, username string) (User, error) {
	var out User

	if !isUsername(username) {
		return out, ErrInvalidUsername
	}

	usr, _ := UserFromContext(ctx)

	out, err := svc.sqlSelectUser(ctx, sqlSelectUser{
		FollowerID: usr.ID,
		Username:   username,
	})
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
