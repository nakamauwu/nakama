package nakama

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_FollowUser(t *testing.T) {
	svc := &Service{Queries: testQueries}
	ctx := context.Background()

	t.Run("invalid_user_id", func(t *testing.T) {
		err := svc.FollowUser(ctx, "@nope@")
		assert.EqualError(t, err, "invalid user ID")
	})

	t.Run("unauthenticated", func(t *testing.T) {
		err := svc.FollowUser(ctx, genID())
		assert.EqualError(t, err, "unauthenticated")
	})

	t.Run("self", func(t *testing.T) {
		usr := genUser(t)
		asUser := ContextWithUser(ctx, usr)
		err := svc.FollowUser(asUser, usr.ID)
		assert.EqualError(t, err, "cannot follow self")
	})

	t.Run("user_not_found", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t))
		err := svc.FollowUser(asUser, genID())
		assert.EqualError(t, err, "user not found")
	})

	t.Run("ok", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t))
		err := svc.FollowUser(asUser, genUser(t).ID)
		assert.NoError(t, err)
	})

	t.Run("exists", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t))
		followUserID := genUser(t).ID
		err := svc.FollowUser(asUser, followUserID)
		assert.NoError(t, err)

		err = svc.FollowUser(asUser, followUserID)
		assert.NoError(t, err)
	})
}
