package nakama

import (
	"context"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_CreateComment(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_post_id", func(t *testing.T) {
		_, err := testService.CreateComment(ctx, CreateComment{
			PostID: "@nope@",
		})
		assert.EqualError(t, err, "invalid post ID")
	})

	t.Run("empty_content", func(t *testing.T) {
		_, err := testService.CreateComment(ctx, CreateComment{
			PostID:  genID(),
			Content: "",
		})
		assert.EqualError(t, err, "invalid comment content")
	})

	t.Run("too_long_content", func(t *testing.T) {
		s := strings.Repeat("a", maxCommentContentLength+1)
		_, err := testService.CreateComment(ctx, CreateComment{
			PostID:  genID(),
			Content: s,
		})
		assert.EqualError(t, err, "invalid comment content")
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := testService.CreateComment(ctx, CreateComment{
			PostID:  genID(),
			Content: genCommentContent(),
		})
		assert.EqualError(t, err, "unauthenticated")
	})

	t.Run("post_not_found", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		_, err := testService.CreateComment(asUser, CreateComment{
			PostID:  genID(),
			Content: genCommentContent(),
		})
		assert.EqualError(t, err, "post not found")
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t)
		asUser := ContextWithUser(ctx, usr.Identity())
		post := genPost(t, usr.ID)
		got, err := testService.CreateComment(asUser, CreateComment{
			PostID:  post.ID,
			Content: genCommentContent(),
		})
		assert.NoError(t, err)
		assert.NotZero(t, got)
	})
}

func TestService_Comments(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_post_id", func(t *testing.T) {
		_, err := testService.Comments(ctx, "@nope@")
		assert.EqualError(t, err, "invalid post ID")
	})

	t.Run("empty", func(t *testing.T) {
		got, err := testService.Comments(ctx, genID())
		assert.NoError(t, err)
		assert.Zero(t, got)
	})

	t.Run("ok", func(t *testing.T) {
		usr := genUser(t)
		post := genPost(t, usr.ID)

		want := 5
		for i := 0; i < want; i++ {
			genComment(t, usr.ID, post.ID)
		}

		got, err := testService.Comments(ctx, post.ID)
		assert.NoError(t, err)
		assert.Equal(t, want, len(got))
		for _, p := range got {
			assert.NotZero(t, p)
		}
	})
}

func genComment(t *testing.T, userID, postID string) Comment {
	t.Helper()

	in := CreateComment{
		PostID:  postID,
		Content: genCommentContent(),

		userID: userID,
	}
	inserted, err := testService.Store.CreateComment(context.Background(), in)
	assert.NoError(t, err)
	return Comment{
		ID:        inserted.ID,
		PostID:    in.PostID,
		UserID:    in.userID,
		Content:   in.Content,
		CreatedAt: inserted.CreatedAt,
		UpdatedAt: inserted.CreatedAt,
	}
}

func genCommentContent() string {
	return randString(10)
}
