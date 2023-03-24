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
	if !validID(followedUserID) {
		return ErrInvalidUserID
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	if user.ID == followedUserID {
		return ErrCannotFollowSelf
	}

	follow := UserFollow{
		FollowerID: user.ID,
		FollowedID: followedUserID,
	}

	return svc.Store.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.Store.UserFollowExists(ctx, follow)
		if err != nil {
			return err
		}

		if exists {
			// Early return if following already.
			return nil
		}

		_, err = svc.Store.CreateUserFollow(ctx, follow)
		if err != nil {
			return err
		}

		// Side effect: increase user's follow counts on inserts,
		// so we don't have to compute them on each read.

		_, err = svc.Store.UpdateUser(ctx, UpdateUser{
			userID:                   user.ID,
			increaseFollowingCountBy: 1,
		})
		if err != nil {
			return err
		}

		_, err = svc.Store.UpdateUser(ctx, UpdateUser{
			userID:                   followedUserID,
			increaseFollowersCountBy: 1,
		})
		return err
	})
}

func (svc *Service) UnfollowUser(ctx context.Context, followedUserID string) error {
	if !validID(followedUserID) {
		return ErrInvalidUserID
	}

	user, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	if user.ID == followedUserID {
		return ErrCannotFollowSelf
	}

	follow := UserFollow{
		FollowerID: user.ID,
		FollowedID: followedUserID,
	}

	return svc.Store.RunTx(ctx, func(ctx context.Context) error {
		exists, err := svc.Store.UserExists(ctx, RetrieveUserExists{UserID: followedUserID})
		if err != nil {
			return err
		}

		if !exists {
			return ErrUserNotFound
		}

		exists, err = svc.Store.UserFollowExists(ctx, follow)
		if err != nil {
			return err
		}

		// Early return if not following already.
		if !exists {
			return nil
		}

		_, err = svc.Store.DeleteUserFollow(ctx, follow)
		if err != nil {
			return err
		}

		// Side effect: increase user's follow counts on inserts,
		// so we don't have to compute them on each read.

		_, err = svc.Store.UpdateUser(ctx, UpdateUser{
			userID:                   user.ID,
			increaseFollowingCountBy: -1,
		})
		if err != nil {
			return err
		}

		_, err = svc.Store.UpdateUser(ctx, UpdateUser{
			userID:                   followedUserID,
			increaseFollowersCountBy: -1,
		})
		return err
	})
}
