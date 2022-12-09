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
	reEmail    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,17}$`)
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

type UpdateUser struct {
	Username *string
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

func (svc *Service) User(ctx context.Context, username string) (User, error) {
	var out User

	usr, authenticated := UserFromContext(ctx)

	q := sqlSelectUser{FollowerID: usr.ID}

	if username == "" {
		if !authenticated {
			return out, errs.Unauthenticated
		}

		q.UserID = usr.ID
	} else {
		if !validUsername(username) {
			return out, ErrInvalidUsername
		}

		q.Username = username
	}

	return svc.sqlSelectUser(ctx, q)
}

func (svc *Service) UpdateUser(ctx context.Context, in UpdateUser) error {
	usr, ok := UserFromContext(ctx)
	if !ok {
		return errs.Unauthenticated
	}

	if err := in.Validate(); err != nil {
		return err
	}

	_, err := svc.sqlUpdateUser(ctx, sqlUpdateUser{
		UserID:   usr.ID,
		Username: in.Username,
	})

	return err
}

func (svc *Service) UpdateAvatar(ctx context.Context, r io.Reader) (UpdatedAvatar, error) {
	var out UpdatedAvatar

	usr, ok := UserFromContext(ctx)
	if !ok {
		return out, errs.Unauthenticated
	}

	resized, err := fillJPEG(r, avatarWidth, avatarHeight)
	if err != nil {
		return out, err
	}

	now := time.Now().UTC().Truncate(time.Second)

	name := fmt.Sprintf("%d/%d/%d/%s-%s.jpeg", now.Year(), now.Month(), now.Day(), usr.ID, genID())
	err = svc.s3StoreObject(ctx, s3StoreObject{
		File:        bytes.NewReader(resized),
		Bucket:      S3BucketAvatars,
		Name:        name,
		Size:        uint64(len(resized)),
		ContentType: "image/jpeg",
		Width:       avatarWidth,
		Height:      avatarHeight,
	})
	if err != nil {
		return out, err
	}

	_, err = svc.sqlUpdateUser(ctx, sqlUpdateUser{
		AvatarPath:   &name,
		AvatarWidth:  ptr(avatarWidth),
		AvatarHeight: ptr(avatarHeight),
		UserID:       usr.ID,
	})
	if err != nil {
		// TODO: delete object from s3 in case of error.
		return out, err
	}

	out.Path = svc.AvatarsPrefix + name
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
