package nakama

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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
	ID             string
	UserID         string
	Content        string
	ReactionsCount ReactionsCount
	CommentsCount  int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
	User           UserPreview
}

type CreatePost struct {
	Content string
}

func (in *CreatePost) Validate() error {
	in.Content = smartTrim(in.Content)

	if in.Content == "" || !utf8.ValidString(in.Content) || utf8.RuneCountInString(in.Content) > maxPostContentLength {
		return ErrInvalidPostContent
	}

	return nil
}

type CreatedPost struct {
	ID        string
	CreatedAt time.Time
}

type PostsParams struct {
	// Username is optional. If empty, all posts are returned.
	// Otherwise, only posts created by this user are returned.
	Username string
}

func (in *PostsParams) Validate() error {
	in.Username = strings.TrimSpace(in.Username)

	if in.Username != "" && !validUsername(in.Username) {
		return ErrInvalidUsername
	}

	return nil
}

func (svc *Service) CreatePost(ctx context.Context, in CreatePost) (CreatedPost, error) {
	var out CreatedPost

	if err := in.Validate(); err != nil {
		return out, err
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	var err error
	out, err = svc.sqlInsertPost(ctx, sqlInsertPost{
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
		PostID: out.ID,
	})
	if err != nil {
		return out, err
	}

	// Side-effect: add the post to all followers' timelines.
	go func() {
		ctx := svc.BaseContext()
		_, err := svc.sqlInsertTimeline(ctx, sqlInsertTimeline{
			PostID:     out.ID,
			FollowedID: usr.ID,
		})
		if err != nil {
			svc.Logger.Printf("failed to fanout timeline: %v\n", err)
		}
	}()

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

func (svc *Service) Posts(ctx context.Context, in PostsParams) ([]Post, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	usr, _ := UserFromContext(ctx)
	return svc.sqlSelectPosts(ctx, sqlSelectPosts{
		AuthUserID: usr.ID,
		Username:   in.Username,
	})
}

func (svc *Service) Post(ctx context.Context, postID string) (Post, error) {
	var out Post

	if !validID(postID) {
		return out, ErrInvalidPostID
	}

	out, err := svc.sqlSelectPost(ctx, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrPostNotFound
	}

	return out, err
}
