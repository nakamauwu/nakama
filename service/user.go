package service

import (
	"context"

	"github.com/nakamauwu/nakama/types"
	"github.com/nicolasparada/go-errs"
)

const ErrUsernameRequired = errs.InvalidArgumentError("username required")

// Login can return ErrUsernameRequired.
func (svc *Service) Login(ctx context.Context, in types.Login) (types.User, error) {
	var out types.User

	exists, err := svc.Cockroach.UserExistsWithEmail(ctx, in.Email)
	if err != nil {
		return out, err
	}

	if exists {
		return svc.Cockroach.UserFromEmail(ctx, in.Email)
	}

	if in.Username == nil {
		return out, ErrUsernameRequired
	}

	return svc.Cockroach.CreateUser(ctx, types.CreateUser{
		Email:    in.Email,
		Username: *in.Username,
	})
}

func (svc *Service) User(ctx context.Context, userID string) (types.User, error) {
	return svc.Cockroach.User(ctx, userID)
}
