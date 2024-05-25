package cockroach

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nakamauwu/nakama/types"
	"github.com/nicolasparada/go-errs"
)

func (c *Cockroach) UserExistsWithEmail(ctx context.Context, email string) (bool, error) {
	var out bool

	const q = `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE email = LOWER(@email)
		)
	`

	row := c.db.QueryRow(ctx, q, pgx.NamedArgs{
		"email": email,
	})
	err := row.Scan(&out)
	if err != nil {
		return out, fmt.Errorf("sql query user exists with email: %w", err)
	}

	return out, nil
}

func (c *Cockroach) UserFromEmail(ctx context.Context, email string) (types.User, error) {
	var out types.User

	const q = `
		SELECT id, email, username, avatar, created_at, updated_at
		FROM users
		WHERE email = LOWER(@email)
	`

	row := c.db.QueryRow(ctx, q, pgx.NamedArgs{
		"email": email,
	})
	err := row.Scan(&out.ID, &out.Email, &out.Username, &out.Avatar, &out.CreatedAt, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, errs.NotFoundError("user not found")
	}

	if err != nil {
		return out, fmt.Errorf("sql query user from email: %w", err)
	}

	return out, nil
}

func (c *Cockroach) CreateUser(ctx context.Context, in types.CreateUser) (types.User, error) {
	var out types.User

	const q = `
		INSERT INTO users (email, username, avatar)
		VALUES (LOWER(@email), @username, @avatar)
		RETURNING id, email, username, avatar, created_at, updated_at
	`

	row := c.db.QueryRow(ctx, q, pgx.NamedArgs{
		"email":    in.Email,
		"username": in.Username,
		"avatar":   in.Avatar,
	})
	err := row.Scan(&out.ID, &out.Email, &out.Username, &out.Avatar, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return out, fmt.Errorf("sql insert user: %w", err)
	}

	return out, nil
}

func (c *Cockroach) User(ctx context.Context, userID string) (types.User, error) {
	var out types.User

	const q = `
		SELECT id, email, username, avatar, created_at, updated_at
		FROM users
		WHERE id = @id
	`

	row := c.db.QueryRow(ctx, q, pgx.NamedArgs{
		"id": userID,
	})
	err := row.Scan(&out.ID, &out.Email, &out.Username, &out.Avatar, &out.CreatedAt, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, errs.NotFoundError("user not found")
	}

	if err != nil {
		return out, fmt.Errorf("sql query user: %w", err)
	}

	return out, nil
}
