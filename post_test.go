package nakama

import (
	"context"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_CreatePost(t *testing.T) {
	svc := &Service{Queries: testQueries}
	ctx := context.Background()

	t.Run("empty_content", func(t *testing.T) {
		_, err := svc.CreatePost(ctx, CreatePostInput{Content: ""})
		assert.EqualError(t, err, "invalid post content")
	})

	t.Run("too_long_content", func(t *testing.T) {
		s := strings.Repeat("a", 1001)
		_, err := svc.CreatePost(ctx, CreatePostInput{Content: s})
		assert.EqualError(t, err, "invalid post content")
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := svc.CreatePost(ctx, CreatePostInput{Content: genPostContent()})
		assert.EqualError(t, err, "unauthenticated")
	})

	t.Run("ok", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t))
		_, err := svc.CreatePost(asUser, CreatePostInput{Content: genPostContent()})
		assert.NoError(t, err)
	})
}

func genPostContent() string {
	return randString(10)
}

func genUser(t *testing.T) User {
	t.Helper()

	ctx := context.Background()
	userID := genID()
	email := genEmail()
	username := genUsername()
	createdAt, err := testQueries.CreateUser(ctx, CreateUserParams{
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
