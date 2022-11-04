package nakama

import (
	"context"
	"fmt"
	"time"

	"github.com/nakamauwu/nakama/db"
)

type sqlInsertTimelineItem struct {
	UserID string
	PostID string
}

type sqlInsertPost struct {
	PostID  string
	UserID  string
	Content string
}

type sqlInsertTimeline struct {
	PostsID    string
	FollowedID string
}

type sqlUpdatePost struct {
	IncreaseCommentsCountBy int32
	PostID                  string
}

type sqlInsertedTimelineItem struct {
	ID     string
	UserID string
}

func (svc *Service) sqlInsertTimelineItem(ctx context.Context, in sqlInsertTimelineItem) (string, error) {
	const query = `
		INSERT INTO timeline (user_id, post_id)
		VALUES ($1, $2)
		RETURNING id
	`
	var id string
	err := svc.DB.QueryRowContext(ctx, query, in.UserID, in.PostID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("sql insert timeline item: %w", err)
	}

	return id, nil
}

func (svc *Service) sqlInsertPost(ctx context.Context, in sqlInsertPost) (time.Time, error) {
	const query = `
		INSERT INTO posts (id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING created_at
	`
	var createdAt time.Time
	err := svc.DB.QueryRowContext(ctx, query, in.PostID, in.UserID, in.Content).Scan(&createdAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql insert post: %w", err)
	}

	return createdAt, nil
}

func (svc *Service) sqlInsertTimeline(ctx context.Context, in sqlInsertTimeline) ([]sqlInsertedTimelineItem, error) {
	const query = `
		INSERT INTO timeline (user_id, post_id)
		SELECT user_follows.follower_id, $1
		FROM user_follows
		WHERE user_follows.followed_id = $2
		ON CONFLICT (user_id, post_id) DO NOTHING
		RETURNING id, user_id
	`

	rows, err := svc.DB.QueryContext(ctx, query, in.PostsID, in.FollowedID)
	if err != nil {
		return nil, fmt.Errorf("sql fanout timeline: %w", err)
	}

	return db.Collect(rows, func(scanner db.Scanner) (sqlInsertedTimelineItem, error) {
		var out sqlInsertedTimelineItem
		return out, scanner.Scan(&out.ID, &out.UserID)
	})
}

func (svc *Service) sqlSelectTimeline(ctx context.Context, userID string) ([]Post, error) {
	const query = `
		SELECT posts.id, posts.user_id, posts.content, posts.comments_count, posts.created_at, posts.updated_at, users.username
		FROM timeline
		INNER JOIN posts ON timeline.post_id = posts.id
		INNER JOIN users ON posts.user_id = users.id
		WHERE timeline.user_id = $1
		ORDER BY timeline.id DESC
	`
	rows, err := svc.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("sql select timeline: %w", err)
	}

	return db.Collect(rows, func(scanner db.Scanner) (Post, error) {
		var out Post
		return out, scanner.Scan(
			&out.ID,
			&out.UserID,
			&out.Content,
			&out.CommentsCount,
			&out.CreatedAt,
			&out.UpdatedAt,
			&out.User.Username,
		)
	})
}

func (svc *Service) sqlSelectPost(ctx context.Context, postID string) (Post, error) {
	const post = `-- name: Post :one
		SELECT posts.id, posts.user_id, posts.content, posts.comments_count, posts.created_at, posts.updated_at, users.username
		FROM posts
		INNER JOIN users ON posts.user_id = users.id
		WHERE posts.id = $1
	`
	var p Post
	err := svc.DB.QueryRowContext(ctx, post, postID).Scan(
		&p.ID,
		&p.UserID,
		&p.Content,
		&p.CommentsCount,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.User.Username,
	)
	if err != nil {
		return Post{}, fmt.Errorf("sql select post: %w", err)
	}

	return p, err
}

func (svc *Service) sqlSelectPosts(ctx context.Context, username string) ([]Post, error) {
	const query = `
		SELECT posts.id, posts.user_id, posts.content, posts.comments_count, posts.created_at, posts.updated_at, users.username
		FROM posts
		INNER JOIN users ON posts.user_id = users.id
		WHERE
			CASE
				WHEN $1::varchar != '' THEN LOWER(users.username) = LOWER($1::varchar)
				ELSE true
			END
		ORDER BY posts.id DESC
	`
	rows, err := svc.DB.QueryContext(ctx, query, username)
	if err != nil {
		return nil, fmt.Errorf("sql select posts: %w", err)
	}

	return db.Collect(rows, func(scanner db.Scanner) (Post, error) {
		var out Post
		return out, scanner.Scan(
			&out.ID,
			&out.UserID,
			&out.Content,
			&out.CommentsCount,
			&out.CreatedAt,
			&out.UpdatedAt,
			&out.User.Username,
		)
	})
}

func (svc *Service) sqlUpdatePost(ctx context.Context, in sqlUpdatePost) (time.Time, error) {
	const query = `
		UPDATE posts
		SET comments_count = comments_count + $1, updated_at = now()
		WHERE id = $2
		RETURNING updated_at
	`
	var updatedAt time.Time
	err := svc.DB.QueryRowContext(ctx, query, in.IncreaseCommentsCountBy, in.PostID).Scan(&updatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql update post: %w", err)
	}

	return updatedAt, nil
}
