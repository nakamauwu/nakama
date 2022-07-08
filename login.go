package nakama

import (
	"context"
	"regexp"
	"strings"

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
func (svc *Service) Login(ctx context.Context, in LoginInput) (User, error) {
	var out User

	in.Prepare()
	if err := in.Validate(); err != nil {
		return out, err
	}

	// TODO: run inside a transaction.
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

func isEmail(s string) bool {
	return reEmail.MatchString(s)
}

func isUsername(s string) bool {
	return reUsername.MatchString(s)
}
