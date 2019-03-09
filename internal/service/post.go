package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

var (
	// ErrInvalidContent denotes an invalid content.
	ErrInvalidContent = errors.New("invalid content")
	// ErrInvalidSpoiler denotes an invalid spoiler title.
	ErrInvalidSpoiler = errors.New("invalid spoiler")
	// ErrPostNotFound denotes a not found post.
	ErrPostNotFound = errors.New("post not found")
)

// Post model.
type Post struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"-"`
	Content       string    `json:"content"`
	SpoilerOf     *string   `json:"spoilerOf"`
	NSFW          bool      `json:"NSFW"`
	LikesCount    int       `json:"likesCount"`
	CommentsCount int       `json:"commentsCount"`
	CreatedAt     time.Time `json:"createdAt"`
	User          *User     `json:"user,omitempty"`
	Mine          bool      `json:"mine"`
	Liked         bool      `json:"liked"`
	Subscribed    bool      `json:"subscribed"`
}

// ToggleLikeOutput response.
type ToggleLikeOutput struct {
	Liked      bool `json:"liked"`
	LikesCount int  `json:"likesCount"`
}

// ToggleSubscriptionOutput response.
type ToggleSubscriptionOutput struct {
	Subscribed bool `json:"subscribed"`
}

// CreatePost publishes a post to the user timeline and fan-outs it to his followers.
func (s *Service) CreatePost(
	ctx context.Context,
	content string,
	spoilerOf *string,
	nsfw bool,
) (TimelineItem, error) {
	var ti TimelineItem
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return ti, ErrUnauthenticated
	}

	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 480 {
		return ti, ErrInvalidContent
	}

	if spoilerOf != nil {
		*spoilerOf = strings.TrimSpace(*spoilerOf)
		if *spoilerOf == "" || len([]rune(*spoilerOf)) > 64 {
			return ti, ErrInvalidSpoiler
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ti, fmt.Errorf("could not begin tx: %v", err)
	}

	defer tx.Rollback()

	var p Post
	query := `
		INSERT INTO posts (user_id, content, spoiler_of, nsfw) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	if err = tx.QueryRowContext(ctx, query, uid, content, spoilerOf, nsfw).
		Scan(&p.ID, &p.CreatedAt); err != nil {
		return ti, fmt.Errorf("could not insert post: %v", err)
	}

	p.UserID = uid
	p.Content = content
	p.SpoilerOf = spoilerOf
	p.NSFW = nsfw
	p.Mine = true

	query = "INSERT INTO post_subscriptions (user_id, post_id) VALUES ($1, $2)"
	if _, err = tx.ExecContext(ctx, query, uid, p.ID); err != nil {
		return ti, fmt.Errorf("could not insert post subscription: %v", err)
	}

	p.Subscribed = true

	query = "INSERT INTO timeline (user_id, post_id) VALUES ($1, $2) RETURNING id"
	if err = tx.QueryRowContext(ctx, query, uid, p.ID).Scan(&ti.ID); err != nil {
		return ti, fmt.Errorf("could not insert timeline item: %v", err)
	}

	ti.UserID = uid
	ti.PostID = p.ID
	ti.Post = &p

	if err = tx.Commit(); err != nil {
		return ti, fmt.Errorf("could not commit to create post: %v", err)
	}

	go s.postCreated(p)

	return ti, nil
}

func (s *Service) postCreated(p Post) {
	u, err := s.userByID(context.Background(), p.UserID)
	if err != nil {
		log.Printf("could not fetch post user: %v\n", err)
		return
	}

	p.User = &u
	p.Mine = false
	p.Subscribed = false

	go s.fanoutPost(p)
	go s.notifyPostMention(p)
}

// Posts from a user in descending order and with backward pagination.
func (s *Service) Posts(ctx context.Context, username string, last int, before int64) ([]Post, error) {
	username = strings.TrimSpace(username)
	if !rxUsername.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	uid, auth := ctx.Value(KeyAuthUserID).(int64)
	last = normalizePageSize(last)
	query, args, err := buildQuery(`
		SELECT id, content, spoiler_of, nsfw, likes_count, comments_count, created_at
		{{if .auth}}
		, posts.user_id = @uid AS mine
		, likes.user_id IS NOT NULL AS liked
		, subscriptions.user_id IS NOT NULL AS subscribed
		{{end}}
		FROM posts
		{{if .auth}}
		LEFT JOIN post_likes AS likes
			ON likes.user_id = @uid AND likes.post_id = posts.id
		LEFT JOIN post_subscriptions AS subscriptions
			ON subscriptions.user_id = @uid AND subscriptions.post_id = posts.id
		{{end}}
		WHERE posts.user_id = (SELECT id FROM users WHERE username = @username)
		{{if gt .before 0}}AND posts.id < @before{{end}}
		ORDER BY created_at DESC
		LIMIT @last`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
		"last":     last,
		"before":   before,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build posts sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select posts: %v", err)
	}

	defer rows.Close()

	pp := make([]Post, 0, last)
	for rows.Next() {
		var p Post
		dest := []interface{}{
			&p.ID,
			&p.Content,
			&p.SpoilerOf,
			&p.NSFW,
			&p.LikesCount,
			&p.CommentsCount,
			&p.CreatedAt,
		}
		if auth {
			dest = append(dest, &p.Mine, &p.Liked, &p.Subscribed)
		}

		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan post: %v", err)
		}

		pp = append(pp, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate posts rows: %v", err)
	}

	return pp, nil
}

