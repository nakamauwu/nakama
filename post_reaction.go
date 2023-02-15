package nakama

import (
	"context"

	"github.com/nicolasparada/go-errs"
)

type PostReaction struct {
	PostID   string
	Reaction string

	userID string
}

func (in *PostReaction) Validate() error {
	if !validID(in.PostID) {
		return ErrInvalidPostID
	}

	if !validEmoji(in.Reaction) {
		return ErrInvalidEmoji
	}

	return nil
}

func (svc *Service) CreatePostReaction(ctx context.Context, in PostReaction) error {
	if err := in.Validate(); err != nil {
		return err
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	in.userID = user.ID

	return svc.Store.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.Store.PostReactionExists(ctx, in)
		if err != nil {
			return err
		}

		if exists {
			return nil
		}

		err = svc.Store.CreatePostReaction(ctx, in)
		if err != nil {
			return err
		}

		reactionsCount, err := svc.Store.PostReactionsCount(ctx, in.PostID)
		if err != nil {
			return err
		}

		reactionsCount.Inc(in.Reaction)

		_, err = svc.Store.UpdatePost(ctx, UpdatePost{
			PostID:         in.PostID,
			ReactionsCount: &reactionsCount,
		})
		return err
	})
}

func (svc *Service) DeletePostReaction(ctx context.Context, in PostReaction) error {
	if err := in.Validate(); err != nil {
		return err
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	in.userID = user.ID

	return svc.Store.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.Store.PostReactionExists(ctx, in)
		if err != nil {
			return err
		}

		if !exists {
			return nil
		}

		err = svc.Store.DeletePostReaction(ctx, in)
		if err != nil {
			return err
		}

		reactionsCount, err := svc.Store.PostReactionsCount(ctx, in.PostID)
		if err != nil {
			return err
		}

		reactionsCount.Dec(in.Reaction)

		_, err = svc.Store.UpdatePost(ctx, UpdatePost{
			PostID:         in.PostID,
			ReactionsCount: &reactionsCount,
		})
		return err
	})
}
