package nakama

import (
	"context"
	"fmt"

	"github.com/nakamauwu/nakama/db"
)

type sqlInsertCommentReaction struct {
	UserID    string
	CommentID string
	Reaction  string
}

type sqlSelectCommentReactionExistence sqlInsertCommentReaction

type sqlDeleteCommentReaction sqlInsertCommentReaction

func (svc *Service) sqlInsertCommentReaction(ctx context.Context, in sqlInsertCommentReaction) error {
	const query = `
		INSERT INTO comment_reactions (user_id, comment_id, reaction)
		VALUES ($1, $2, $3)
	`

	_, err := svc.DB.ExecContext(ctx, query, in.UserID, in.CommentID, in.Reaction)
	if db.IsPqForeignKeyViolationError(err, "comment_id") {
		return ErrCommentNotFound
	}

	if err != nil {
		return fmt.Errorf("sql insert comment reaction: %w", err)
	}

	return nil
}

func (svc *Service) sqlSelectCommentReactionExistence(ctx context.Context, in sqlSelectCommentReactionExistence) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM comment_reactions
			WHERE user_id = $1
				AND comment_id = $2
				AND reaction = $3
		)
	`

	var exists bool
	row := svc.DB.QueryRowContext(ctx, query, in.UserID, in.CommentID, in.Reaction)
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select comment reaction existence: %w", err)
	}

	return exists, nil
}

func (svc *Service) sqlDeleteCommentReaction(ctx context.Context, in sqlDeleteCommentReaction) error {
	const query = `
		DELETE FROM comment_reactions
		WHERE user_id = $1
			AND comment_id = $2
			AND reaction = $3
	`

	_, err := svc.DB.ExecContext(ctx, query, in.UserID, in.CommentID, in.Reaction)
	if err != nil {
		return fmt.Errorf("sql delete comment reaction: %w", err)
	}

	return nil
}
