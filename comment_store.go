package nakama

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/nakamauwu/nakama/db"
)

type sqlInsertComment struct {
	UserID  string
	PostID  string
	Content string
}

type sqlSelectComment struct {
	CommentID  string
	AuthUserID string
}

type sqlUpdateComment struct {
	CommentID      string
	ReactionsCount *ReactionsCount
}

func (svc *Service) sqlInsertComment(ctx context.Context, in sqlInsertComment) (CreatedComment, error) {
	var out CreatedComment

	const createComment = `
		INSERT INTO comments (id, user_id, post_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`
	commentID := genID()
	err := svc.DB.QueryRowContext(ctx, createComment,
		commentID,
		in.UserID,
		in.PostID,
		in.Content,
	).Scan(&out.CreatedAt)
	if db.IsPqForeignKeyViolationError(err, "post_id") {
		return out, ErrPostNotFound
	}

	if err != nil {
		return out, fmt.Errorf("sql insert comment: %w", err)
	}

	out.ID = commentID

	return out, nil
}

func (svc *Service) sqlSelectComments(ctx context.Context, postID string) ([]Comment, error) {
	const comments = `
		SELECT
			  comments.id
			, comments.user_id
			, comments.post_id
			, comments.content
			, comments.created_at
			, comments.updated_at
			, users.username
			, users.avatar_path
			, users.avatar_width
			, users.avatar_height
		FROM comments
		INNER JOIN users ON comments.user_id = users.id
		WHERE comments.post_id = $1
		ORDER BY comments.id DESC
	`
	rows, err := svc.DB.QueryContext(ctx, comments, postID)
	if err != nil {
		return nil, fmt.Errorf("sql select comments: %w", err)
	}

	return db.Collect(rows, func(scan db.ScanFunc) (Comment, error) {
		var out Comment
		return out, scan(
			&out.ID,
			&out.UserID,
			&out.PostID,
			&out.Content,
			&out.CreatedAt,
			&out.UpdatedAt,
			&out.User.Username,
			svc.sqlScanAvatar(&out.User.AvatarPath),
			&out.User.AvatarWidth,
			&out.User.AvatarHeight,
		)
	})
}

func (svc *Service) sqlSelectComment(ctx context.Context, in sqlSelectComment) (Comment, error) {
	const query = `
		SELECT
			  comments.id
			, comments.user_id
			, comments.post_id
			, comments.content
			, comments.reactions_count
			, reactions
			, comments.created_at
			, comments.updated_at
			, users.username
			, users.avatar_path
			, users.avatar_width
			, users.avatar_height
		FROM comments
		INNER JOIN users ON comments.user_id = users.id
		LEFT JOIN (
			SELECT comment_id, array_agg(reaction) AS reactions
			FROM comment_reactions
			WHERE comment_reactions.user_id = $1
			GROUP BY comment_id
		) AS comment_reactions ON comment_reactions.comment_id = comments.id
		WHERE comments.id = $2
	`
	var c Comment
	var reactions []string
	err := svc.DB.QueryRowContext(ctx, query, in.AuthUserID, in.CommentID).Scan(
		&c.ID,
		&c.UserID,
		&c.PostID,
		&c.Content,
		&db.JSONValue{Dst: &c.ReactionsCount},
		pq.Array(&reactions),
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.User.Username,
		svc.sqlScanAvatar(&c.User.AvatarPath),
		&c.User.AvatarWidth,
		&c.User.AvatarHeight,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, ErrCommentNotFound
	}

	if err != nil {
		return Comment{}, fmt.Errorf("sql select comment: %w", err)
	}

	c.ReactionsCount.Apply(reactions)

	return c, nil
}

func (svc *Service) sqlSelectCommentReactionsCount(ctx context.Context, commentID string) (ReactionsCount, error) {
	const query = `
		SELECT reactions_count FROM comments WHERE id = $1
	`

	var out ReactionsCount
	row := svc.DB.QueryRowContext(ctx, query, commentID)
	err := row.Scan(&db.JSONValue{Dst: &out})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommentNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("sql select comment reactions count: %w", err)
	}

	return out, nil
}

func (svc *Service) sqlUpdateComment(ctx context.Context, in sqlUpdateComment) (time.Time, error) {
	const query = `
		UPDATE comments
		SET reactions_count = COALESCE($1, reactions_count)
			, updated_at = now()
		WHERE id = $2
		RETURNING updated_at
	`
	var updatedAt time.Time
	row := svc.DB.QueryRowContext(ctx, query,
		db.JSONValue{Dst: in.ReactionsCount},
		in.CommentID,
	)
	err := row.Scan(&updatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql update comment: %w", err)
	}

	return updatedAt, nil
}
