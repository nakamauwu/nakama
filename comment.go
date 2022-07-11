package nakama

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrInvalidCommentContent = errs.InvalidArgumentError("invalid comment content")
)

const maxCommentContentLength = 1000

type CreateCommentInput struct {
	PostID  string
	Content string
}

func (in *CreateCommentInput) Prepare() {
	in.Content = strings.TrimSpace(in.Content)
	in.Content = strings.ReplaceAll(in.Content, "\n\n", "\n")
	in.Content = strings.ReplaceAll(in.Content, "  ", " ")
}

func (in CreateCommentInput) Validate() error {
	if !isID(in.PostID) {
		return ErrInvalidPostID
	}
	if in.Content == "" || utf8.RuneCountInString(in.Content) > maxCommentContentLength {
		return ErrInvalidCommentContent
	}
	return nil
}

type CreateCommentOutput struct {
	ID        string
	CreatedAt time.Time
}

func (svc *Service) CreateComment(ctx context.Context, in CreateCommentInput) (CreateCommentOutput, error) {
	var out CreateCommentOutput

	in.Prepare()
	if err := in.Validate(); err != nil {
		return out, err
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return out, errs.ErrUnauthenticated
	}

	// TODO: run inside a transaction.

	// Note: maybe check foreign key constraint violation
	// error returned by `Queries.CreateComment`.
	exists, err := svc.Queries.PostExists(ctx, in.PostID)
	if err != nil {
		return out, err
	}

	if !exists {
		return out, ErrPostNotFound
	}

	commentID := genID()
	createdAt, err := svc.Queries.CreateComment(ctx, CreateCommentParams{
		CommentID: commentID,
		PostID:    in.PostID,
		UserID:    usr.ID,
		Content:   in.Content,
	})
	if err != nil {
		return out, err
	}

	_, err = svc.Queries.UpdatePost(ctx, UpdatePostParams{
		PostID:                  in.PostID,
		IncreaseCommentsCountBy: 1,
	})
	if err != nil {
		return out, err
	}

	out.ID = commentID
	out.CreatedAt = createdAt

	return out, nil
}

func (svc *Service) Comments(ctx context.Context, postID string) ([]CommentsRow, error) {
	if !isID(postID) {
		return nil, ErrInvalidPostID
	}
	return svc.Queries.Comments(ctx, postID)
}
