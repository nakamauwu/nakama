package nakama

import (
	"context"
	"fmt"
	"time"
)

func (db *Store) CreateUserFollow(ctx context.Context, in UserFollow) (time.Time, error) {
	const createUserFollow = `
		INSERT INTO user_follows (follower_id, followed_id)
		VALUES ($1, $2)
		RETURNING created_at
	`
	var createdAt time.Time
	err := db.QueryRowContext(ctx, createUserFollow, in.FollowerID, in.FollowedID).Scan(&createdAt)
	if isPqForeignKeyViolationError(err, "followed_id") {
		return time.Time{}, ErrUserNotFound
	}

	if err != nil {
		return time.Time{}, fmt.Errorf("sql insert user follow: %w", err)
	}

	return createdAt, nil
}

func (db *Store) UserFollowExists(ctx context.Context, in UserFollow) (bool, error) {
	const userFollowExists = `
		SELECT EXISTS (
			SELECT 1 FROM user_follows
			WHERE follower_id = $1
			AND followed_id = $2
		)
	`
	var exists bool
	err := db.QueryRowContext(ctx, userFollowExists, in.FollowerID, in.FollowedID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select user follow exists: %w", err)
	}

	return exists, nil
}

func (db *Store) DeleteUserFollow(ctx context.Context, in UserFollow) (time.Time, error) {
	const deleteUserFollow = `
		DELETE FROM user_follows
		WHERE follower_id = $1
		AND followed_id = $2
		RETURNING now()::timestamp AS deleted_at
	`
	var deletedAt time.Time
	err := db.QueryRowContext(ctx, deleteUserFollow, in.FollowerID, in.FollowedID).Scan(&deletedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql delete user follow: %w", err)
	}

	return deletedAt, nil
}
