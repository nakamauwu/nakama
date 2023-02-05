package nakama

import (
	"context"

	"github.com/nicolasparada/go-errs"
)

type AddPostReaction struct {
	PostID   string
	Reaction string
}

func (in *AddPostReaction) Validate() error {
	if !validID(in.PostID) {
		return ErrInvalidPostID
	}

	if !validEmoji(in.Reaction) {
		return ErrInvalidEmoji
	}

	return nil
}

func (svc *Service) AddPostReaction(ctx context.Context, in AddPostReaction) error {
	if err := in.Validate(); err != nil {
		return err
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	return svc.DB.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.sqlSelectPostReactionExistence(ctx, sqlSelectPostReactionExistence{
			UserID:   usr.ID,
			PostID:   in.PostID,
			Reaction: in.Reaction,
		})
		if err != nil {
			return err
		}

		if exists {
			return nil
		}

		err = svc.sqlInsertPostReaction(ctx, sqlInsertPostReaction{
			UserID:   usr.ID,
			PostID:   in.PostID,
			Reaction: in.Reaction,
		})
		if err != nil {
			return err
		}

		reactionsCount, err := svc.sqlSelectPostReactionsCount(ctx, in.PostID)
		if err != nil {
			return err
		}

		reactionsCount.Inc(in.Reaction)

		_, err = svc.sqlUpdatePost(ctx, sqlUpdatePost{
			PostID:         in.PostID,
			ReactionsCount: &reactionsCount,
		})
		return err
	})
}
