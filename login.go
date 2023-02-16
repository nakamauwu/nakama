package nakama

import (
	"context"
	"strings"
)

type UserIdentity struct {
	ID           string
	Email        string
	Username     string
	AvatarPath   *string
	AvatarWidth  *uint
	AvatarHeight *uint
}

func (u User) Identity() UserIdentity {
	return UserIdentity{
		ID:           u.ID,
		Email:        u.Email,
		Username:     u.Username,
		AvatarPath:   u.AvatarPath,
		AvatarWidth:  u.AvatarWidth,
		AvatarHeight: u.AvatarHeight,
	}
}

type Login struct {
	Email    string
	Username *string
}

func (in *Login) Validate() error {
	in.Email = strings.TrimSpace(in.Email)
	in.Email = strings.ToLower(in.Email)
	if in.Username != nil {
		*in.Username = strings.TrimSpace(*in.Username)
	}

	if !validEmail(in.Email) {
		return ErrInvalidEmail
	}

	if in.Username != nil && !validUsername(*in.Username) {
		return ErrInvalidUsername
	}

	return nil
}

// Login insecurely. Only for development purposes.
// TODO: add 2nd factor.
func (svc *Service) Login(ctx context.Context, in Login) (UserIdentity, error) {
	var out UserIdentity

	if err := in.Validate(); err != nil {
		return out, err
	}

	return out, svc.Store.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.Store.UserExists(ctx, RetrieveUserExists{Email: in.Email})
		if err != nil {
			return err
		}

		if exists {
			usr, err := svc.Store.User(ctx, RetrieveUser{email: in.Email})
			if err != nil {
				return err
			}

			out.ID = usr.ID
			out.Email = usr.Email
			out.Username = usr.Username
			out.AvatarPath = usr.AvatarPath
			out.AvatarWidth = usr.AvatarWidth
			out.AvatarHeight = usr.AvatarHeight

			return nil
		}

		if in.Username == nil {
			return ErrUserNotFound
		}

		exists, err = svc.Store.UserExists(ctx, RetrieveUserExists{Username: *in.Username})
		if err != nil {
			return err
		}

		if exists {
			return ErrUsernameTaken
		}

		inserted, err := svc.Store.CreateUser(ctx, CreateUser{
			Email:    in.Email,
			Username: *in.Username,
		})
		if err != nil {
			return err
		}

		out.ID = inserted.ID
		out.Email = in.Email
		out.Username = *in.Username

		return nil
	})
}
