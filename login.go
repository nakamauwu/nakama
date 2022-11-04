package nakama

import (
	"context"
	"strings"
)

type UserIdentity struct {
	ID       string
	Email    string
	Username string
}

func (u User) Identity() UserIdentity {
	return UserIdentity{
		ID:       u.ID,
		Email:    u.Email,
		Username: u.Username,
	}
}

type LoginInput struct {
	Email    string
	Username *string
}

func (in *LoginInput) Prepare() {
	in.Email = strings.ToLower(in.Email)
}

func (in LoginInput) Validate() error {
	if !isEmail(in.Email) {
		return ErrInvalidEmail
	}

	if in.Username != nil && !isUsername(*in.Username) {
		return ErrInvalidUsername
	}

	return nil
}

// Login insecurely. Only for development purposes.
// TODO: add 2nd factor.
func (svc *Service) Login(ctx context.Context, in LoginInput) (UserIdentity, error) {
	var out UserIdentity

	in.Prepare()
	if err := in.Validate(); err != nil {
		return out, err
	}

	return out, svc.DB.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.sqlSelectUserExists(ctx, sqlSelectUserExists{Email: in.Email})
		if err != nil {
			return err
		}

		if exists {
			row, err := svc.sqlSelectUser(ctx, sqlSelectUser{Email: in.Email})
			if err != nil {
				return err
			}

			out.ID = row.ID
			out.Email = row.Email
			out.Username = row.Username
			return nil
		}

		if in.Username == nil {
			return ErrUserNotFound
		}

		exists, err = svc.sqlSelectUserExists(ctx, sqlSelectUserExists{Username: *in.Username})
		if err != nil {
			return err
		}

		if exists {
			return ErrUsernameTaken
		}

		userID := genID()
		_, err = svc.sqlInsertUser(ctx, sqlInsertUser{
			UserID:   userID,
			Email:    in.Email,
			Username: *in.Username,
		})
		if err != nil {
			return err
		}

		out.ID = userID
		out.Email = in.Email
		out.Username = *in.Username

		return nil
	})
}
