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

type CreatePostInput struct {
	Content string
}

func (in *CreatePostInput) Prepare() {
	in.Content = strings.TrimSpace(in.Content)
	// TODO: fix post content sanitization not removing
	// duplicate spaces and line breaks properly.
	in.Content = strings.ReplaceAll(in.Content, "\n\n", "\n")
	in.Content = strings.ReplaceAll(in.Content, "  ", " ")
}

func (in CreatePostInput) Validate() error {
	if in.Content == "" || utf8.RuneCountInString(in.Content) > maxPostContentLength {
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
		return out, errs.ErrUnauthenticated
	}

	postID := genID()
	createdAt, err := svc.Queries.CreatePost(ctx, CreatePostParams{
		PostID:  postID,
		UserID:  usr.ID,
		Content: in.Content,
	})
	if err != nil {
		return out, err
	}

	// Side-effect: increase user's posts count on inserts
	// so we don't have to compute it on each read.
	_, err = svc.Queries.UpdateUser(ctx, UpdateUserParams{
		UserID:               usr.ID,
		IncreasePostsCountBy: 1,
	})
	if err != nil {
		return out, err
	}

	out.ID = postID
	out.CreateAt = createdAt

	return out, nil
}

func (svc *Service) Posts(ctx context.Context, in PostsInput) ([]PostsRow, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	return svc.Queries.Posts(ctx, in.Username)
}

func (svc *Service) Post(ctx context.Context, postID string) (PostRow, error) {
	var out PostRow

	if !isID(postID) {
		return out, ErrInvalidPostID
	}

	out, err := svc.Queries.Post(ctx, postID)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrPostNotFound
	}

	return out, err
}
