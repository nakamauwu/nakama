package nakama

import (
	"context"
	"fmt"
	"time"

	"github.com/nakamauwu/nakama/db"
)

type sqlInsertComment struct {
	CommentID string
	UserID    string
	PostID    string
	Content   string
}

func (svc *Service) sqlSelectComments(ctx context.Context, postID string) ([]Comment, error) {
	const comments = `
		SELECT comments.id, comments.user_id, comments.post_id, comments.content, comments.created_at, comments.updated_at, users.username
		FROM comments
		INNER JOIN users ON comments.user_id = users.id
		WHERE comments.post_id = $1
		ORDER BY comments.id DESC
	`
	rows, err := svc.DB.QueryContext(ctx, comments, postID)
	if err != nil {
		return nil, fmt.Errorf("sql select comments: %w", err)
	}

	return db.Collect(rows, func(scanner db.Scanner) (Comment, error) {
		var out Comment
		return out, scanner.Scan(
			&out.ID,
			&out.UserID,
			&out.PostID,
			&out.Content,
			&out.CreatedAt,
			&out.UpdatedAt,
			&out.User.Username,
		)
	})
}

func (svc *Service) sqlInsertComment(ctx context.Context, in sqlInsertComment) (time.Time, error) {
	const createComment = `-- name: CreateComment :one
		INSERT INTO comments (id, user_id, post_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`
	var createdAt time.Time
	err := svc.DB.QueryRowContext(ctx, createComment,
		in.CommentID,
		in.UserID,
		in.PostID,
		in.Content,
	).Scan(&createdAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql insert comment: %w", err)
	}

	return createdAt, nil
}
