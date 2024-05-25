package cockroach

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nakamauwu/nakama/types"
)

func (c *Cockroach) CreatePost(ctx context.Context, in types.CreatePost) (types.Created, error) {
	var out types.Created

	const q = `
		INSERT INTO posts (user_id, content)
		VALUES (@user_id, @content)
		RETURNING id, created_at
	`

	row := c.db.QueryRow(ctx, q, pgx.NamedArgs{
		"user_id": in.UserID,
		"content": in.Content,
	})
	err := row.Scan(&out.ID, &out.CreatedAt)
	if err != nil {
		return out, fmt.Errorf("sql insert post: %w", err)
	}

	return out, nil
}

func (c *Cockroach) Posts(ctx context.Context, in types.ListPosts) (types.List[types.Post], error) {
	var out types.List[types.Post]

	const q = `
		SELECT *
		FROM posts
		WHERE user_id = @user_id
		ORDER BY id DESC
	`

	rows, err := c.db.Query(ctx, q, pgx.NamedArgs{
		"user_id": in.UserID,
	})
	if err != nil {
		return out, fmt.Errorf("sql select posts: %w", err)
	}

	out.Items, err = pgx.CollectRows(rows, pgx.RowToStructByName[types.Post])
	if err != nil {
		return out, fmt.Errorf("sql collect posts: %w", err)
	}

	return out, nil
}
