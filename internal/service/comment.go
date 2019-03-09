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

// ErrCommentNotFound denotes a not found comment.
var ErrCommentNotFound = errors.New("comment not found")

// Comment model.
type Comment struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"-"`
	PostID     int64     `json:"-"`
	Content    string    `json:"content"`
	LikesCount int       `json:"likesCount"`
	CreatedAt  time.Time `json:"createdAt"`
	User       *User     `json:"user,omitempty"`
	Mine       bool      `json:"mine"`
	Liked      bool      `json:"liked"`
}

type commentClient struct {
	comments chan Comment
	postID   int64
	userID   *int64
}

// CreateComment on a post.
func (s *Service) CreateComment(ctx context.Context, postID int64, content string) (Comment, error) {
	var c Comment
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return c, ErrUnauthenticated
	}

	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 480 {
		return c, ErrInvalidContent
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return c, fmt.Errorf("could not begin tx: %v", err)
	}

	defer tx.Rollback()

	query := `
		INSERT INTO comments (user_id, post_id, content) VALUES ($1, $2, $3)
		RETURNING id, created_at`
	err = tx.QueryRowContext(ctx, query, uid, postID, content).Scan(&c.ID, &c.CreatedAt)
	if isForeignKeyViolation(err) {
		return c, ErrPostNotFound
	}

	if err != nil {
		return c, fmt.Errorf("could not insert comment: %v", err)
	}

	c.UserID = uid
	c.PostID = postID
	c.Content = content
	c.Mine = true

	query = `
		INSERT INTO post_subscriptions (user_id, post_id) VALUES ($1, $2)
		ON CONFLICT (user_id, post_id) DO NOTHING`
	if _, err = tx.ExecContext(ctx, query, uid, postID); err != nil {
		return c, fmt.Errorf("could not insert post subcription after commenting: %v", err)
	}

	query = "UPDATE posts SET comments_count = comments_count + 1 WHERE id = $1"
	if _, err = tx.ExecContext(ctx, query, postID); err != nil {
		return c, fmt.Errorf("could not update and increment post comments count: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return c, fmt.Errorf("could not commit to create comment: %v", err)
	}

	go s.commentCreated(c)

	return c, nil
}

func (s *Service) commentCreated(c Comment) {
	u, err := s.userByID(context.Background(), c.UserID)
	if err != nil {
		log.Printf("could not fetch comment user: %v\n", err)
		return
	}

	c.User = &u
	c.Mine = false

	go s.notifyComment(c)
	go s.notifyCommentMention(c)
	go s.broadcastComment(c)
}

// Comments from a post in descending order with backward pagination.
func (s *Service) Comments(ctx context.Context, postID int64, last int, before int64) ([]Comment, error) {
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
	last = normalizePageSize(last)
	query, args, err := buildQuery(`
		SELECT comments.id, content, likes_count, created_at, username, avatar
		{{if .auth}}
		, comments.user_id = @uid AS mine
		, likes.user_id IS NOT NULL AS liked
		{{end}}
		FROM comments
		INNER JOIN users ON comments.user_id = users.id
		{{if .auth}}
		LEFT JOIN comment_likes AS likes
			ON likes.comment_id = comments.id AND likes.user_id = @uid
		{{end}}
		WHERE comments.post_id = @post_id
		{{if gt .before 0}}AND comments.id < @before{{end}}
		ORDER BY created_at DESC
		LIMIT @last`, map[string]interface{}{
		"auth":    auth,
		"uid":     uid,
		"post_id": postID,
		"before":  before,
		"last":    last,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build comments sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select comments: %v", err)
	}

	defer rows.Close()

	cc := make([]Comment, 0, last)
	for rows.Next() {
		var c Comment
		var u User
		var avatar sql.NullString
		dest := []interface{}{&c.ID, &c.Content, &c.LikesCount, &c.CreatedAt, &u.Username, &avatar}
		if auth {
			dest = append(dest, &c.Mine, &c.Liked)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan comment: %v", err)
		}

		u.AvatarURL = s.avatarURL(avatar)
		c.User = &u
		cc = append(cc, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate comment rows: %v", err)
	}

	return cc, nil
}

// CommentSubscription to receive comments in realtime.
func (s *Service) CommentSubscription(ctx context.Context, postID int64) chan Comment {
	cc := make(chan Comment)
	c := &commentClient{comments: cc, postID: postID}
	if uid, ok := ctx.Value(KeyAuthUserID).(int64); ok {
		c.userID = &uid
	}
	s.commentClients.Store(c, nil)

	go func() {
		<-ctx.Done()
		s.commentClients.Delete(c)
		close(cc)
	}()

	return cc
}

// ToggleCommentLike 🖤
func (s *Service) ToggleCommentLike(ctx context.Context, commentID int64) (ToggleLikeOutput, error) {
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
			SELECT 1 FROM comment_likes WHERE user_id = $1 AND comment_id = $2
		)`
	if err = tx.QueryRowContext(ctx, query, uid, commentID).Scan(&out.Liked); err != nil {
		return out, fmt.Errorf("could not query select comment like existence: %v", err)
	}

	if out.Liked {
		query = "DELETE FROM comment_likes WHERE user_id = $1 AND comment_id = $2"
		if _, err = tx.ExecContext(ctx, query, uid, commentID); err != nil {
			return out, fmt.Errorf("could not delete comment like: %v", err)
		}

		query = "UPDATE comments SET likes_count = likes_count - 1 WHERE id = $1 RETURNING likes_count"
		if err = tx.QueryRowContext(ctx, query, commentID).Scan(&out.LikesCount); err != nil {
			return out, fmt.Errorf("could not update and decrement comment likes count: %v", err)
		}
	} else {
		query = "INSERT INTO comment_likes (user_id, comment_id) VALUES ($1, $2)"
		_, err = tx.ExecContext(ctx, query, uid, commentID)
		if isForeignKeyViolation(err) {
			return out, ErrCommentNotFound
		}

		if err != nil {
			return out, fmt.Errorf("could not insert comment like: %v", err)
		}

		query = "UPDATE comments SET likes_count = likes_count + 1 WHERE id = $1 RETURNING likes_count"
		if err = tx.QueryRowContext(ctx, query, commentID).Scan(&out.LikesCount); err != nil {
			return out, fmt.Errorf("could not update and increment comment likes count: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("could not commit to toggle comment like: %v", err)
	}

	out.Liked = !out.Liked

	return out, nil
}

func (s *Service) broadcastComment(c Comment) {
	s.commentClients.Range(func(key, _ interface{}) bool {
		client := key.(*commentClient)
		if client.postID == c.PostID && !(client.userID != nil && *client.userID == c.UserID) {
			client.comments <- c
		}
		return true
	})
}
