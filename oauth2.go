package nakama

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach-go/crdb"
)

type ProvidedUser struct {
	ID       string
	Email    string
	Username *string
}

func (svc *Service) LoginFromProvider(ctx context.Context, name string, providedUser ProvidedUser) (User, error) {
	var u User

	providedUser.Email = strings.ToLower(providedUser.Email)
	if !reEmail.MatchString(providedUser.Email) {
		return u, ErrInvalidEmail
	}

	if providedUser.Username != nil && !ValidUsername(*providedUser.Username) {
		return u, ErrInvalidUsername
	}

	err := crdb.ExecuteTx(ctx, svc.DB, nil, func(tx *sql.Tx) error {
		var existsWithProviderID bool

		query := fmt.Sprintf(`
			SELECT EXISTS (
				SELECT 1 FROM users WHERE %s_provider_id = $1
			)
		`, name)
		row := tx.QueryRowContext(ctx, query, providedUser.ID)
		err := row.Scan(&existsWithProviderID)
		if err != nil {
			return fmt.Errorf("could not sql query user existence with provider id: %w", err)
		}

		if !existsWithProviderID {
			var existsWithEmail bool

			query := `
				SELECT EXISTS (
					SELECT 1 FROM users WHERE email = $1
				)
			`
			row := tx.QueryRowContext(ctx, query, providedUser.Email)
			err := row.Scan(&existsWithEmail)
			if err != nil {
				return fmt.Errorf("could not sql query user existence with provider email: %w", err)
			}

			if !existsWithEmail {
				if providedUser.Username == nil {
					return ErrUserNotFound
				}

				query = fmt.Sprintf(`INSERT INTO users (email, username, %s_provider_id) VALUES ($1, $2, $3) RETURNING id`, name)
				row = tx.QueryRowContext(ctx, query, providedUser.Email, *providedUser.Username, providedUser.ID)
				err = row.Scan(&u.ID)
				if isUniqueViolation(err) && strings.Contains(err.Error(), "username") {
					return ErrUsernameTaken
				}

				if err != nil {
					return fmt.Errorf("could not sql insert provided user: %w", err)
				}

				u.Username = *providedUser.Username
				return nil
			}

			query = fmt.Sprintf(`UPDATE users SET %s_provider_id = $1 WHERE email = $2`, name)
			_, err = tx.ExecContext(ctx, query, providedUser.ID, providedUser.Email)
			if err != nil {
				return fmt.Errorf("could not sql update user with provider id: %w", err)
			}
		}

		var avatar sql.NullString
		query = fmt.Sprintf(`SELECT id, username, avatar FROM users WHERE %s_provider_id = $1`, name)
		row = tx.QueryRowContext(ctx, query, providedUser.ID)
		err = row.Scan(&u.ID, &u.Username, &avatar)
		if err != nil {
			return fmt.Errorf("could not sql query user by provider id: %w", err)
		}

		u.AvatarURL = svc.avatarURL(avatar)

		return nil
	})

	return u, err
}
