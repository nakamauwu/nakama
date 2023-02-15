package nakama

import (
	"context"

	"github.com/nicolasparada/go-errs"
)

type CommentReaction struct {
	CommentID string
	Reaction  string

	userID string
}

func (in *CommentReaction) Validate() error {
	if !validID(in.CommentID) {
		return ErrInvalidCommentID
	}

	if !validEmoji(in.Reaction) {
		return ErrInvalidEmoji
	}

	return nil
}

func (svc *Service) CreateCommentReaction(ctx context.Context, in CommentReaction) error {
	if err := in.Validate(); err != nil {
		return err
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	in.userID = user.ID

	return svc.Store.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.Store.CommentReactionExists(ctx, in)
		if err != nil {
			return err
		}

		if exists {
			return nil
		}

		err = svc.Store.CreateCommentReaction(ctx, in)
		if err != nil {
			return err
		}

		reactionsCount, err := svc.Store.CommentReactionsCount(ctx, in.CommentID)
		if err != nil {
			return err
		}

		reactionsCount.Inc(in.Reaction)

		_, err = svc.Store.UpdateComment(ctx, UpdateComment{
			CommentID:      in.CommentID,
			ReactionsCount: &reactionsCount,
		})
		return err
	})
}

func (svc *Service) DeleteCommentReaction(ctx context.Context, in CommentReaction) error {
	if err := in.Validate(); err != nil {
		return err
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	in.userID = user.ID

	return svc.Store.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.Store.CommentReactionExists(ctx, in)
		if err != nil {
			return err
		}

		if !exists {
			return nil
		}

		err = svc.Store.DeleteCommentReaction(ctx, in)
		if err != nil {
			return err
		}

		reactionsCount, err := svc.Store.CommentReactionsCount(ctx, in.CommentID)
		if err != nil {
			return err
		}

		reactionsCount.Dec(in.Reaction)

		_, err = svc.Store.UpdateComment(ctx, UpdateComment{
			CommentID:      in.CommentID,
			ReactionsCount: &reactionsCount,
		})
		return err
	})
}
