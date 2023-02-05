package nakama

import (
	"context"

	"github.com/nicolasparada/go-errs"
)

type AddCommentReaction struct {
	CommentID string
	Reaction  string
}

func (in *AddCommentReaction) Validate() error {
	if !validID(in.CommentID) {
		return ErrInvalidCommentID
	}

	if !validEmoji(in.Reaction) {
		return ErrInvalidEmoji
	}

	return nil
}

type RemoveCommentReaction = AddCommentReaction

func (svc *Service) AddCommentReaction(ctx context.Context, in AddCommentReaction) error {
	if err := in.Validate(); err != nil {
		return err
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	return svc.DB.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.sqlSelectCommentReactionExistence(ctx, sqlSelectCommentReactionExistence{
			UserID:    usr.ID,
			CommentID: in.CommentID,
			Reaction:  in.Reaction,
		})
		if err != nil {
			return err
		}

		if exists {
			return nil
		}

		err = svc.sqlInsertCommentReaction(ctx, sqlInsertCommentReaction{
			UserID:    usr.ID,
			CommentID: in.CommentID,
			Reaction:  in.Reaction,
		})
		if err != nil {
			return err
		}

		reactionsCount, err := svc.sqlSelectCommentReactionsCount(ctx, in.CommentID)
		if err != nil {
			return err
		}

		reactionsCount.Inc(in.Reaction)

		_, err = svc.sqlUpdateComment(ctx, sqlUpdateComment{
			CommentID:      in.CommentID,
			ReactionsCount: &reactionsCount,
		})
		return err
	})
}

func (svc *Service) RemoveCommentReaction(ctx context.Context, in RemoveCommentReaction) error {
	if err := in.Validate(); err != nil {
		return err
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	return svc.DB.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.sqlSelectCommentReactionExistence(ctx, sqlSelectCommentReactionExistence{
			UserID:    usr.ID,
			CommentID: in.CommentID,
			Reaction:  in.Reaction,
		})
		if err != nil {
			return err
		}

		if !exists {
			return nil
		}

		err = svc.sqlDeleteCommentReaction(ctx, sqlDeleteCommentReaction{
			UserID:    usr.ID,
			CommentID: in.CommentID,
			Reaction:  in.Reaction,
		})
		if err != nil {
			return err
		}

		reactionsCount, err := svc.sqlSelectCommentReactionsCount(ctx, in.CommentID)
		if err != nil {
			return err
		}

		reactionsCount.Dec(in.Reaction)

		_, err = svc.sqlUpdateComment(ctx, sqlUpdateComment{
			CommentID:      in.CommentID,
			ReactionsCount: &reactionsCount,
		})
		return err
	})
}
