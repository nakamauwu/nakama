package nakama

import (
	"context"
	"fmt"
	"time"
)

type sqlInsertUserFollow struct {
	FollowerID string
	FollowedID string
}

type sqlDeleteUserFollow struct {
	FollowerID string
	FollowedID string
}

type sqlSelectUserFollowExists struct {
	FollowerID string
	FollowedID string
}

func (svc *Service) sqlInsertUserFollow(ctx context.Context, in sqlInsertUserFollow) (time.Time, error) {
	const createUserFollow = `
		INSERT INTO user_follows (follower_id, followed_id)
		VALUES ($1, $2)
		RETURNING created_at
	`
	var createdAt time.Time
	err := svc.DB.QueryRowContext(ctx, createUserFollow, in.FollowerID, in.FollowedID).Scan(&createdAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql insert user follow: %w", err)
	}
	return createdAt, nil
}

func (svc *Service) sqlDeleteUserFollow(ctx context.Context, in sqlDeleteUserFollow) (time.Time, error) {
	const deleteUserFollow = `
		DELETE FROM user_follows
		WHERE follower_id = $1
		AND followed_id = $2
		RETURNING now()::timestamp AS deleted_at
	`
	var deletedAt time.Time
	err := svc.DB.QueryRowContext(ctx, deleteUserFollow, in.FollowerID, in.FollowedID).Scan(&deletedAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("sql delete user follow: %w", err)
	}

	return deletedAt, nil
}

func (svc *Service) sqlSelectUserFollowExists(ctx context.Context, in sqlSelectUserFollowExists) (bool, error) {
	const userFollowExists = `
		SELECT EXISTS (
			SELECT 1 FROM user_follows
			WHERE follower_id = $1
			AND followed_id = $2
		)
	`
	var exists bool
	err := svc.DB.QueryRowContext(ctx, userFollowExists, in.FollowerID, in.FollowedID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("sql select user follow exists: %w", err)
	}

	return exists, nil
}
