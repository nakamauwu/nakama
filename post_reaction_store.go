package nakama

import (
	"context"
	"fmt"

	"github.com/nakamauwu/nakama/db"
)

type sqlInsertPostReaction struct {
	UserID   string
	PostID   string
	Reaction string
}

type sqlSelectPostReactionExistence sqlInsertPostReaction

type sqlUpdatePostReactions struct {
	PostID         string
	ReactionsCount ReactionsCount
}

func (svc *Service) sqlInsertPostReaction(ctx context.Context, in sqlInsertPostReaction) error {
	const query = `
		INSERT INTO post_reactions (user_id, post_id, reaction)
		VALUES ($1, $2, $3)
	`

	_, err := svc.DB.ExecContext(ctx, query, in.UserID, in.PostID, in.Reaction)
	if db.IsPqForeignKeyViolationError(err, "post_id") {
		return ErrPostNotFound
	}

	if err != nil {
		return fmt.Errorf("sql insert post reaction: %w", err)
	}

	return nil
}

func (svc *Service) sqlSelectPostReactionExistence(ctx context.Context, in sqlSelectPostReactionExistence) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM post_reactions
			WHERE user_id = $1
				AND post_id = $2
				AND reaction = $3
		)
	`

	var exists bool
	row := svc.DB.QueryRowContext(ctx, query, in.UserID, in.PostID, in.Reaction)
	err := row.Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select post reaction existence: %w", err)
	}

	return exists, nil
}
