package nakama

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type sqlInsertUser struct {
	Email    string
	Username string
}

type sqlUpdateUser struct {
	Username                 *string
	AvatarPath               *string
	AvatarWidth              *uint
	AvatarHeight             *uint
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

type sqlInsertedUser struct {
	ID        string
	CreatedAt time.Time
}

func (svc *Service) sqlInsertUser(ctx context.Context, in sqlInsertUser) (sqlInsertedUser, error) {
	var out sqlInsertedUser

	const query = `
		INSERT INTO users (id, email, username)
		VALUES ($1, LOWER($2), $3)
		RETURNING created_at
	`
	userID := genID()
	err := svc.DB.QueryRowContext(ctx, query, userID, in.Email, in.Username).Scan(&out.CreatedAt)
	if err != nil {
		return out, fmt.Errorf("sql insert user: %w", err)
	}

	out.ID = userID

	return out, nil
}

func (svc *Service) sqlSelectUser(ctx context.Context, in sqlSelectUser) (User, error) {
	const query = `
		SELECT users.id
			, users.email
			, users.username
			, users.avatar_path
			, users.avatar_width
			, users.avatar_height
			, users.posts_count
			, users.followers_count
			, users.following_count
			, users.created_at
			, users.updated_at
			, (
				CASE
					WHEN $1::varchar != '' AND $1 != users.id THEN (
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
		svc.sqlScanAvatar(&usr.AvatarPath),
		&usr.AvatarWidth,
		&usr.AvatarHeight,
		&usr.PostsCount,
		&usr.FollowersCount,
		&usr.FollowingCount,
		&usr.CreatedAt,
		&usr.UpdatedAt,
		&usr.Following,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return usr, ErrUserNotFound
	}
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

func (svc *Service) sqlUpdateUser(ctx context.Context, in sqlUpdateUser) (time.Time, error) {
	const query = `
		UPDATE users SET
			   username = COALESCE($1, username)
			,  avatar_path = COALESCE($2, avatar_path)
			, avatar_width = COALESCE($3, avatar_width)
			, avatar_height = COALESCE($4, avatar_height)
			, posts_count = posts_count + $5
			, followers_count = followers_count + $6
			, following_count = following_count + $7
			, updated_at = now()
		WHERE id = $8
		RETURNING updated_at
	`
	var updatedAt time.Time
	err := svc.DB.QueryRowContext(ctx, query,
		in.Username,
		in.AvatarPath,
		in.AvatarWidth,
		in.AvatarHeight,
		in.IncreasePostsCountBy,
		in.IncreaseFollowersCountBy,
		in.IncreaseFollowingCountBy,
		in.UserID,
	).Scan(&updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrUserNotFound
	}

	if err != nil {
		return time.Time{}, fmt.Errorf("sql update user: %w", err)
	}

	return updatedAt, nil
}

func (svc Service) sqlScanAvatar(dst **string) sql.Scanner {
	return &sqlAvatarScanner{Prefix: svc.AvatarsPrefix, Destination: dst}
}

type sqlAvatarScanner struct {
	Prefix      string
	Destination **string
}

func (s *sqlAvatarScanner) Scan(src any) error {
	str, ok := src.(string)
	if !ok {
		return nil
	}

	*s.Destination = ptr(s.Prefix + str)
	return nil
}
