package nakama

import (
	"context"
	"fmt"
	"time"
)

type sqlInsertUser struct {
	UserID   string
	Email    string
	Username string
}

type sqlUpdateUser struct {
	IncreasePostsCountBy     int
	IncreaseFollowersCountBy int
	IncreaseFollowingCountBy int
	UserID                   string
}

type sqlSelectUser struct {
	FollowerID string
	UserID     string
	Email      string
	Username   string
}

type sqlSelectUserExists struct {
	UserID   string
	Email    string
	Username string
}

func (svc *Service) sqlInsertUser(ctx context.Context, in sqlInsertUser) (time.Time, error) {
	const query = `
		INSERT INTO users (id, email, username)
		VALUES ($1, LOWER($2), $3)
		RETURNING created_at
	`
	var createdAt time.Time
	err := svc.DB.QueryRowContext(ctx, query, in.UserID, in.Email, in.Username).Scan(&createdAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql insert user: %w", err)
	}

	return createdAt, nil
}

func (svc *Service) sqlUpdateUser(ctx context.Context, in sqlUpdateUser) (time.Time, error) {
	const query = `
		UPDATE users SET
			posts_count = posts_count + $1,
			followers_count = followers_count + $2,
			following_count = following_count + $3,
			updated_at = now()
		WHERE id = $4
		RETURNING updated_at
	`
	var updatedAt time.Time
	err := svc.DB.QueryRowContext(ctx, query,
		in.IncreasePostsCountBy,
		in.IncreaseFollowersCountBy,
		in.IncreaseFollowingCountBy,
		in.UserID,
	).Scan(&updatedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql update user: %w", err)
	}

	return updatedAt, nil
}

func (svc *Service) sqlSelectUser(ctx context.Context, in sqlSelectUser) (User, error) {
	const query = `
		SELECT users.id, users.email, users.username, users.posts_count, users.followers_count, users.following_count, users.created_at, users.updated_at,
		(
			CASE
				WHEN $1::varchar != '' THEN (
					SELECT EXISTS (
						SELECT 1 FROM user_follows
						WHERE follower_id = $1::varchar
						AND followed_id = users.id
					)
				)
				ELSE false
			END
		) AS following
		FROM users
		WHERE CASE
			WHEN $2::varchar != '' THEN users.id = $2::varchar
			WHEN $3::varchar != '' THEN users.email = LOWER($3::varchar)
			WHEN $4::varchar != '' THEN LOWER(users.username) = LOWER($4::varchar)
			ELSE false
		END
	`
	var usr User
	err := svc.DB.QueryRowContext(ctx, query,
		in.FollowerID,
		in.UserID,
		in.Email,
		in.Username,
	).Scan(
		&usr.ID,
		&usr.Email,
		&usr.Username,
		&usr.PostsCount,
		&usr.FollowersCount,
		&usr.FollowingCount,
		&usr.CreatedAt,
		&usr.UpdatedAt,
		&usr.Following,
	)
	if err != nil {
		return usr, fmt.Errorf("sql select user: %w", err)
	}

	return usr, nil
}

func (svc *Service) sqlSelectUserExists(ctx context.Context, in sqlSelectUserExists) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE CASE
				WHEN $1::varchar != '' THEN id = $1::varchar
				WHEN $2::varchar != '' THEN email = LOWER($2::varchar)
				WHEN $3::varchar != '' THEN LOWER(username) = LOWER($3::varchar)
				ELSE false
			END
		)
	`
	var exists bool
	err := svc.DB.QueryRowContext(ctx, query, in.UserID, in.Email, in.Username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select user exists: %w", err)
	}

	return exists, nil
}
