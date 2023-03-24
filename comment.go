package nakama

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrInvalidCommentID      = errs.InvalidArgumentError("invalid comment ID")
	ErrInvalidCommentContent = errs.InvalidArgumentError("invalid comment content")
	ErrCommentNotFound       = errs.NotFoundError("comment not found")
)

const maxCommentContentLength = 1000

type Comment struct {
	ID             string
	UserID         string
	PostID         string
	Content        string
	ReactionsCount ReactionsCount
	CreatedAt      time.Time
	UpdatedAt      time.Time
	User           UserPreview
}

type CreateComment struct {
	PostID  string
	Content string

	userID string
}

func (in *CreateComment) Validate() error {
	in.PostID = strings.TrimSpace(in.PostID)
	in.Content = smartTrim(in.Content)

	if !validID(in.PostID) {
		return ErrInvalidPostID
	}

	if in.Content == "" || !utf8.ValidString(in.Content) || utf8.RuneCountInString(in.Content) > maxCommentContentLength {
		return ErrInvalidCommentContent
	}

	return nil
}

type ListComments struct {
	PostID string

	authUserID string
}

func (in *ListComments) Validate() error {
	in.PostID = strings.TrimSpace(in.PostID)

	if !validID(in.PostID) {
		return ErrInvalidPostID
	}

	return nil
}

type RetrieveComment struct {
	ID string

	authUserID string
}

func (in *RetrieveComment) Validate() error {
	in.ID = strings.TrimSpace(in.ID)

	if !validID(in.ID) {
		return ErrInvalidCommentID
	}

	return nil
}

type UpdateComment struct {
	ID             string
	ReactionsCount *ReactionsCount
}

func (svc *Service) CreateComment(ctx context.Context, in CreateComment) (Created, error) {
	var out Created

	if err := in.Validate(); err != nil {
		return out, err
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	in.userID = user.ID

	return out, svc.Store.RunTx(ctx, func(ctx context.Context) error {
		var err error
		out, err = svc.Store.CreateComment(ctx, in)
		if err != nil {
			return err
		}

		// Side effect: increase post's comments count on inserts,
		// so we don't have to compute it on each read.
		_, err = svc.Store.UpdatePost(ctx, UpdatePost{
			PostID:                  in.PostID,
			IncreaseCommentsCountBy: 1,
		})
		return err
	})
}

func (svc *Service) Comments(ctx context.Context, in ListComments) ([]Comment, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	user, _ := UserFromContext(ctx)
	in.authUserID = user.ID

	return svc.Store.Comments(ctx, in)
}

func (svc *Service) Comment(ctx context.Context, in RetrieveComment) (Comment, error) {
	var out Comment

	if err := in.Validate(); err != nil {
		return out, err
	}

	user, _ := UserFromContext(ctx)
	in.authUserID = user.ID

	return svc.Store.Comment(ctx, in)
}
