package nakama

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach-go/crdb"
)

func (svc *Service) EnsureUser(ctx context.Context, email string, username *string) (User, error) {
	var u User

	if !reEmail.MatchString(email) {
		return u, ErrInvalidEmail
	}

	if username != nil && !ValidUsername(*username) {
		return u, ErrInvalidUsername
	}

	err := crdb.ExecuteTx(ctx, svc.DB, nil, func(tx *sql.Tx) error {
		var exists bool

		query := `
			SELECT EXISTS (
				SELECT 1 FROM users WHERE email = $1
			)
		`
		row := tx.QueryRowContext(ctx, query, email)
		err := row.Scan(&exists)
		if err != nil {
			return fmt.Errorf("could not sql query user existence by email: %w", err)
		}

		if exists {
			var avatar sql.NullString
			query := `SELECT id, username, avatar FROM users WHERE email = $1`
			row := tx.QueryRowContext(ctx, query, email)
			err := row.Scan(&u.ID, &u.Username, &avatar)
			if err != nil {
				return fmt.Errorf("could not sql query user by email: %w", err)
			}

			u.AvatarURL = svc.avatarURL(avatar)

			return nil
		}

		if username == nil {
			return ErrUserNotFound
		}

		query = `INSERT INTO users (email, username) VALUES ($1, $2) RETURNING id`
		row = tx.QueryRowContext(ctx, query, email, *username)
		err = row.Scan(&u.ID)
		if isUniqueViolation(err) && strings.Contains(err.Error(), "username") {
			return ErrUsernameTaken
		}

		if err != nil {
			return fmt.Errorf("could not sql insert user: %w", err)
		}

		u.Username = *username

		return nil
	})

	return u, err
}
