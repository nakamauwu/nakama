package nakama

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_FollowUser(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_user_id", func(t *testing.T) {
		err := testService.FollowUser(ctx, "@nope@")
		assert.EqualError(t, err, "invalid user ID")
	})

	t.Run("unauthenticated", func(t *testing.T) {
		err := testService.FollowUser(ctx, genID())
		assert.EqualError(t, err, "unauthenticated")
	})

	t.Run("self", func(t *testing.T) {
		usr := genUser(t)
		asUser := ContextWithUser(ctx, usr.Identity())
		err := testService.FollowUser(asUser, usr.ID)
		assert.EqualError(t, err, "cannot follow self")
	})

	t.Run("user_not_found", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		err := testService.FollowUser(asUser, genID())
		assert.EqualError(t, err, "user not found")
	})

	t.Run("ok", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		err := testService.FollowUser(asUser, genUser(t).ID)
		assert.NoError(t, err)
	})

	t.Run("exists", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		followedUserID := genUser(t).ID
		err := testService.FollowUser(asUser, followedUserID)
		assert.NoError(t, err)

		err = testService.FollowUser(asUser, followedUserID)
		assert.NoError(t, err)
	})

	t.Run("follow_counts", func(t *testing.T) {
		follower := genUser(t)
		followed := genUser(t)

		asFollower := ContextWithUser(ctx, follower.Identity())
		err := testService.FollowUser(asFollower, followed.ID)
		assert.NoError(t, err)

		{
			follower, err := testService.User(ctx, follower.Username)
			assert.NoError(t, err)
			assert.Equal(t, 1, follower.FollowingCount)
		}

		{
			followed, err := testService.User(ctx, followed.Username)
			assert.NoError(t, err)
			assert.Equal(t, 1, followed.FollowersCount)
		}
	})
}

func TestService_UnfollowUser(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_user_id", func(t *testing.T) {
		err := testService.UnfollowUser(ctx, "@nope@")
		assert.EqualError(t, err, "invalid user ID")
	})

	t.Run("unauthenticated", func(t *testing.T) {
		err := testService.UnfollowUser(ctx, genID())
		assert.EqualError(t, err, "unauthenticated")
	})

	t.Run("self", func(t *testing.T) {
		usr := genUser(t)
		asUser := ContextWithUser(ctx, usr.Identity())
		err := testService.UnfollowUser(asUser, usr.ID)
		assert.EqualError(t, err, "cannot follow self")
	})

	t.Run("user_not_found", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		err := testService.UnfollowUser(asUser, genID())
		assert.EqualError(t, err, "user not found")
	})

	t.Run("ok", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		followedUserID := genUser(t).ID
		err := testService.FollowUser(asUser, followedUserID)
		assert.NoError(t, err)

		err = testService.UnfollowUser(asUser, followedUserID)
		assert.NoError(t, err)
	})

	t.Run("exists", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		followedUserID := genUser(t).ID
		err := testService.FollowUser(asUser, followedUserID)
		assert.NoError(t, err)

		err = testService.UnfollowUser(asUser, followedUserID)
		assert.NoError(t, err)

		err = testService.UnfollowUser(asUser, followedUserID)
		assert.NoError(t, err)
	})

	t.Run("follow_counts", func(t *testing.T) {
		follower := genUser(t)
		followed := genUser(t)

		asFollower := ContextWithUser(ctx, follower.Identity())
		err := testService.FollowUser(asFollower, followed.ID)
		assert.NoError(t, err)

		err = testService.UnfollowUser(asFollower, followed.ID)
		assert.NoError(t, err)

		{
			follower, err := testService.User(ctx, follower.Username)
			assert.NoError(t, err)
			assert.Equal(t, 0, follower.FollowingCount)
		}

		{
			followed, err := testService.User(ctx, followed.Username)
			assert.NoError(t, err)
			assert.Equal(t, 0, followed.FollowersCount)
		}
	})
}
