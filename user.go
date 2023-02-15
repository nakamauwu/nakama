package nakama

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/nicolasparada/go-errs"
)

const (
	ErrUserNotFound    = errs.NotFoundError("user not found")
	ErrUsernameTaken   = errs.ConflictError("username taken")
	ErrInvalidUserID   = errs.InvalidArgumentError("invalid user ID")
	ErrInvalidEmail    = errs.InvalidArgumentError("invalid email")
	ErrInvalidUsername = errs.InvalidArgumentError("invalid username")
)

const (
	avatarWidth  = uint(400)
	avatarHeight = uint(400)
)

var (
	reEmail         = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reUsername      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,17}$`)
	reUsernameQuery = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,18}$`)
)

type User struct {
	ID             string
	Email          string
	Username       string
	AvatarPath     *string
	AvatarWidth    *uint
	AvatarHeight   *uint
	PostsCount     int32
	FollowersCount int32
	FollowingCount int32
	Following      bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserPreview struct {
	Username     string
	AvatarPath   *string
	AvatarWidth  *uint
	AvatarHeight *uint
}

type CreateUser struct {
	Email    string
	Username string
}

type CreatedUser struct {
	ID        string
	CreatedAt time.Time
}

type UsersParams struct {
	UsernameQuery string

	authUserID string
}

func (in *UsersParams) Validate() error {
	in.UsernameQuery = strings.ToLower(in.UsernameQuery)
	in.UsernameQuery = strings.TrimSpace(in.UsernameQuery)

	if in.UsernameQuery != "" && !validUsernameQuery(in.UsernameQuery) {
		return ErrInvalidUsername
	}

	return nil
}

type RetrieveUser struct {
	FollowerID string
	UserID     string
	Email      string
	Username   string
}

type UserExistsParams struct {
	UserID   string
	Email    string
	Username string
}

type UpdateUser struct {
	Username *string

	avatarPath               *string
	avatarWidth              *uint
	avatarHeight             *uint
	increasePostsCountBy     int
	increaseFollowersCountBy int
	increaseFollowingCountBy int
	userID                   string
}

func (in *UpdateUser) Validate() error {
	if in.Username != nil {
		*in.Username = strings.TrimSpace(*in.Username)
		if !validUsername(*in.Username) {
			return ErrInvalidUsername
		}
	}

	return nil
}

type UpdatedAvatar struct {
	Path   string
	Width  uint
	Height uint
}

func (svc *Service) Users(ctx context.Context, in UsersParams) ([]User, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	user, _ := UserFromContext(ctx)
	in.authUserID = user.ID

	return svc.Store.Users(ctx, in)
}

func (svc *Service) User(ctx context.Context, username string) (User, error) {
	var out User

	user, authenticated := UserFromContext(ctx)

	q := RetrieveUser{FollowerID: user.ID}

	if username == "" {
		if !authenticated {
			return out, errs.Unauthenticated
		}

		q.UserID = user.ID
	} else {
		if !validUsername(username) {
			return out, ErrInvalidUsername
		}

		q.Username = username
	}

	return svc.Store.User(ctx, q)
}

func (svc *Service) UpdateUser(ctx context.Context, in UpdateUser) error {
	user, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	if err := in.Validate(); err != nil {
		return err
	}

	_, err := svc.Store.UpdateUser(ctx, UpdateUser{
		userID:   user.ID,
		Username: in.Username,
	})

	return err
}

func (svc *Service) UpdateAvatar(ctx context.Context, avatar io.Reader) (UpdatedAvatar, error) {
	var out UpdatedAvatar

	user, ok := UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	resized, err := fillJPEG(avatar, avatarWidth, avatarHeight)
	if err != nil {
		return out, err
	}

	now := time.Now().UTC()
	path := fmt.Sprintf("%d/%d/%d/%s-%s.jpeg", now.Year(), now.Month(), now.Day(), user.ID, genID())
	err = svc.s3StoreObject(ctx, s3StoreObject{
		File:        bytes.NewReader(resized),
		Bucket:      S3BucketAvatars,
		Name:        path,
		Size:        uint64(len(resized)),
		ContentType: "image/jpeg",
	})
	if err != nil {
		return out, err
	}

	_, err = svc.Store.UpdateUser(ctx, UpdateUser{
		avatarPath:   &path,
		avatarWidth:  ptr(avatarWidth),
		avatarHeight: ptr(avatarHeight),
		userID:       user.ID,
	})
	if err != nil {
		errS3 := svc.s3RemoveObject(ctx, s3RemoveObject{
			Bucket: S3BucketAvatars,
			Name:   path,
		})
		if errS3 != nil {
			svc.Logger.Printf("could not remove avatar after user update failure: %v\n", errS3)
		}

		return out, err
	}

	out.Path = svc.AvatarsPrefix + path
	out.Width = avatarWidth
	out.Height = avatarHeight

	return out, nil
}

func validEmail(s string) bool {
	return reEmail.MatchString(s)
}

func validUsername(s string) bool {
	return reUsername.MatchString(s)
}

func validUsernameQuery(s string) bool {
	return reUsernameQuery.MatchString(s)
}
