package nakama

import (
	"context"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrCannotFollowSelf = errs.ConflictError("cannot follow self")
)

func (svc *Service) FollowUser(ctx context.Context, followUserID string) error {
	if !isID(followUserID) {
		return ErrInvalidUserID
	}

	usr, ok := UserFromContext(ctx)
	if !ok {
		return errs.ErrUnauthenticated
	}

	if usr.ID == followUserID {
		return ErrCannotFollowSelf
	}

	exists, err := svc.Queries.UserFollowExists(ctx, UserFollowExistsParams{
		FollowerID: usr.ID,
		FollowedID: followUserID,
	})
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	exists, err = svc.Queries.UserExists(ctx, followUserID)
	if err != nil {
		return err
	}

	if !exists {
		return ErrUserNotFound
	}

	_, err = svc.Queries.CreateUserFollow(ctx, CreateUserFollowParams{
		FollowerID: usr.ID,
		FollowedID: followUserID,
	})
	if err != nil {
		return err
	}

	_, err = svc.Queries.UpdateUser(ctx, UpdateUserParams{
		UserID:                   usr.ID,
		IncreaseFollowingCountBy: 1,
	})
	if err != nil {
		return err
	}

	_, err = svc.Queries.UpdateUser(ctx, UpdateUserParams{
		UserID:                   followUserID,
		IncreaseFollowersCountBy: 1,
	})
	if err != nil {
		return err
	}

	return nil
}
