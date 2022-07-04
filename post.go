package nakama

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nicolasparada/go-errs"
)

type CreatePostInput struct {
	Content string
}

func (in *CreatePostInput) Prepare() {
	in.Content = strings.TrimSpace(in.Content)
	in.Content = strings.ReplaceAll(in.Content, "\n\n", "\n")
	in.Content = strings.ReplaceAll(in.Content, "  ", " ")
}

func (in CreatePostInput) Validate() error {
	if in.Content == "" || utf8.RuneCountInString(in.Content) > 1000 {
		return errs.InvalidArgumentError("invalid post content")
	}
	return nil
}

type CreatePostOutput struct {
	ID       string
	CreateAt time.Time
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

	out.ID = postID
	out.CreateAt = createdAt

	return out, nil
}
