package nakama

import (
	"context"
	"fmt"
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

type Post struct {
	ID             string
	UserID         string
	Content        string
	Media          []Media
	ReactionsCount ReactionsCount
	CommentsCount  int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
	User           UserPreview
}

type CreatePost struct {
	Content string
	Media   []Media

	userID string
}

func (in *CreatePost) Validate() error {
	in.Content = smartTrim(in.Content)

	if in.Content == "" || !utf8.ValidString(in.Content) || utf8.RuneCountInString(in.Content) > maxPostContentLength {
		return ErrInvalidPostContent
	}

	for _, media := range in.Media {
		if err := media.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type CreateTimelineItem struct {
	userID string
	postID string
}

type CreateTimeline struct {
	postID     string
	followedID string
}

type CreatedTimelineItem struct {
	ID     string
	UserID string
}

type ListPosts struct {
	// Username is optional. If empty, all posts are returned.
	// Otherwise, only posts created by this user are returned.
	Username string

	authUserID string
}

func (in *ListPosts) Validate() error {
	in.Username = strings.TrimSpace(in.Username)

	if in.Username != "" && !validUsername(in.Username) {
		return ErrInvalidUsername
	}

	return nil
}

type RetrievePost struct {
	ID string

	authUserID string
}

func (in *RetrievePost) Validate() error {
	in.ID = strings.TrimSpace(in.ID)

	if !validID(in.ID) {
		return ErrInvalidPostID
	}

	return nil
}

type UpdatePost struct {
	PostID                  string
	IncreaseCommentsCountBy int32
	ReactionsCount          *ReactionsCount
}

func (svc *Service) CreatePost(ctx context.Context, in CreatePost) (Created, error) {
	var out Created

	if err := in.Validate(); err != nil {
		return out, err
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	in.userID = user.ID

	for _, media := range in.Media {
		switch {
		case media.IsImage():
			img := *media.AsImage
			err := svc.s3StoreObject(ctx, s3StoreObject{
				File:        img,
				Bucket:      S3BucketMedia,
				Name:        img.Path,
				Size:        img.byteSize,
				ContentType: img.contentType,
			})
			if err != nil {
				return out, err
			}
		}
	}

	err := svc.Store.RunTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = svc.Store.CreatePost(ctx, in)
		if err != nil {
			return err
		}

		// Side effect: increase user's posts count on inserts,
		// so we don't have to compute it on each read.
		_, err = svc.Store.UpdateUser(ctx, UpdateUser{
			userID:               user.ID,
			increasePostsCountBy: 1,
		})
		if err != nil {
			return err
		}

		// Side effect: add the post to the user's timeline.
		_, err = svc.Store.CreateTimelineItem(ctx, CreateTimelineItem{
			userID: user.ID,
			postID: out.ID,
		})
		return err
	})
	if err != nil {
		return out, err
	}

	// Side effect: add the post to all followers' timelines.
	svc.background(func(ctx context.Context) error {
		_, err := svc.Store.CreateTimeline(ctx, CreateTimeline{
			postID:     out.ID,
			followedID: user.ID,
		})
		if err != nil {
			return fmt.Errorf("fanout timeline: %w", err)
		}

		return nil
	})

	return out, nil
}

// Timeline is personalized list of posts to each user.
// When a post is created, a reference to the post is added to the user's timeline
// and is also fanned-out to all followers' timelines.
// This is so reads are faster since we don't have query the posts
// doing a JOIN with the user_follows table.
// We can query the timeline table of each user directly.
func (svc *Service) Timeline(ctx context.Context) ([]Post, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return nil, errs.Unauthenticated
	}

	return svc.Store.Timeline(ctx, user.ID)
}

func (svc *Service) Posts(ctx context.Context, in ListPosts) ([]Post, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	user, _ := UserFromContext(ctx)
	in.authUserID = user.ID

	return svc.Store.Posts(ctx, in)
}

func (svc *Service) Post(ctx context.Context, in RetrievePost) (Post, error) {
	var out Post

	if err := in.Validate(); err != nil {
		return out, err
	}

	user, _ := UserFromContext(ctx)
	in.authUserID = user.ID

	return svc.Store.Post(ctx, in)
}
