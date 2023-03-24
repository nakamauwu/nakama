package nakama

import (
	"context"
	"fmt"

	"github.com/nicolasparada/go-db"
)

func (s *Store) CreateCommentReaction(ctx context.Context, in CommentReaction) error {
	const query = `
		INSERT INTO comment_reactions (user_id, comment_id, reaction)
		VALUES ($1, $2, $3)
	`

	_, err := s.db.Exec(ctx, query, in.userID, in.CommentID, in.Reaction)
	if db.IsForeignKeyViolationError(err, "comment_id") {
		return ErrCommentNotFound
	}

	if err != nil {
		return fmt.Errorf("sql insert comment reaction: %w", err)
	}

	return nil
}

func (s *Store) CommentReactionExists(ctx context.Context, in CommentReaction) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM comment_reactions
			WHERE user_id = $1
				AND comment_id = $2
				AND reaction = $3
		)
	`

	var exists bool
	row := s.db.QueryRow(ctx, query, in.userID, in.CommentID, in.Reaction)
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql scan selected comment reaction existence: %w", err)
	}

	return exists, nil
}

func (s *Store) DeleteCommentReaction(ctx context.Context, in CommentReaction) error {
	const query = `
		DELETE FROM comment_reactions
		WHERE user_id = $1
			AND comment_id = $2
			AND reaction = $3
	`

	_, err := s.db.Exec(ctx, query, in.userID, in.CommentID, in.Reaction)
	if err != nil {
		return fmt.Errorf("sql delete comment reaction: %w", err)
	}

	return nil
}
