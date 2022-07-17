package nakama

import (
	"context"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrCannotFollowSelf = errs.ConflictError("cannot follow self")
)

func (svc *Service) FollowUser(ctx context.Context, followedUserID string) error {
	if !isID(followedUserID) {
		return ErrInvalidUserID
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return errs.ErrUnauthenticated
	}

	if usr.ID == followedUserID {
		return ErrCannotFollowSelf
	}

	// TODO: run inside a transaction.

	// Note: maybe check unique violation
	// error returned by `Queries.CreateUserFollow`.

	exists, err := svc.Queries.UserFollowExists(ctx, UserFollowExistsParams{
		FollowerID: usr.ID,
		FollowedID: followedUserID,
	})
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	exists, err = svc.Queries.UserExists(ctx, followedUserID)
	if err != nil {
		return err
	}

	if !exists {
		return ErrUserNotFound
	}

	_, err = svc.Queries.CreateUserFollow(ctx, CreateUserFollowParams{
		FollowerID: usr.ID,
		FollowedID: followedUserID,
	})
	if err != nil {
		return err
	}

	// Side-effect: increase user's follow count on inserts
	// so we don't have to compute it on each read.

	_, err = svc.Queries.UpdateUser(ctx, UpdateUserParams{
		UserID:                   usr.ID,
		IncreaseFollowingCountBy: 1,
	})
	if err != nil {
		return err
	}

	_, err = svc.Queries.UpdateUser(ctx, UpdateUserParams{
		UserID:                   followedUserID,
		IncreaseFollowersCountBy: 1,
	})
	return err
}
