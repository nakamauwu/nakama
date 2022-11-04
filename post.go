package nakama

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrInvalidPostID      = errs.InvalidArgumentError("invalid post ID")
	ErrInvalidPostContent = errs.InvalidArgumentError("invalid post content")
	ErrPostNotFound       = errs.NotFoundError("post not found")
)

const maxPostContentLength = 1000

// type TimelineItem struct {
// 	ID     string
// 	UserID string
// 	PostID string
// 	Post   Post
// }

type Post struct {
	ID            string
	UserID        string
	Content       string
	CommentsCount int32
	CreatedAt     time.Time
	UpdatedAt     time.Time
	User          UserPreview
}

type CreatePostInput struct {
	Content string
}

func (in *CreatePostInput) Prepare() {
	in.Content = smartTrim(in.Content)
}

func (in CreatePostInput) Validate() error {
	if in.Content == "" || !utf8.ValidString(in.Content) || utf8.RuneCountInString(in.Content) > maxPostContentLength {
		return ErrInvalidPostContent
	}
	return nil
}

type CreatePostOutput struct {
	ID       string
	CreateAt time.Time
}

type PostsInput struct {
	// Username is optional. If empty, all posts are returned.
	// Otherwise, only posts created by this user are returned.
	Username string
}

func (in PostsInput) Validate() error {
	if in.Username != "" && !isUsername(in.Username) {
		return ErrInvalidUsername
	}
	return nil
}

func (svc *Service) CreatePost(ctx context.Context, in CreatePostInput) (CreatePostOutput, error) {
	var out CreatePostOutput

	in.Prepare()
	if err := in.Validate(); err != nil {
		return out, err
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	postID := genID()
	createdAt, err := svc.sqlInsertPost(ctx, sqlInsertPost{
		PostID:  postID,
		UserID:  usr.ID,
		Content: in.Content,
	})
	if err != nil {
		return out, err
	}

	// Side-effect: increase user's posts count on inserts
	// so we don't have to compute it on each read.
	_, err = svc.sqlUpdateUser(ctx, sqlUpdateUser{
		UserID:               usr.ID,
		IncreasePostsCountBy: 1,
	})
	if err != nil {
		return out, err
	}

	// Side-effect: add the post to the user's timeline.
	_, err = svc.sqlInsertTimelineItem(ctx, sqlInsertTimelineItem{
		UserID: usr.ID,
		PostID: postID,
	})
	if err != nil {
		return out, err
	}

	// Side-effect: add the post to all followers' timelines.
	go func() {
		ctx := svc.BaseContext()
		_, err := svc.sqlInsertTimeline(ctx, sqlInsertTimeline{
			FollowedID: usr.ID,
			PostsID:    postID,
		})
		if err != nil {
			svc.Logger.Printf("failed to fanout timeline: %v\n", err)
		}
	}()

	out.ID = postID
	out.CreateAt = createdAt

	return out, nil
}

// Timeline is personalized list of posts to each user.
// When a post is created, a reference to the post is added to the user's timeline
// and is also fanned-out to all followers' timelines.
// This is so reads are faster since we don't have query the posts
// doing a JOIN with the user_follows table.
// We can query the timeline table of each user directly.
func (svc *Service) Timeline(ctx context.Context) ([]Post, error) {
	usr, ok := UserFromContext(ctx)
	if !ok {
		return nil, errs.Unauthenticated
	}

	return svc.sqlSelectTimeline(ctx, usr.ID)
}

func (svc *Service) Posts(ctx context.Context, in PostsInput) ([]Post, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	return svc.sqlSelectPosts(ctx, in.Username)
}

func (svc *Service) Post(ctx context.Context, postID string) (Post, error) {
	var out Post

	if !isID(postID) {
		return out, ErrInvalidPostID
	}

	out, err := svc.sqlSelectPost(ctx, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrPostNotFound
	}

	return out, err
}
