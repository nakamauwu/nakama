package nakama

import (
	"context"
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestService_CreatePost(t *testing.T) {
	ctx := context.Background()

	t.Run("empty_content", func(t *testing.T) {
		_, err := testService.CreatePost(ctx, CreatePostInput{Content: ""})
		assert.EqualError(t, err, "invalid post content")
	})

	t.Run("too_long_content", func(t *testing.T) {
		s := strings.Repeat("x", maxPostContentLength+1)
		_, err := testService.CreatePost(ctx, CreatePostInput{Content: s})
		assert.EqualError(t, err, "invalid post content")
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := testService.CreatePost(ctx, CreatePostInput{Content: genPostContent()})
		assert.EqualError(t, err, "unauthenticated")
	})

	t.Run("ok", func(t *testing.T) {
		asUser := ContextWithUser(ctx, genUser(t).Identity())
		got, err := testService.CreatePost(asUser, CreatePostInput{Content: genPostContent()})
		assert.NoError(t, err)
		assert.NotZero(t, got)
	})

	t.Run("user_posts_count", func(t *testing.T) {
		usr := genUser(t)
		asUser := ContextWithUser(ctx, usr.Identity())

		want := 5
		for i := 0; i < want; i++ {
			got, err := testService.CreatePost(asUser, CreatePostInput{Content: genPostContent()})
			assert.NoError(t, err)
			assert.NotZero(t, got)
		}

		got, err := testService.Queries.User(ctx, UserParams{Username: usr.Username})
		assert.NoError(t, err)
		assert.Equal(t, want, int(got.PostsCount))
	})

	t.Run("fanout", func(t *testing.T) {
		follower := genUser(t)
		followed := genUser(t)
		anotherUser := genUser(t)

		asFollower := ContextWithUser(ctx, follower.Identity())
		err := testService.FollowUser(asFollower, followed.ID)
		assert.NoError(t, err)

		asFollowed := ContextWithUser(ctx, followed.Identity())
		post, err := testService.CreatePost(asFollowed, CreatePostInput{Content: genPostContent()})
		assert.NoError(t, err)

		// The post should have been added the the author's timeline.
		timeline, err := testService.HomeTimeline(asFollowed)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(timeline))
		assert.Equal(t, post.ID, timeline[0].ID)

		// The post should have been added to the follower's timeline.
		timeline, err = testService.HomeTimeline(asFollower)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(timeline))
		assert.Equal(t, post.ID, timeline[0].ID)

		// The post should not have been added to any other user's timeline.
		asAnotherUser := ContextWithUser(ctx, anotherUser.Identity())
		timeline, err = testService.HomeTimeline(asAnotherUser)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(timeline))
	})
}

func TestService_Posts(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_username", func(t *testing.T) {
		_, err := testService.Posts(ctx, PostsInput{Username: "@nope@"})
		assert.EqualError(t, err, "invalid username")
	})

	t.Run("optional_username", func(t *testing.T) {
		_, err := testService.Posts(ctx, PostsInput{})
		assert.NoError(t, err)
	})

	t.Run("ok", func(t *testing.T) {
		wantAtLeast := 5
		for i := 0; i < wantAtLeast; i++ {
			genPost(t, genUser(t).ID)
		}

		got, err := testService.Posts(ctx, PostsInput{})
		assert.NoError(t, err)
		assert.True(t, len(got) >= wantAtLeast, "got %d posts, want at least %d", len(got), wantAtLeast)
		for _, p := range got {
			assert.NotZero(t, p)
		}
	})

	t.Run("ok_with_username", func(t *testing.T) {
		usr := genUser(t)
		want := 5
		for i := 0; i < want; i++ {
			genPost(t, usr.ID)
		}
		genPost(t, genUser(t).ID) // additional post from another user

		got, err := testService.Posts(ctx, PostsInput{Username: usr.Username})
		assert.NoError(t, err)
		assert.Equal(t, want, len(got))
		for _, p := range got {
			assert.NotZero(t, p)
		}
	})
}

func TestService_Post(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_id", func(t *testing.T) {
		_, err := testService.Post(ctx, "@nope@")
		assert.EqualError(t, err, "invalid post ID")
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := testService.Post(ctx, genID())
		assert.EqualError(t, err, "post not found")
	})

	t.Run("ok", func(t *testing.T) {
		post := genPost(t, genUser(t).ID)
		got, err := testService.Post(ctx, post.ID)
		assert.NoError(t, err)
		assert.NotZero(t, got)
	})
}

func genPost(t *testing.T, userID string) Post {
	t.Helper()

	ctx := context.Background()
	postID := genID()
	createdAt, err := testService.Queries.CreatePost(ctx, CreatePostParams{
		PostID:  postID,
		UserID:  userID,
		Content: genPostContent(),
	})
	assert.NoError(t, err)
	return Post{
		ID:        postID,
		UserID:    userID,
		Content:   genPostContent(),
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func genPostContent() string {
	return randString(10)
}
