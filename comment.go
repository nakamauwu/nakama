package nakama

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/cockroachdb/cockroach-go/crdb"
)

var (
	// ErrInvalidCommentID denotes an invalid comment id; that is not uuid.
	ErrInvalidCommentID = InvalidArgumentError("invalid comment id")
	// ErrCommentNotFound denotes a not found comment.
	ErrCommentNotFound = NotFoundError("comment not found")
)

// Comment model.
type Comment struct {
	ID         string    `json:"id"`
	UserID     string    `json:"-"`
	PostID     string    `json:"-"`
	Content    string    `json:"content"`
	LikesCount int       `json:"likesCount"`
	CreatedAt  time.Time `json:"createdAt"`
	User       *User     `json:"user,omitempty"`
	Mine       bool      `json:"mine"`
	Liked      bool      `json:"liked"`
}

// CreateComment on a post.
func (s *Service) CreateComment(ctx context.Context, postID string, content string) (Comment, error) {
	var c Comment
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return c, ErrUnauthenticated
	}

	if !reUUID.MatchString(postID) {
		return c, ErrInvalidPostID
	}

	content = smartTrim(content)
	if content == "" || utf8.RuneCountInString(content) > 480 {
		return c, ErrInvalidContent
	}

	err := crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		query := `
			INSERT INTO comments (user_id, post_id, content) VALUES ($1, $2, $3)
			RETURNING id, created_at`
		err := tx.QueryRowContext(ctx, query, uid, postID, content).Scan(&c.ID, &c.CreatedAt)
		if isForeignKeyViolation(err) {
			return ErrPostNotFound
		}

		if err != nil {
			return fmt.Errorf("could not insert comment: %w", err)
		}

		c.UserID = uid
		c.PostID = postID
		c.Content = content
		c.Mine = true

		query = `
			INSERT INTO post_subscriptions (user_id, post_id) VALUES ($1, $2)
			ON CONFLICT (user_id, post_id) DO NOTHING`
		if _, err = tx.ExecContext(ctx, query, uid, postID); err != nil {
			return fmt.Errorf("could not insert post subcription after commenting: %w", err)
		}

		query = "UPDATE posts SET comments_count = comments_count + 1 WHERE id = $1"
		if _, err = tx.ExecContext(ctx, query, postID); err != nil {
			return fmt.Errorf("could not update and increment post comments count: %w", err)
		}

		return nil
	})
	if err != nil {
		return c, err
	}

	go s.commentCreated(c)

	return c, nil
}

func (s *Service) commentCreated(c Comment) {
	u, err := s.userByID(context.Background(), c.UserID)
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not fetch comment user: %w", err))
		return
	}

	c.User = &u
	c.Mine = false

	go s.notifyComment(c)
	go s.notifyCommentMention(c)
	go s.broadcastComment(c)
}

type Comments []Comment

func (cc Comments) EndCursor() *string {
	if len(cc) == 0 {
		return nil
	}

	last := cc[len(cc)-1]
	return strPtr(encodeCursor(last.ID, last.CreatedAt))
}

