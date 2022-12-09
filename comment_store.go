package nakama

import (
	"context"
	"fmt"

	"github.com/nakamauwu/nakama/db"
)

type sqlInsertComment struct {
	UserID  string
	PostID  string
	Content string
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
