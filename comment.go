package nakama

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrInvalidCommentContent = errs.InvalidArgumentError("invalid comment content")
)

const maxCommentContentLength = 1000

type Comment struct {
	ID        string
	UserID    string
	PostID    string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	User      UserPreview
}

type CreateCommentInput struct {
	PostID  string
	Content string
}

func (in *CreateCommentInput) Prepare() {
	in.Content = smartTrim(in.Content)
}

func (in CreateCommentInput) Validate() error {
	if !isID(in.PostID) {
		return ErrInvalidPostID
	}
	if in.Content == "" || !utf8.ValidString(in.Content) || utf8.RuneCountInString(in.Content) > maxCommentContentLength {
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
		return out, errs.Unauthenticated
	}

	return out, svc.DB.RunTx(ctx, func(ctx context.Context) error {
		commentID := genID()
		createdAt, err := svc.sqlInsertComment(ctx, sqlInsertComment{
			CommentID: commentID,
			PostID:    in.PostID,
			UserID:    usr.ID,
			Content:   in.Content,
		})
		if isPqForeignKeyViolationError(err, "post_id") {
			return ErrPostNotFound
		}

		if err != nil {
			return err
		}

		// Side-effect: increase post's comments count on inserts
		// so we don't have to compute it on each read.
		_, err = svc.sqlUpdatePost(ctx, sqlUpdatePost{
			PostID:                  in.PostID,
			IncreaseCommentsCountBy: 1,
		})
		if err != nil {
			return err
		}

		out.ID = commentID
		out.CreatedAt = createdAt

		return nil
	})
}

func (svc *Service) Comments(ctx context.Context, postID string) ([]Comment, error) {
	if !isID(postID) {
		return nil, ErrInvalidPostID
	}
	return svc.sqlSelectComments(ctx, postID)
}