// Comments from a post in descending order with backward pagination.
func (s *Service) Comments(ctx context.Context, postID string, last uint64, before *string) (Comments, error) {
	if !reUUID.MatchString(postID) {
		return nil, ErrInvalidPostID
	}

	var beforeCommentID string
	var beforeCreatedAt time.Time

	if before != nil {
		var err error
		beforeCommentID, beforeCreatedAt, err = decodeCursor(*before)
		if err != nil || !reUUID.MatchString(beforeCommentID) {
			return nil, ErrInvalidCursor
		}
	}

	uid, auth := ctx.Value(KeyAuthUserID).(string)
	last = normalizePageSize(last)
	query, args, err := buildQuery(`
		SELECT comments.id
		, comments.content
		, comments.likes_count
		, comments.created_at
		, users.username
		, users.avatar
		{{if .auth}}
		, comments.user_id = @uid AS comment_mine
		, likes.user_id IS NOT NULL AS comment_liked
		{{end}}
		FROM comments
		INNER JOIN users ON comments.user_id = users.id
		{{if .auth}}
		LEFT JOIN comment_likes AS likes
			ON likes.comment_id = comments.id AND likes.user_id = @uid
		{{end}}
		WHERE comments.post_id = @postID
		{{ if and .beforeCommentID .beforeCreatedAt }}
			AND comments.created_at <= @beforeCreatedAt
			AND (
				comments.id < @beforeCommentID
					OR comments.created_at < @beforeCreatedAt
			)
		{{ end }}
		ORDER BY comments.created_at DESC, comments.id ASC
		LIMIT @last`, map[string]interface{}{
		"auth":            auth,
		"uid":             uid,
		"postID":          postID,
		"last":            last,
		"beforeCommentID": beforeCommentID,
		"beforeCreatedAt": beforeCreatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build comments sql query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select comments: %w", err)
	}

	defer rows.Close()

	var cc Comments
	for rows.Next() {
		var c Comment
		var u User
		var avatar sql.NullString
		dest := []interface{}{&c.ID, &c.Content, &c.LikesCount, &c.CreatedAt, &u.Username, &avatar}
		if auth {
			dest = append(dest, &c.Mine, &c.Liked)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan comment: %w", err)
		}

		u.AvatarURL = s.avatarURL(avatar)
		c.User = &u
		cc = append(cc, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate comment rows: %w", err)
	}

	return cc, nil
}

// CommentStream to receive comments in realtime.
func (s *Service) CommentStream(ctx context.Context, postID string) (<-chan Comment, error) {
	if !reUUID.MatchString(postID) {
		return nil, ErrInvalidPostID
	}

	cc := make(chan Comment)
	uid, auth := ctx.Value(KeyAuthUserID).(string)
	unsub, err := s.PubSub.Sub(commentTopic(postID), func(data []byte) {
		go func(r io.Reader) {
			var c Comment
			err := gob.NewDecoder(r).Decode(&c)
			if err != nil {
				_ = s.Logger.Log("error", fmt.Errorf("could not gob decode comment: %w", err))
				return
			}

			if auth && uid == c.UserID {
				return
			}

			cc <- c
		}(bytes.NewReader(data))
	})
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to comments: %w", err)
	}

	go func() {
		<-ctx.Done()
		if err := unsub(); err != nil {
			_ = s.Logger.Log("error", fmt.Errorf("could not unsubcribe from comments: %w", err))
			// don't return
		}
		close(cc)
	}()

	return cc, nil
}

// ToggleCommentLike 🖤
func (s *Service) ToggleCommentLike(ctx context.Context, commentID string) (ToggleLikeOutput, error) {
	var out ToggleLikeOutput
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return out, ErrUnauthenticated
	}

	if !reUUID.MatchString(commentID) {
		return out, ErrInvalidCommentID
	}

	err := crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		query := `
			SELECT EXISTS (
				SELECT 1 FROM comment_likes WHERE user_id = $1 AND comment_id = $2
			)`
		err := tx.QueryRowContext(ctx, query, uid, commentID).Scan(&out.Liked)
		if err != nil {
			return fmt.Errorf("could not query select comment like existence: %w", err)
		}

		if out.Liked {
			query = "DELETE FROM comment_likes WHERE user_id = $1 AND comment_id = $2"
			if _, err = tx.ExecContext(ctx, query, uid, commentID); err != nil {
				return fmt.Errorf("could not delete comment like: %w", err)
			}

			query = "UPDATE comments SET likes_count = likes_count - 1 WHERE id = $1 RETURNING likes_count"
			err = tx.QueryRowContext(ctx, query, commentID).Scan(&out.LikesCount)
			if err != nil {
				return fmt.Errorf("could not update and decrement comment likes count: %w", err)
			}
		} else {
			query = "INSERT INTO comment_likes (user_id, comment_id) VALUES ($1, $2)"
			_, err = tx.ExecContext(ctx, query, uid, commentID)
			if isForeignKeyViolation(err) {
				return ErrCommentNotFound
			}

			if err != nil {
				return fmt.Errorf("could not insert comment like: %w", err)
			}

			query = "UPDATE comments SET likes_count = likes_count + 1 WHERE id = $1 RETURNING likes_count"
			err = tx.QueryRowContext(ctx, query, commentID).Scan(&out.LikesCount)
			if err != nil {
				return fmt.Errorf("could not update and increment comment likes count: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return out, err
	}

	out.Liked = !out.Liked

	return out, nil
}

func (s *Service) broadcastComment(c Comment) {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(c)
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not gob encode comment: %w", err))
		return
	}

	err = s.PubSub.Pub(commentTopic(c.PostID), b.Bytes())
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not publish comment: %w", err))
		return
	}
}

func commentTopic(postID string) string { return "comment_" + postID }
