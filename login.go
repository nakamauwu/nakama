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

	// TODO: run inside a transaction.
	exists, err := svc.sqlSelectUserExists(ctx, sqlSelectUserExists{Email: in.Email})
	if err != nil {
		return out, err
	}

	if exists {
		row, err := svc.sqlSelectUser(ctx, sqlSelectUser{Email: in.Email})
		if err != nil {
			return out, err
		}

		return UserIdentity{
			ID:       row.ID,
			Email:    row.Email,
			Username: row.Username,
		}, nil
	}

	if in.Username == nil {
		return out, ErrUserNotFound
	}

	exists, err = svc.sqlSelectUserExists(ctx, sqlSelectUserExists{Username: *in.Username})
	if err != nil {
		return out, err
	}

	if exists {
		return out, ErrUsernameTaken
	}

	userID := genID()
	_, err = svc.sqlInsertUser(ctx, sqlInsertUser{
		UserID:   userID,
		Email:    in.Email,
		Username: *in.Username,
	})
	if err != nil {
		return out, err
	}

	return UserIdentity{
		ID:       userID,
		Email:    in.Email,
		Username: *in.Username,
	}, nil
}
