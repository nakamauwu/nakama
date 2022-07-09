package nakama

import (
	"context"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_UserByUsername(t *testing.T) {
	svc := &Service{Queries: testQueries}
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		_, err := svc.UserByUsername(ctx, "@nope@")
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := svc.UserByUsername(ctx, genUsername())
		assert.EqualError(t, err, "user not found")
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t)
		got, err := svc.UserByUsername(ctx, usr.Username)
		assert.NoError(t, err)
		assert.Equal(t, usr, got)
	})
}
