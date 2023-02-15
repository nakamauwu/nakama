package nakama

import (
	"context"
	"fmt"
)

func (db *Store) CreateCommentReaction(ctx context.Context, in CommentReaction) error {
	const query = `
		INSERT INTO comment_reactions (user_id, comment_id, reaction)
		VALUES ($1, $2, $3)
	`

	_, err := db.ExecContext(ctx, query, in.userID, in.CommentID, in.Reaction)
	if isPqForeignKeyViolationError(err, "comment_id") {
		return ErrCommentNotFound
	}

	if err != nil {
		return fmt.Errorf("sql insert comment reaction: %w", err)
	}

	return nil
}

func (db *Store) CommentReactionExists(ctx context.Context, in CommentReaction) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM comment_reactions
			WHERE user_id = $1
				AND comment_id = $2
				AND reaction = $3
		)
	`

	var exists bool
	row := db.QueryRowContext(ctx, query, in.userID, in.CommentID, in.Reaction)
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select comment reaction existence: %w", err)
	}

	return exists, nil
}

func (db *Store) DeleteCommentReaction(ctx context.Context, in CommentReaction) error {
	const query = `
		DELETE FROM comment_reactions
		WHERE user_id = $1
			AND comment_id = $2
			AND reaction = $3
	`

	_, err := db.ExecContext(ctx, query, in.userID, in.CommentID, in.Reaction)
	if err != nil {
		return fmt.Errorf("sql delete comment reaction: %w", err)
	}

	return nil
}
