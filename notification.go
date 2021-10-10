package nakama

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"time"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/lib/pq"
)

// ErrInvalidNotificationID denotes an invalid notification id; that is not uuid.
var ErrInvalidNotificationID = InvalidArgumentError("invalid notification ID")

// Notification model.
type Notification struct {
	ID       string    `json:"id"`
	UserID   string    `json:"-"`
	Actors   []string  `json:"actors"`
	Type     string    `json:"type"`
	PostID   *string   `json:"postID,omitempty"`
	Read     bool      `json:"read"`
	IssuedAt time.Time `json:"issuedAt"`
}

type Notifications []Notification

func (pp Notifications) EndCursor() *string {
	if len(pp) == 0 {
		return nil
	}

	last := pp[len(pp)-1]
	return strPtr(encodeCursor(last.ID, last.IssuedAt))
}

// Notifications from the authenticated user in descending order with backward pagination.
func (s *Service) Notifications(ctx context.Context, last uint64, before *string) (Notifications, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return nil, ErrUnauthenticated
	}

	var beforeNotificationID string
	var beforeIssuedAt time.Time

	if before != nil {
		var err error
		beforeNotificationID, beforeIssuedAt, err = decodeCursor(*before)
		if err != nil || !reUUID.MatchString(beforeNotificationID) {
			return nil, ErrInvalidCursor
		}
	}

	last = normalizePageSize(last)
	query, args, err := buildQuery(`
		SELECT id
		, actors
		, type
		, post_id
		, read_at
		, issued_at
		FROM notifications
		WHERE user_id = @uid
		{{ if and .beforeNotificationID .beforeIssuedAt }}
			AND issued_at <= @beforeIssuedAt
			AND (
				id < @beforeNotificationID
					OR issued_at < @beforeIssuedAt
			)
		{{ end }}
		ORDER BY issued_at DESC
		LIMIT @last`, map[string]interface{}{
		"uid":                  uid,
		"last":                 last,
		"beforeNotificationID": beforeNotificationID,
		"beforeIssuedAt":       beforeIssuedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build notifications sql query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select notifications: %w", err)
	}

	defer rows.Close()

	var nn Notifications
	for rows.Next() {
		var n Notification
		var readAt *time.Time
		if err = rows.Scan(&n.ID, pq.Array(&n.Actors), &n.Type, &n.PostID, &readAt, &n.IssuedAt); err != nil {
			return nil, fmt.Errorf("could not scan notification: %w", err)
		}

		n.Read = readAt != nil && !readAt.IsZero()
		nn = append(nn, n)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate over notification rows: %w", err)
	}

	return nn, nil
}

// NotificationStream to receive notifications in realtime.
func (s *Service) NotificationStream(ctx context.Context) (<-chan Notification, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return nil, ErrUnauthenticated
	}

	nn := make(chan Notification)
	unsub, err := s.PubSub.Sub(notificationTopic(uid), func(data []byte) {
		go func(r io.Reader) {
			var n Notification
			err := gob.NewDecoder(r).Decode(&n)
			if err != nil {
				_ = s.Logger.Log("error", fmt.Errorf("could not gob decode notification: %w", err))
				return
			}

			nn <- n
		}(bytes.NewReader(data))
	})
	if err != nil {
		return nil, fmt.Errorf("could not subcribe to notifications: %w", err)
	}

	go func() {
		<-ctx.Done()
		if err := unsub(); err != nil {
			_ = s.Logger.Log("error", fmt.Errorf("could not unsubcribe from notifications: %w", err))
			// don't return
		}
		close(nn)
	}()

	return nn, nil
}

