package nakama

import (
	"context"
	"fmt"

	"github.com/nicolasparada/go-db"
)

func (s *Store) CreatePostReaction(ctx context.Context, in PostReaction) error {
	const query = `
		INSERT INTO post_reactions (user_id, post_id, reaction)
		VALUES ($1, $2, $3)
	`

	_, err := s.db.Exec(ctx, query, in.userID, in.PostID, in.Reaction)
	if db.IsForeignKeyViolationError(err, "post_id") {
		return ErrPostNotFound
	}

	if err != nil {
		return fmt.Errorf("sql insert post reaction: %w", err)
	}

	return nil
}

func (s *Store) PostReactionExists(ctx context.Context, in PostReaction) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM post_reactions
			WHERE user_id = $1
				AND post_id = $2
				AND reaction = $3
		)
	`

	var exists bool
	row := s.db.QueryRow(ctx, query, in.userID, in.PostID, in.Reaction)
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql scan selected post reaction existence: %w", err)
	}

	return exists, nil
}

func (s *Store) DeletePostReaction(ctx context.Context, in PostReaction) error {
	const query = `
		DELETE FROM post_reactions
		WHERE user_id = $1
			AND post_id = $2
			AND reaction = $3
	`

	_, err := s.db.Exec(ctx, query, in.userID, in.PostID, in.Reaction)
	if err != nil {
		return fmt.Errorf("sql delete post reaction: %w", err)
	}

	return nil
}
