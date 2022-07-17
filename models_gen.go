// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.14.0

package nakama

import (
	"time"
)

type Comment struct {
	ID        string
	UserID    string
	PostID    string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Post struct {
	ID            string
	UserID        string
	Content       string
	CommentsCount int32
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type User struct {
	ID             string
	Email          string
	Username       string
	PostsCount     int32
	FollowersCount int32
	FollowingCount int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserFollow struct {
	FollowerID string
	FollowedID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
