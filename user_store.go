package nakama

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func (db *Store) CreateUser(ctx context.Context, in CreateUser) (Created, error) {
	var out Created

	const query = `
		INSERT INTO users (id, email, username)
		VALUES ($1, LOWER($2), $3)
		RETURNING created_at
	`
	userID := genID()
	err := db.QueryRowContext(ctx, query, userID, in.Email, in.Username).Scan(&out.CreatedAt)
	if err != nil {
		return out, fmt.Errorf("sql insert user: %w", err)
	}

	out.ID = userID

	return out, nil
}

func (db *Store) Users(ctx context.Context, in ListUsers) ([]User, error) {
	const query = `
		SELECT users.id
			, users.email
			, users.username
			, (
				CASE
					WHEN $1::varchar != '' THEN similarity(username, $1::varchar)
					ELSE 0
				END
			) AS similarity
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
					WHEN $2::varchar != '' AND $2 != users.id THEN (
						SELECT EXISTS (
							SELECT 1 FROM user_follows
							WHERE follower_id = $2::varchar
							AND followed_id = users.id
						)
					)
					ELSE false
				END
			) AS following
		FROM users
		WHERE CASE
			-- Text search over a GiST index.
			WHEN $1::varchar != '' THEN LOWER(users.username) % LOWER($1::varchar)
			ELSE false
		END
		ORDER BY similarity DESC, users.id DESC
	`

	rows, err := db.QueryContext(ctx, query, in.UsernameQuery, in.authUserID)
	if err != nil {
		return nil, fmt.Errorf("sql select users: %w", err)
	}

	return collect(rows, func(scanner scanner) (User, error) {
		var u User
		var sim float64 // unused
		return u, scanner.Scan(
			&u.ID,
			&u.Email,
			&u.Username,
			&sim,
			db.AvatarScanFunc(&u.AvatarPath),
			&u.AvatarWidth,
			&u.AvatarHeight,
			&u.PostsCount,
			&u.FollowersCount,
			&u.FollowingCount,
			&u.CreatedAt,
			&u.UpdatedAt,
			&u.Following,
		)
	})
}

func (db *Store) User(ctx context.Context, in RetrieveUser) (User, error) {
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
	err := db.QueryRowContext(ctx, query,
		in.authUserID,
		in.id,
		in.email,
		in.Username,
	).Scan(
		&usr.ID,
		&usr.Email,
		&usr.Username,
		db.AvatarScanFunc(&usr.AvatarPath),
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

func (db *Store) UserExists(ctx context.Context, in RetrieveUserExists) (bool, error) {
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
	err := db.QueryRowContext(ctx, query, in.UserID, in.Email, in.Username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select user exists: %w", err)
	}

	return exists, nil
}

func (db *Store) UpdateUser(ctx context.Context, in UpdateUser) (time.Time, error) {
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
	err := db.QueryRowContext(ctx, query,
		in.Username,
		in.avatarPath,
		in.avatarWidth,
		in.avatarHeight,
		in.increasePostsCountBy,
		in.increaseFollowersCountBy,
		in.increaseFollowingCountBy,
		in.userID,
	).Scan(&updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrUserNotFound
	}

	if err != nil {
		return time.Time{}, fmt.Errorf("sql update user: %w", err)
	}

	return updatedAt, nil
}
