package nakama

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_UserByUsername(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		_, err := testService.UserByUsername(ctx, "@nope@")
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := testService.UserByUsername(ctx, genUsername())
		assert.EqualError(t, err, "user not found")
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t)
		got, err := testService.UserByUsername(ctx, usr.Username)
		assert.NoError(t, err)
		assert.Equal(t, usr.ID, got.ID)
		assert.Equal(t, usr.Email, got.Email)
		assert.Equal(t, usr.Username, got.Username)
		assert.Equal(t, usr.CreatedAt, got.CreatedAt)
		assert.Equal(t, usr.UpdatedAt, got.UpdatedAt)
	})

	t.Run("following", func(t *testing.T) {
		{
			usr, err := testService.UserByUsername(ctx, genUser(t).Username)
			assert.NoError(t, err)
			assert.False(t, usr.Following)
		}

		follower := genUser(t)
		followed := genUser(t)
		asFollower := ContextWithUser(ctx, follower)

		{
			err := testService.FollowUser(asFollower, followed.ID)
			assert.NoError(t, err)

			usr, err := testService.UserByUsername(asFollower, followed.Username)
			assert.NoError(t, err)
			assert.True(t, usr.Following)
		}

		{
			err := testService.UnfollowUser(asFollower, followed.ID)
			assert.NoError(t, err)

			usr, err := testService.UserByUsername(asFollower, followed.Username)
			assert.NoError(t, err)
			assert.False(t, usr.Following)
		}
	})
}

func genUser(t *testing.T) User {
	t.Helper()

	ctx := context.Background()
	userID := genID()
	email := genEmail()
	username := genUsername()
	createdAt, err := testService.Queries.CreateUser(ctx, CreateUserParams{
		UserID:   userID,
		Email:    email,
		Username: username,
	})
	assert.NoError(t, err)
	return User{
		ID:        userID,
		Email:     email,
		Username:  username,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func genEmail() string {
	return randString(10) + "@example.org"
}

func genUsername() string {
	return randString(10)
}