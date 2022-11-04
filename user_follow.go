package nakama

import (
	"context"
	"time"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrCannotFollowSelf = errs.ConflictError("cannot follow self")
)

type UserFollow struct {
	FollowerID string
	FollowedID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

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

	_, err := svc.sqlInsertUserFollow(ctx, sqlInsertUserFollow{
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

	_, err = svc.sqlUpdateUser(ctx, sqlUpdateUser{
		UserID:                   usr.ID,
		IncreaseFollowingCountBy: 1,
	})
	if err != nil {
		return err
	}

	_, err = svc.sqlUpdateUser(ctx, sqlUpdateUser{
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

	exists, err := svc.sqlSelectUserExists(ctx, sqlSelectUserExists{UserID: followedUserID})
	if err != nil {
		return err
	}

	if !exists {
		return ErrUserNotFound
	}

	exists, err = svc.sqlSelectUserFollowExists(ctx, sqlSelectUserFollowExists{
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

	_, err = svc.sqlDeleteUserFollow(ctx, sqlDeleteUserFollow{
		FollowerID: usr.ID,
		FollowedID: followedUserID,
	})
	if err != nil {
		return err
	}

	// Side-effect: increase user's follow counts on inserts
	// so we don't have to compute them on each read.

	_, err = svc.sqlUpdateUser(ctx, sqlUpdateUser{
		UserID:                   usr.ID,
		IncreaseFollowingCountBy: -1,
	})
	if err != nil {
		return err
	}

	_, err = svc.sqlUpdateUser(ctx, sqlUpdateUser{
		UserID:                   followedUserID,
		IncreaseFollowersCountBy: -1,
	})
	return err
}
