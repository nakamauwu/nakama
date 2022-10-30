// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.15.0
// source: user_follow.sql

package nakama

import (
	"context"
	"time"
)

const createUserFollow = `-- name: CreateUserFollow :one
INSERT INTO user_follows (follower_id, followed_id)
VALUES ($1, $2)
RETURNING created_at
`

type CreateUserFollowParams struct {
	FollowerID string
	FollowedID string
}

func (q *Queries) CreateUserFollow(ctx context.Context, arg CreateUserFollowParams) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, createUserFollow, arg.FollowerID, arg.FollowedID)
	var created_at time.Time
	err := row.Scan(&created_at)
	return created_at, err
}

const deleteUserFollow = `-- name: DeleteUserFollow :one
DELETE FROM user_follows
WHERE follower_id = $1
AND followed_id = $2
RETURNING now()::timestamp AS deleted_at
`

type DeleteUserFollowParams struct {
	FollowerID string
	FollowedID string
}

func (q *Queries) DeleteUserFollow(ctx context.Context, arg DeleteUserFollowParams) (time.Time, error) {
	row := q.db.QueryRowContext(ctx, deleteUserFollow, arg.FollowerID, arg.FollowedID)
	var deleted_at time.Time
	err := row.Scan(&deleted_at)
	return deleted_at, err
}

const userFollowExists = `-- name: UserFollowExists :one
SELECT EXISTS (
    SELECT 1 FROM user_follows
    WHERE follower_id = $1
    AND followed_id = $2
)
`

type UserFollowExistsParams struct {
	FollowerID string
	FollowedID string
}

func (q *Queries) UserFollowExists(ctx context.Context, arg UserFollowExistsParams) (bool, error) {
	row := q.db.QueryRowContext(ctx, userFollowExists, arg.FollowerID, arg.FollowedID)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}
