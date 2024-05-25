package service

import (
	"context"

	"github.com/nakamauwu/nakama/auth"
	"github.com/nakamauwu/nakama/types"
	"github.com/nicolasparada/go-errs"
)

func (svc *Service) CreatePost(ctx context.Context, in types.CreatePost) (types.Created, error) {
	var out types.Created

	if err := in.Validate(); err != nil {
		return out, err
	}

	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	in.UserID = user.ID
	return svc.Cockroach.CreatePost(ctx, in)
}

func (svc *Service) Posts(ctx context.Context, in types.ListPosts) (types.List[types.Post], error) {
	var out types.List[types.Post]

	if err := in.Validate(); err != nil {
		return out, err
	}

	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	in.UserID = user.ID

	return svc.Cockroach.Posts(ctx, in)
}
