package nakama

import (
	"context"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func TestService_Login(t *testing.T) {
	svc := &Service{Queries: testQueries}
	ctx := context.Background()

	t.Run("invalid_email", func(t *testing.T) {
		_, err := svc.Login(ctx, LoginInput{Email: "nope"})
		assert.EqualError(t, err, "invalid email")
	})

	t.Run("invalid_username", func(t *testing.T) {
		_, err := svc.Login(ctx, LoginInput{Email: genEmail(), Username: ptr("@nope@")})
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("user_not_found", func(t *testing.T) {
		_, err := svc.Login(ctx, LoginInput{Email: genEmail()})
		assert.EqualError(t, err, "user not found")
	})

	t.Run("username_taken", func(t *testing.T) {
		sameUsername := genUsername()
		_, err := svc.Login(ctx, LoginInput{Email: genEmail(), Username: ptr(sameUsername)})
		assert.NoError(t, err)

		_, err = svc.Login(ctx, LoginInput{Email: genEmail(), Username: ptr(sameUsername)})
		assert.EqualError(t, err, "username taken")
	})

	t.Run("ok", func(t *testing.T) {
		email := genEmail()
		got, err := svc.Login(ctx, LoginInput{Email: email, Username: ptr(genUsername())})
		assert.NoError(t, err)
		assert.NotZero(t, got)
		assert.Equal(t, email, got.Email)

		got2, err := svc.Login(ctx, LoginInput{Email: email})
		assert.NoError(t, err)
		assert.Equal(t, got, got2)
	})

	t.Run("lowercase_email", func(t *testing.T) {
		email := genEmail()
		got, err := svc.Login(ctx, LoginInput{Email: strings.ToUpper(email), Username: ptr(genUsername())})
		assert.NoError(t, err)
		assert.Equal(t, strings.ToLower(email), got.Email)

		_, err = svc.Login(ctx, LoginInput{Email: strings.ToLower(email)})
		assert.NoError(t, err)
	})
}

func ptr[T any](v T) *T {
	return &v
}

func genEmail() string {
	return randString(10) + "@example.org"
}

func genUsername() string {
	return randString(10)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
