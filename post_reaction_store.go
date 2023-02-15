package nakama

import (
	"context"
	"fmt"
)

func (db *Store) CreatePostReaction(ctx context.Context, in PostReaction) error {
	const query = `
		INSERT INTO post_reactions (user_id, post_id, reaction)
		VALUES ($1, $2, $3)
	`

	_, err := db.ExecContext(ctx, query, in.userID, in.PostID, in.Reaction)
	if isPqForeignKeyViolationError(err, "post_id") {
		return ErrPostNotFound
	}

	if err != nil {
		return fmt.Errorf("sql insert post reaction: %w", err)
	}

	return nil
}

func (db *Store) PostReactionExists(ctx context.Context, in PostReaction) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM post_reactions
			WHERE user_id = $1
				AND post_id = $2
				AND reaction = $3
		)
	`

	var exists bool
	row := db.QueryRowContext(ctx, query, in.userID, in.PostID, in.Reaction)
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select post reaction existence: %w", err)
	}

	return exists, nil
}

func (db *Store) DeletePostReaction(ctx context.Context, in PostReaction) error {
	const query = `
		DELETE FROM post_reactions
		WHERE user_id = $1
			AND post_id = $2
			AND reaction = $3
	`

	_, err := db.ExecContext(ctx, query, in.userID, in.PostID, in.Reaction)
	if err != nil {
		return fmt.Errorf("sql delete post reaction: %w", err)
	}

	return nil
}
