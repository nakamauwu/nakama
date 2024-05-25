package types

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nicolasparada/go-errs"
)

type Post struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type CreatePost struct {
	UserID  string `form:"-"`
	Content string `form:"content"`
}

func (in *CreatePost) Validate() error {
	in.UserID = strings.TrimSpace(in.UserID)
	if in.UserID != "" {
		return fmt.Errorf("create post with user ID from request")
	}

	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" {
		return errs.InvalidArgumentError("content is required")
	}

	if utf8.RuneCountInString(in.Content) > 500 {
		return errs.InvalidArgumentError("content too long")
	}

	return nil
}

type ListPosts struct {
	UserID string `form:"-"`
}

func (in *ListPosts) Validate() error {
	in.UserID = strings.TrimSpace(in.UserID)
	if in.UserID != "" {
		return fmt.Errorf("list posts with user ID from request")
	}

	return nil
}
