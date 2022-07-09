package nakama

import (
	"context"
	"database/sql"
	"errors"
)

func (svc *Service) UserByUsername(ctx context.Context, username string) (User, error) {
	if !isUsername(username) {
		return User{}, ErrInvalidUsername
	}

	usr, err := svc.Queries.UserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}

	return usr, err
}
