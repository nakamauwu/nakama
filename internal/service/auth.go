package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// TokenLifespan until tokens are valid.
	TokenLifespan = time.Hour * 24 * 14
	// KeyAuthUserID to use in context.
	KeyAuthUserID key = "auth_user_id"
)

var (
	// ErrUnauthenticated used when there is no authenticated user in context.
	ErrUnauthenticated = errors.New("unauthenticated")
)

type key string

// LoginOutput response.
type LoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	AuthUser  User      `json:"authUser"`
}

// AuthUserID from token.
func (s *Service) AuthUserID(token string) (int64, error) {
	str, err := s.cdc.DecodeToString(token)
	if err != nil {
		return 0, fmt.Errorf("could not decode token: %v", err)
	}

	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("could not parse auth user id from token: %v", err)
	}

	return i, nil
}

// Login insecurely.
func (s *Service) Login(ctx context.Context, email string) (LoginOutput, error) {
	var out LoginOutput

	email = strings.TrimSpace(email)
	if !rxEmail.MatchString(email) {
		return out, ErrInvalidEmail
	}

	query := "SELECT id, username FROM users WHERE email = $1"
	err := s.db.QueryRowContext(ctx, query, email).Scan(&out.AuthUser.ID, &out.AuthUser.Username)

	if err == sql.ErrNoRows {
		return out, ErrUserNotFound
	}

	if err != nil {
		return out, fmt.Errorf("could not query select user: %v", err)
	}

	out.Token, err = s.cdc.EncodeToString(strconv.FormatInt(out.AuthUser.ID, 10))
	if err != nil {
		return out, fmt.Errorf("could not create token: %v", err)
	}

	out.ExpiresAt = time.Now().Add(TokenLifespan)

	return out, nil
}

// AuthUser from context.
// It requires the user ID in the context, so add it with a middleware or something.
func (s *Service) AuthUser(ctx context.Context) (User, error) {
	var u User
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return u, ErrUnauthenticated
	}

	query := "SELECT username FROM users WHERE id = $1"
	err := s.db.QueryRowContext(ctx, query, uid).Scan(&u.Username)
	if err == sql.ErrNoRows {
		return u, ErrUserNotFound
	}

	if err != nil {
		return u, fmt.Errorf("could not query select auth user: %v", err)
	}

	u.ID = uid
	return u, nil
}