// HasUnreadNotifications checks if the authenticated user has any unread notification.
func (s *Service) HasUnreadNotifications(ctx context.Context) (bool, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return false, ErrUnauthenticated
	}

	var unread bool
	if err := s.DB.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM notifications WHERE user_id = $1 AND (read_at IS NULL OR read_at = '0001-01-01 00:00:00')
	)`, uid).Scan(&unread); err != nil {
		return false, fmt.Errorf("could not query select unread notifications existence: %w", err)
	}

	return unread, nil
}

// MarkNotificationAsRead sets a notification from the authenticated user as read.
func (s *Service) MarkNotificationAsRead(ctx context.Context, notificationID string) error {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return ErrUnauthenticated
	}

	if !reUUID.MatchString(notificationID) {
		return ErrInvalidNotificationID
	}

	if _, err := s.DB.Exec(`
		UPDATE notifications SET read_at = now()
		WHERE id = $1 AND user_id = $2 AND (read_at IS NULL OR read_at = '0001-01-01 00:00:00')`, notificationID, uid); err != nil {
		return fmt.Errorf("could not update and mark notification as read: %w", err)
	}

	return nil
}

// MarkNotificationsAsRead sets all notification from the authenticated user as read.
func (s *Service) MarkNotificationsAsRead(ctx context.Context) error {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return ErrUnauthenticated
	}

	if _, err := s.DB.Exec(`
		UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND (read_at IS NULL OR read_at = '0001-01-01 00:00:00')
	`, uid); err != nil {
		return fmt.Errorf("could not update and mark notifications as read: %w", err)
	}

	return nil
}

func (s *Service) notifyFollow(followerID, followeeID string) {
	ctx := context.Background()
	var n Notification
	err := crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		var actor string
		query := "SELECT username FROM users WHERE id = $1"
		err := tx.QueryRowContext(ctx, query, followerID).Scan(&actor)
		if err != nil {
			return fmt.Errorf("could not query select follow notification actor: %w", err)
		}

		var notified bool
		query = `SELECT EXISTS (
			SELECT 1 FROM notifications
			WHERE user_id = $1
				AND $2:::VARCHAR = ANY(actors)
				AND type = 'follow'
		)`
		err = tx.QueryRowContext(ctx, query, followeeID, actor).Scan(&notified)
		if err != nil {
			return fmt.Errorf("could not query select follow notification existence: %w", err)
		}

		if notified {
			return nil
		}

		var nid string
		query = "SELECT id FROM notifications WHERE user_id = $1 AND type = 'follow' AND (read_at IS NULL OR read_at = '0001-01-01 00:00:00')"
		err = tx.QueryRowContext(ctx, query, followeeID).Scan(&nid)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("could not query select unread follow notification: %w", err)
		}

		if err == sql.ErrNoRows {
			actors := []string{actor}
			query = `
				INSERT INTO notifications (user_id, actors, type) VALUES ($1, $2, 'follow')
				RETURNING id, issued_at`
			row := tx.QueryRowContext(ctx, query, followeeID, pq.Array(actors))
			err = row.Scan(&n.ID, &n.IssuedAt)
			if err != nil {
				return fmt.Errorf("could not insert follow notification: %w", err)
			}

			n.Actors = actors
		} else {
			query = `
				UPDATE notifications SET
					actors = array_prepend($1, notifications.actors),
					issued_at = now()
				WHERE id = $2
				RETURNING actors, issued_at`
			row := tx.QueryRowContext(ctx, query, actor, nid)
			err = row.Scan(pq.Array(&n.Actors), &n.IssuedAt)
			if err != nil {
				return fmt.Errorf("could not update follow notification: %w", err)
			}

			n.ID = nid
		}

		n.UserID = followeeID
		n.Type = "follow"

		return nil
	})
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not notify follow: %w", err))
		return
	}

	go s.broadcastNotification(n)
}

func (s *Service) notifyComment(c Comment) {
	actor := c.User.Username
	rows, err := s.DB.Query(`
		INSERT INTO notifications (user_id, actors, type, post_id, read_at)
		SELECT user_id, $1, 'comment', $2, '0001-01-01 00:00:00' FROM post_subscriptions
		WHERE post_subscriptions.user_id != $3
			AND post_subscriptions.post_id = $2
		ON CONFLICT (user_id, type, post_id, read_at) DO UPDATE SET
			actors = array_prepend($4, array_remove(notifications.actors, $4)),
			issued_at = now()
		RETURNING id, user_id, actors, issued_at`,
		pq.Array([]string{actor}),
		c.PostID,
		c.UserID,
		actor,
	)
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not insert comment notifications: %w", err))
		return
	}

	defer rows.Close()

	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.UserID, pq.Array(&n.Actors), &n.IssuedAt); err != nil {
			_ = s.Logger.Log("error", fmt.Errorf("could not scan comment notification: %w", err))
			return
		}

		n.Type = "comment"
		n.PostID = &c.PostID

		go s.broadcastNotification(n)
	}

	if err = rows.Err(); err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not iterate over comment notification rows: %w", err))
		return
	}
}

func (s *Service) notifyPostMention(p Post) {
	mentions := collectMentions(p.Content)
	if len(mentions) == 0 {
		return
	}

	actors := []string{p.User.Username}
	rows, err := s.DB.Query(`
		INSERT INTO notifications (user_id, actors, type, post_id)
		SELECT users.id, $1, 'post_mention', $2 FROM users
		WHERE users.id != $3
			AND username = ANY($4)
		RETURNING id, user_id, issued_at`,
		pq.Array(actors),
		p.ID,
		p.UserID,
		pq.Array(mentions),
	)
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not insert post mention notifications: %w", err))
		return
	}

	defer rows.Close()

	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.UserID, &n.IssuedAt); err != nil {
			_ = s.Logger.Log("error", fmt.Errorf("could not scan post mention notification: %w", err))
			return
		}

		n.Actors = actors
		n.Type = "post_mention"
		n.PostID = &p.ID

		go s.broadcastNotification(n)
	}

	if err = rows.Err(); err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not iterate post mention notification rows: %w", err))
		return
	}
}

func (s *Service) notifyCommentMention(c Comment) {
	mentions := collectMentions(c.Content)
	if len(mentions) == 0 {
		return
	}

	actor := c.User.Username
	rows, err := s.DB.Query(`
		INSERT INTO notifications (user_id, actors, type, post_id, read_at)
		SELECT users.id, $1, 'comment_mention', $2, '0001-01-01 00:00:00' FROM users
		WHERE users.id != $3
			AND username = ANY($4)
		ON CONFLICT (user_id, type, post_id, read_at) DO UPDATE SET
			actors = array_prepend($5, array_remove(notifications.actors, $5)),
			issued_at = now()
		RETURNING id, user_id, actors, issued_at`,
		pq.Array([]string{actor}),
		c.PostID,
		c.UserID,
		pq.Array(mentions),
		actor,
	)
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not insert comment mention notifications: %w", err))
		return
	}

	defer rows.Close()

	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.UserID, pq.Array(&n.Actors), &n.IssuedAt); err != nil {
			_ = s.Logger.Log("error", fmt.Errorf("could not scan comment mention notification: %w", err))
			return
		}

		n.Type = "comment_mention"
		n.PostID = &c.PostID

		go s.broadcastNotification(n)
	}

	if err = rows.Err(); err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not iterate comment mention notification rows: %w", err))
		return
	}
}

func (s *Service) broadcastNotification(n Notification) {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(n)
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not gob encode notification: %w", err))
		return
	}

	err = s.PubSub.Pub(notificationTopic(n.UserID), b.Bytes())
	if err != nil {
		_ = s.Logger.Log("error", fmt.Errorf("could not publish notification: %w", err))
		return
	}

	go s.sendWebPushNotifications(n)
}

func notificationTopic(userID string) string { return "notification_" + userID }
