package nakama

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_Users(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		_, err := testService.Users(ctx, ListUsers{UsernameQuery: "@nope@"})
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("no_match", func(t *testing.T) {
		genUser(t, func(in *CreateUser) {
			in.Username = "tomas"
		})

		got, err := testService.Users(ctx, ListUsers{UsernameQuery: "liz"})
		assert.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t, func(in *CreateUser) {
			in.Username = "bob"
		})

		got, err := testService.Users(ctx, ListUsers{UsernameQuery: "boo"})
		assert.NoError(t, err)
		assert.Equal(t, usr, got[0])
	})
}

func TestService_User(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		_, err := testService.User(ctx, RetrieveUser{Username: "@nope@"})
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := testService.User(ctx, RetrieveUser{Username: randString(10)})
		assert.EqualError(t, err, "user not found")
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t)
		got, err := testService.User(ctx, RetrieveUser{Username: usr.Username})
		assert.NoError(t, err)
		assert.Equal(t, usr.ID, got.ID)
		assert.Equal(t, usr.Email, got.Email)
		assert.Equal(t, usr.Username, got.Username)
		assert.Equal(t, usr.CreatedAt, got.CreatedAt)
		assert.Equal(t, usr.UpdatedAt, got.UpdatedAt)
	})

	t.Run("following", func(t *testing.T) {
		{
			usr, err := testService.User(ctx, RetrieveUser{Username: genUser(t).Username})
			assert.NoError(t, err)
			assert.False(t, usr.Following)
		}

		follower := genUser(t)
		followed := genUser(t)
		asFollower := ContextWithUser(ctx, follower.Identity())

		{
			err := testService.FollowUser(asFollower, followed.ID)
			assert.NoError(t, err)

			usr, err := testService.User(asFollower, RetrieveUser{Username: followed.Username})
			assert.NoError(t, err)
			assert.True(t, usr.Following)
		}

		{
			err := testService.UnfollowUser(asFollower, followed.ID)
			assert.NoError(t, err)

			usr, err := testService.User(asFollower, RetrieveUser{Username: followed.Username})
			assert.NoError(t, err)
			assert.False(t, usr.Following)
		}
	})
}

func genUser(t *testing.T, override ...func(in *CreateUser)) User {
	t.Helper()

	ctx := context.Background()
	in := CreateUser{
		Email:    genEmail(),
		Username: randString(10),
	}
	for _, fn := range override {
		fn(&in)
	}
	inserted, err := testService.Store.CreateUser(ctx, in)
	assert.NoError(t, err)

	return User{
		ID:        inserted.ID,
		Email:     in.Email,
		Username:  in.Username,
		CreatedAt: inserted.CreatedAt,
		UpdatedAt: inserted.CreatedAt,
	}
}

func genEmail() string {
	return randString(10) + "@example.org"
}
