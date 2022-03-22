package nakama

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cockroachdb/cockroach-go/crdb"
)

const commentContentMaxLength = 2048

var (
	// ErrInvalidCommentID denotes an invalid comment ID; that is not uuid.
	ErrInvalidCommentID = InvalidArgumentError("invalid comment ID")
	// ErrCommentNotFound denotes a not found comment.
	ErrCommentNotFound = NotFoundError("comment not found")
)

// Comment model.
type Comment struct {
	ID        string     `json:"id"`
	UserID    string     `json:"-"`
	PostID    string     `json:"-"`
	Content   string     `json:"content"`
	Reactions []Reaction `json:"reactions"`
	CreatedAt time.Time  `json:"createdAt"`
	User      *User      `json:"user,omitempty"`
	Mine      bool       `json:"mine"`
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
	if content == "" || utf8.RuneCountInString(content) > commentContentMaxLength {
		return c, ErrInvalidContent
	}

	tags := collectTags(content)

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

		if len(tags) != 0 {
			var values []string
			args := []interface{}{postID, c.ID}
			for i := 0; i < len(tags); i++ {
				values = append(values, fmt.Sprintf("($1, $2, $%d)", i+3))
				args = append(args, tags[i])
			}

			query := `INSERT INTO post_tags (post_id, comment_id, tag) VALUES ` + strings.Join(values, ", ")
			_, err := tx.ExecContext(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("could not sql insert post (comment) tags: %w", err)
			}
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
	return ptr(encodeCursor(last.ID, last.CreatedAt))
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
		, comments.reactions
		, comments.created_at
		, users.username
		, users.avatar
		{{if .auth}}
		, comments.user_id = @uid AS comment_mine
		, reactions.user_reactions
		{{end}}
		FROM comments
		INNER JOIN users ON comments.user_id = users.id
		{{if .auth}}
		LEFT JOIN (
			SELECT user_id
			, comment_id
			, json_agg(json_build_object('reaction', reaction, 'type', type)) AS user_reactions
			FROM comment_reactions
			GROUP BY user_id, comment_id
		) AS reactions ON reactions.user_id = @uid AND reactions.comment_id = comments.id
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
		var rawReactions []byte
		var rawUserReactions []byte
		var u User
		var avatar sql.NullString
		dest := []interface{}{&c.ID, &c.Content, &rawReactions, &c.CreatedAt, &u.Username, &avatar}
		if auth {
			dest = append(dest, &c.Mine, &rawUserReactions)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan comment: %w", err)
		}

		if rawReactions != nil {
			err = json.Unmarshal(rawReactions, &c.Reactions)
			if err != nil {
				return nil, fmt.Errorf("could not json unmarshall comment reactions: %w", err)
			}
		}

		if rawUserReactions != nil {
			var userReactions []userReaction
			err = json.Unmarshal(rawUserReactions, &userReactions)
			if err != nil {
				return nil, fmt.Errorf("could not json unmarshall user comment reactions: %w", err)
			}

			for i, r := range c.Reactions {
				var reacted bool
				for _, ur := range userReactions {
					if r.Type == ur.Type && r.Reaction == ur.Reaction {
						reacted = true
						break
					}
				}
				c.Reactions[i].Reacted = &reacted
			}
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

func (s *Service) DeleteComment(ctx context.Context, commentID string) error {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return ErrUnauthenticated
	}

	if !reUUID.MatchString(commentID) {
		return ErrInvalidCommentID
	}

	err := crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		var postID string
		query := "SELECT post_id FROM comments WHERE id = $1 AND user_id = $2"
		row := tx.QueryRowContext(ctx, query, commentID, uid)
		err := row.Scan(&postID)
		if err == sql.ErrNoRows {
			return ErrCommentNotFound
		}

		if err != nil {
			return fmt.Errorf("could not sql query select comment to delete post id: %w", err)
		}

		query = "DELETE FROM comments WHERE id = $1"
		_, err = tx.ExecContext(ctx, query, commentID)
		if err != nil {
			return fmt.Errorf("could not delete comment: %w", err)
		}

		query = "UPDATE posts SET comments_count = comments_count - 1 WHERE id = $1"
		_, err = tx.ExecContext(ctx, query, postID)
		if err != nil {
			return fmt.Errorf("could not update post comments count after comment deletion: %w", err)
		}

		return nil
	})
	if err != nil && err != ErrCommentNotFound {
		return err
	}

	return nil
}

