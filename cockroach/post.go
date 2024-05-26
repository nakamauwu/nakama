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
		SELECT posts.*, to_jsonb(users.*) AS user
		FROM posts
		INNER JOIN users ON posts.user_id = users.id
		WHERE posts.user_id = @user_id
			AND @before::varchar IS NULL OR posts.id < @before::varchar
		ORDER BY posts.id DESC
		LIMIT @last::integer
	`

	rows, err := c.db.Query(ctx, q, pgx.NamedArgs{
		"user_id": in.UserID,
		"before":  in.Before,
		"last":    in.Last,
	})
	if err != nil {
		return out, fmt.Errorf("sql select posts: %w", err)
	}

	out.Items, err = pgx.CollectRows(rows, pgx.RowToStructByName[types.Post])
	if err != nil {
		return out, fmt.Errorf("sql collect posts: %w", err)
	}

	if l := len(out.Items); l != 0 {
		out.LastID = ptr(out.Items[l-1].ID)
		out.HasNextPage = in.Last != nil && *in.Last != 0 && len(out.Items) == int(*in.Last)
	}

	return out, nil
}
