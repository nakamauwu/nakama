package nakama

import (
	"context"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_email", func(t *testing.T) {
		_, err := testService.Login(ctx, Login{Email: "nope"})
		assert.EqualError(t, err, "invalid email")
	})

	t.Run("invalid_username", func(t *testing.T) {
		_, err := testService.Login(ctx, Login{Email: genEmail(), Username: ptr("@nope@")})
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("user_not_found", func(t *testing.T) {
		_, err := testService.Login(ctx, Login{Email: genEmail()})
		assert.EqualError(t, err, "user not found")
	})

	t.Run("username_taken", func(t *testing.T) {
		sameUsername := genUsername()
		_, err := testService.Login(ctx, Login{Email: genEmail(), Username: ptr(sameUsername)})
		assert.NoError(t, err)

		_, err = testService.Login(ctx, Login{Email: genEmail(), Username: ptr(sameUsername)})
		assert.EqualError(t, err, "username taken")
	})

	t.Run("ok", func(t *testing.T) {
		email := genEmail()
		got, err := testService.Login(ctx, Login{Email: email, Username: ptr(genUsername())})
		assert.NoError(t, err)
		assert.NotZero(t, got)
		assert.Equal(t, email, got.Email)

		got2, err := testService.Login(ctx, Login{Email: email})
		assert.NoError(t, err)
		assert.Equal(t, got, got2)
	})

	t.Run("lowercase_email", func(t *testing.T) {
		email := genEmail()
		got, err := testService.Login(ctx, Login{Email: strings.ToUpper(email), Username: ptr(genUsername())})
		assert.NoError(t, err)
		assert.Equal(t, strings.ToLower(email), got.Email)

		_, err = testService.Login(ctx, Login{Email: strings.ToLower(email)})
		assert.NoError(t, err)
	})
}
