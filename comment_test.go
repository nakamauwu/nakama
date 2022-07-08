package nakama

import (
	"context"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_CreateComment(t *testing.T) {
	svc := &Service{Queries: testQueries}
	ctx := context.Background()

	t.Run("invalid_post_id", func(t *testing.T) {
		_, err := svc.CreateComment(ctx, CreateCommentInput{
			PostID: "@nope@",
		})
		assert.EqualError(t, err, "invalid post ID")
	})

	t.Run("empty_content", func(t *testing.T) {
		_, err := svc.CreateComment(ctx, CreateCommentInput{
			PostID:  genID(),
			Content: "",
		})
		assert.EqualError(t, err, "invalid comment content")
	})

	t.Run("too_long_content", func(t *testing.T) {
		s := strings.Repeat("a", 1001)
		_, err := svc.CreateComment(ctx, CreateCommentInput{
			PostID:  genID(),
			Content: s,
		})
		assert.EqualError(t, err, "invalid comment content")
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := svc.CreateComment(ctx, CreateCommentInput{
			PostID:  genID(),
			Content: genCommentContent(),
		})
		assert.EqualError(t, err, "unauthenticated")
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t)
		asUser := ContextWithUser(ctx, usr)
		post := genPost(t, usr.ID)
		got, err := svc.CreateComment(asUser, CreateCommentInput{
			PostID:  post.ID,
			Content: genCommentContent(),
		})
		assert.NoError(t, err)
		assert.NotZero(t, got)
	})
}

func TestService_Comments(t *testing.T) {
	svc := &Service{Queries: testQueries}
	ctx := context.Background()

	t.Run("invalid_post_id", func(t *testing.T) {
		_, err := svc.Comments(ctx, "@nope@")
		assert.EqualError(t, err, "invalid post ID")
	})

	t.Run("empty", func(t *testing.T) {
		got, err := svc.Comments(ctx, genID())
		assert.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t)
		post := genPost(t, usr.ID)

		want := 10
		for i := 0; i < want; i++ {
			_ = genComment(t, usr.ID, post.ID)
		}

		got, err := svc.Comments(ctx, post.ID)
		assert.NoError(t, err)
		assert.Equal(t, want, len(got))
		for _, p := range got {
			assert.NotZero(t, p)
		}
	})
}

func genComment(t *testing.T, userID, postID string) Comment {
	t.Helper()
	commentID := genID()
	createdAt, err := testQueries.CreateComment(context.Background(), CreateCommentParams{
		CommentID: commentID,
		PostID:    postID,
		UserID:    userID,
		Content:   genCommentContent(),
	})
	assert.NoError(t, err)
	return Comment{
		ID:        commentID,
		CreatedAt: createdAt,
	}
}

func genCommentContent() string {
	return randString(10)
}