func (s *Service) ToggleCommentReaction(ctx context.Context, commentID string, in ReactionInput) ([]Reaction, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return nil, ErrUnauthenticated
	}

	if !reUUID.MatchString(commentID) {
		return nil, ErrInvalidCommentID
	}

	if in.Type != "emoji" || in.Reaction == "" {
		return nil, ErrInvalidReaction
	}

	if in.Type == "emoji" && !validEmoji(in.Reaction) {
		return nil, ErrInvalidReaction
	}

	var out []Reaction
	err := crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		out = nil

		var rawReactions []byte
		var rawUserReactions []byte
		query := `
			SELECT comments.reactions, reactions.user_reactions
			FROM comments
			LEFT JOIN (
				SELECT user_id
				, comment_id
				, json_agg(json_build_object('reaction', reaction, 'type', type)) AS user_reactions
				FROM comment_reactions
				GROUP BY user_id, comment_id
			) AS reactions ON reactions.user_id = $1 AND reactions.comment_id = comments.id
			WHERE comments.id = $2`
		row := tx.QueryRowContext(ctx, query, uid, commentID)
		err := row.Scan(&rawReactions, &rawUserReactions)
		if err == sql.ErrNoRows {
			return ErrCommentNotFound
		}

		if err != nil {
			return fmt.Errorf("could not sql scan comment and user reactions: %w", err)
		}

		var reactions []Reaction
		if rawReactions != nil {
			err = json.Unmarshal(rawReactions, &reactions)
			if err != nil {
				return fmt.Errorf("could not json unmarshall comment reactions: %w", err)
			}
		}

		var userReactions []userReaction
		if rawUserReactions != nil {
			err = json.Unmarshal(rawUserReactions, &userReactions)
			if err != nil {
				return fmt.Errorf("could not json unmarshall user comment reactions: %w", err)
			}
		}

		userReactionIdx := -1
		for i, ur := range userReactions {
			if ur.Type == in.Type && ur.Reaction == in.Reaction {
				userReactionIdx = i
				break
			}
		}

		reacted := userReactionIdx != -1
		if !reacted {
			query = "INSERT INTO comment_reactions (user_id, comment_id, type, reaction) VALUES ($1, $2, $3, $4)"
			_, err = tx.ExecContext(ctx, query, uid, commentID, in.Type, in.Reaction)
			if err != nil {
				return fmt.Errorf("could not sql insert comment reaction: %w", err)
			}
		} else {
			query = `
				DELETE FROM comment_reactions
				WHERE user_id = $1
					AND comment_id = $2
					AND type = $3
					AND reaction = $4
			`
			_, err = tx.ExecContext(ctx, query, uid, commentID, in.Type, in.Reaction)
			if err != nil {
				return fmt.Errorf("could not sql delete comment reaction: %w", err)
			}
		}

		if reacted {
			userReactions = append(userReactions[:userReactionIdx], userReactions[userReactionIdx+1:]...)
		} else {
			userReactions = append(userReactions, userReaction{
				Type:     in.Type,
				Reaction: in.Reaction,
			})
		}

		var updated bool
		zeroReactionsIdx := -1
		for i, r := range reactions {
			if !(r.Type == in.Type && r.Reaction == in.Reaction) {
				continue
			}

			if !reacted {
				reactions[i].Count++
			} else {
				reactions[i].Count--
				if reactions[i].Count == 0 {
					zeroReactionsIdx = i
				}
			}
			updated = true
			break
		}

		if !updated {
			reactions = append(reactions, Reaction{
				Type:     in.Type,
				Reaction: in.Reaction,
				Count:    1,
			})
		}

		if zeroReactionsIdx != -1 {
			reactions = append(reactions[:zeroReactionsIdx], reactions[zeroReactionsIdx+1:]...)
		}

		rawReactions, err = json.Marshal(reactions)
		if err != nil {
			return fmt.Errorf("could not json marshall comment reactions: %w", err)
		}

		query = "UPDATE comments SET reactions = $1 WHERE comments.id = $2"
		_, err = tx.ExecContext(ctx, query, rawReactions, commentID)
		if err != nil {
			return fmt.Errorf("could not sql update comment reactions: %w", err)
		}

		if len(userReactions) != 0 {
			for i, r := range reactions {
				var reacted bool
				for _, ur := range userReactions {
					if r.Type == ur.Type && r.Reaction == ur.Reaction {
						reacted = true
						break
					}
				}
				reactions[i].Reacted = &reacted
			}
		}

		out = reactions

		return nil
	})
	if err != nil {
		return nil, err
	}

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