// Post with the given ID.
func (s *Service) Post(ctx context.Context, postID int64) (Post, error) {
	var p Post
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
	query, args, err := buildQuery(`
		SELECT posts.id, content, spoiler_of, nsfw, likes_count, comments_count, created_at
		, users.username, users.avatar
		{{if .auth}}
		, posts.user_id = @uid AS mine
		, likes.user_id IS NOT NULL AS liked
		, subscriptions.user_id IS NOT NULL AS subscribed
		{{end}}
		FROM posts
		INNER JOIN users ON posts.user_id = users.id
		{{if .auth}}
		LEFT JOIN post_likes AS likes
			ON likes.user_id = @uid AND likes.post_id = posts.id
		LEFT JOIN post_subscriptions AS subscriptions
			ON subscriptions.user_id = @uid AND subscriptions.post_id = posts.id
		{{end}}
		WHERE posts.id = @post_id`, map[string]interface{}{
		"auth":    auth,
		"uid":     uid,
		"post_id": postID,
	})
	if err != nil {
		return p, fmt.Errorf("could not build post sql query: %v", err)
	}

	var u User
	var avatar sql.NullString
	dest := []interface{}{
		&p.ID,
		&p.Content,
		&p.SpoilerOf,
		&p.NSFW,
		&p.LikesCount,
		&p.CommentsCount,
		&p.CreatedAt,
		&u.Username,
		&avatar,
	}
	if auth {
		dest = append(dest, &p.Mine, &p.Liked, &p.Subscribed)
	}
	err = s.db.QueryRowContext(ctx, query, args...).Scan(dest...)
	if err == sql.ErrNoRows {
		return p, ErrPostNotFound
	}

	if err != nil {
		return p, fmt.Errorf("could not query select post: %v", err)
	}

	u.AvatarURL = s.avatarURL(avatar)
	p.User = &u

	return p, nil
}

// TogglePostLike 🖤
func (s *Service) TogglePostLike(ctx context.Context, postID int64) (ToggleLikeOutput, error) {
	var out ToggleLikeOutput
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return out, ErrUnauthenticated
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("could not begin tx: %v", err)
	}

	defer tx.Rollback()

	query := `
		SELECT EXISTS (
			SELECT 1 FROM post_likes WHERE user_id = $1 AND post_id = $2
		)`
	if err = tx.QueryRowContext(ctx, query, uid, postID).Scan(&out.Liked); err != nil {
		return out, fmt.Errorf("could not query select post like existence: %v", err)
	}

	if out.Liked {
		query = "DELETE FROM post_likes WHERE user_id = $1 AND post_id = $2"
		if _, err = tx.ExecContext(ctx, query, uid, postID); err != nil {
			return out, fmt.Errorf("could not delete post like: %v", err)
		}

		query = "UPDATE posts SET likes_count = likes_count - 1 WHERE id = $1 RETURNING likes_count"
		if err = tx.QueryRowContext(ctx, query, postID).Scan(&out.LikesCount); err != nil {
			return out, fmt.Errorf("could not update and decrement post likes count: %v", err)
		}
	} else {
		query = "INSERT INTO post_likes (user_id, post_id) VALUES ($1, $2)"
		_, err = tx.ExecContext(ctx, query, uid, postID)

		if isForeignKeyViolation(err) {
			return out, ErrPostNotFound
		}

		if err != nil {
			return out, fmt.Errorf("could not insert post like: %v", err)
		}

		query = "UPDATE posts SET likes_count = likes_count + 1 WHERE id = $1 RETURNING likes_count"
		if err = tx.QueryRowContext(ctx, query, postID).Scan(&out.LikesCount); err != nil {
			return out, fmt.Errorf("could not update and increment post likes count: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("could not commit to toggle post like: %v", err)
	}

	out.Liked = !out.Liked

	return out, nil
}

// TogglePostSubscription so you can stop receiving notifications from a thread.
func (s *Service) TogglePostSubscription(ctx context.Context, postID int64) (ToggleSubscriptionOutput, error) {
	var out ToggleSubscriptionOutput
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return out, ErrUnauthenticated
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("could not begin tx: %v", err)
	}

	defer tx.Rollback()

	query := `SELECT EXISTS (
		SELECT 1 FROM post_subscriptions WHERE user_id = $1 AND post_id = $2
	)`
	if err = tx.QueryRowContext(ctx, query, uid, postID).Scan(&out.Subscribed); err != nil {
		return out, fmt.Errorf("could not query select post subscription existence: %v", err)
	}

	if out.Subscribed {
		query = "DELETE FROM post_subscriptions WHERE user_id = $1 AND post_id = $2"
		if _, err = tx.ExecContext(ctx, query, uid, postID); err != nil {
			return out, fmt.Errorf("could not delete post subscription: %v", err)
		}
	} else {
		query = "INSERT INTO post_subscriptions (user_id, post_id) VALUES ($1, $2)"
		_, err = tx.ExecContext(ctx, query, uid, postID)
		if isForeignKeyViolation(err) {
			return out, ErrPostNotFound
		}

		if err != nil {
			return out, fmt.Errorf("could not insert post subscription: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("could not commit to toggle post subscription: %v", err)
	}

	out.Subscribed = !out.Subscribed

	return out, nil
}
