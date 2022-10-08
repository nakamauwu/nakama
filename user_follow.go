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
		return errs.Unauthenticated
	}

	if usr.ID == followedUserID {
		return ErrCannotFollowSelf
	}

	// TODO: run inside a transaction.

	_, err := svc.Queries.CreateUserFollow(ctx, CreateUserFollowParams{
		FollowerID: usr.ID,
		FollowedID: followedUserID,
	})
	if isPqUniqueViolationError(err) {
		// Early return if following already.
		// Note: Careful! As this query contains an error at the database layer.
		// Running this inside a transaction
		// will cause the entire transaction to fail.
		return nil
	}

	if isPqForeignKeyViolationError(err, "followed_id") {
		return ErrUserNotFound
	}

	if err != nil {
		return err
	}

	// Side-effect: increase user's follow counts on inserts
	// so we don't have to compute them on each read.

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

func (svc *Service) UnfollowUser(ctx context.Context, followedUserID string) error {
	if !isID(followedUserID) {
		return ErrInvalidUserID
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	if usr.ID == followedUserID {
		return ErrCannotFollowSelf
	}

	// TODO: run inside a transaction.

	exists, err := svc.Queries.UserExists(ctx, UserExistsParams{UserID: followedUserID})
	if err != nil {
		return err
	}

	if !exists {
		return ErrUserNotFound
	}

	exists, err = svc.Queries.UserFollowExists(ctx, UserFollowExistsParams{
		FollowerID: usr.ID,
		FollowedID: followedUserID,
	})
	if err != nil {
		return err
	}

	// Early return if not following already.
	if !exists {
		return nil
	}

	_, err = svc.Queries.DeleteUserFollow(ctx, DeleteUserFollowParams{
		FollowerID: usr.ID,
		FollowedID: followedUserID,
	})
	if err != nil {
		return err
	}

	// Side-effect: increase user's follow counts on inserts
	// so we don't have to compute them on each read.

	_, err = svc.Queries.UpdateUser(ctx, UpdateUserParams{
		UserID:                   usr.ID,
		IncreaseFollowingCountBy: -1,
	})
	if err != nil {
		return err
	}

	_, err = svc.Queries.UpdateUser(ctx, UpdateUserParams{
		UserID:                   followedUserID,
		IncreaseFollowersCountBy: -1,
	})
	return err
}
