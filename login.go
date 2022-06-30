package nakama

import (
	"context"
	"errors"

	"github.com/rs/xid"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username taken")
)

type LoginInput struct {
	Email    string
	Username *string
}

func (svc *Service) Login(ctx context.Context, in LoginInput) (User, error) {
	var out User

	exists, err := svc.Queries.UserExistsByEmail(ctx, in.Email)
	if err != nil {
		return out, err
	}

	if exists {
		return svc.Queries.UserByEmail(ctx, in.Email)
	}

	if in.Username == nil {
		return out, ErrUserNotFound
	}

	exists, err = svc.Queries.UserExistsByUsername(ctx, *in.Username)
	if err != nil {
		return out, err
	}

	if exists {
		return out, ErrUsernameTaken
	}

	userID := genID()
	createdAt, err := svc.Queries.CreateUser(ctx, CreateUserParams{
		UserID:   userID,
		Email:    in.Email,
		Username: *in.Username,
	})
	if err != nil {
		return out, err
	}

	return User{
		ID:        userID,
		Email:     in.Email,
		Username:  *in.Username,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, nil
}

func genID() string {
	return xid.New().String()
}
