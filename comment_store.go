package nakama

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (db *Store) CreateComment(ctx context.Context, in CreateComment) (Created, error) {
	var out Created

	const createComment = `
		INSERT INTO comments (id, user_id, post_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`
	commentID := genID()
	err := db.QueryRow(ctx, createComment,
		commentID,
		in.userID,
		in.PostID,
		in.Content,
	).Scan(&out.CreatedAt)
	if isForeignKeyViolationError(err, "post_id") {
		return out, ErrPostNotFound
	}

	if err != nil {
		return out, fmt.Errorf("sql scan inserted comment: %w", err)
	}

	out.ID = commentID

	return out, nil
}

func (db *Store) Comments(ctx context.Context, in ListComments) ([]Comment, error) {
	const comments = `
		SELECT
			  comments.id
			, comments.user_id
			, comments.content
			, comments.reactions_count
			, comment_reactions.reactions
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
		WHERE comments.post_id = $2
		ORDER BY comments.id DESC
	`
	rows, err := db.Query(ctx, comments, in.authUserID, in.PostID)
	if err != nil {
		return nil, fmt.Errorf("sql select comments: %w", err)
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Comment, error) {
		var out Comment
		var reactions []string
		err := row.Scan(
			&out.ID,
			&out.UserID,
			&out.Content,
			&out.ReactionsCount,
			&reactions,
			&out.CreatedAt,
			&out.UpdatedAt,
			&out.User.Username,
			db.AvatarScanFunc(&out.User.AvatarPath),
			&out.User.AvatarWidth,
			&out.User.AvatarHeight,
		)
		if err != nil {
			return out, fmt.Errorf("sql scan comments: %w", err)
		}

		out.PostID = in.PostID
		out.ReactionsCount.Apply(reactions)

		return out, nil
	})
}

func (db *Store) Comment(ctx context.Context, in RetrieveComment) (Comment, error) {
	const query = `
		SELECT
			  comments.user_id
			, comments.post_id
			, comments.content
			, comments.reactions_count
			, comment_reactions.reactions
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
	err := db.QueryRow(ctx, query, in.authUserID, in.ID).Scan(
		&c.UserID,
		&c.PostID,
		&c.Content,
		&c.ReactionsCount,
		&reactions,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.User.Username,
		db.AvatarScanFunc(&c.User.AvatarPath),
		&c.User.AvatarWidth,
		&c.User.AvatarHeight,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Comment{}, ErrCommentNotFound
	}

	if err != nil {
		return Comment{}, fmt.Errorf("sql scan selected comment: %w", err)
	}

	c.ID = in.ID
	c.ReactionsCount.Apply(reactions)

	return c, nil
}

func (db *Store) CommentReactionsCount(ctx context.Context, commentID string) (ReactionsCount, error) {
	const query = `
		SELECT reactions_count FROM comments WHERE id = $1
	`

	var out ReactionsCount
	row := db.QueryRow(ctx, query, commentID)
	err := row.Scan(&out)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCommentNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("sql scan selected comment reactions count: %w", err)
	}

	return out, nil
}

func (db *Store) UpdateComment(ctx context.Context, in UpdateComment) (time.Time, error) {
	const query = `
		UPDATE comments
		SET reactions_count = COALESCE($1, reactions_count)
			, updated_at = now()
		WHERE id = $2
		RETURNING updated_at
	`
	var updatedAt time.Time
	row := db.QueryRow(ctx, query,
		in.ReactionsCount,
		in.ID,
	)
	err := row.Scan(&updatedAt)
	if err != nil {
		return updatedAt, fmt.Errorf("sql scan updated comment: %w", err)
	}

	return updatedAt, nil
}
