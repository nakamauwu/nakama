package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
)

// TimelineItem model.
type TimelineItem struct {
	ID     int64 `json:"id"`
	UserID int64 `json:"-"`
	PostID int64 `json:"-"`
	Post   *Post `json:"post,omitempty"`
}

// Timeline of the authenticated user in descending order and with backward pagination.
func (s *Service) Timeline(ctx context.Context, last int, before int64) ([]TimelineItem, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}

	last = normalizePageSize(last)
	query, args, err := buildQuery(`
		SELECT timeline.id, posts.id, content, spoiler_of, nsfw, likes_count, comments_count, created_at
		, posts.user_id = @uid AS mine
		, likes.user_id IS NOT NULL AS liked
		, subscriptions.user_id IS NOT NULL AS subscribed
		, users.username, users.avatar
		FROM timeline
		INNER JOIN posts ON timeline.post_id = posts.id
		INNER JOIN users ON posts.user_id = users.id
		LEFT JOIN post_likes AS likes
			ON likes.user_id = @uid AND likes.post_id = posts.id
		LEFT JOIN post_subscriptions AS subscriptions
			ON subscriptions.user_id = @uid AND subscriptions.post_id = posts.id
		WHERE timeline.user_id = @uid
		{{if gt .before 0}}AND timeline.id < @before{{end}}
		ORDER BY created_at DESC
		LIMIT @last`, map[string]interface{}{
		"uid":    uid,
		"last":   last,
		"before": before,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build timeline sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select timeline: %v", err)
	}

	defer rows.Close()

	tt := make([]TimelineItem, 0, last)
	for rows.Next() {
		var ti TimelineItem
		var p Post
		var u User
		var avatar sql.NullString
		if err = rows.Scan(
			&ti.ID,
			&p.ID,
			&p.Content,
			&p.SpoilerOf,
			&p.NSFW,
			&p.LikesCount,
			&p.CommentsCount,
			&p.CreatedAt,
			&p.Mine,
			&p.Liked,
			&p.Subscribed,
			&u.Username,
			&avatar,
		); err != nil {
			return nil, fmt.Errorf("could not scan timeline item: %v", err)
		}

		u.AvatarURL = s.avatarURL(avatar)
		p.User = &u
		ti.Post = &p
		tt = append(tt, ti)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate timeline rows: %v", err)
	}

	return tt, nil
}

// SubscribeToTimeline to receive timeline items in realtime.
func (s *Service) SubscribeToTimeline(ctx context.Context) (chan TimelineItem, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}

	topic := fmt.Sprintf("timeline_item:%d", uid)
	tt := make(chan TimelineItem)
	unsub, err := s.pubsub.Sub(topic, func(b []byte) {
		var ti TimelineItem
		if err := gob.NewDecoder(bytes.NewBuffer(b)).Decode(&ti); err != nil {
			log.Printf("could not decode timeline item gob: %v\n", err)
			return
		}

		tt <- ti
	})

	if err != nil {
		return nil, fmt.Errorf("could not subscribe to timeline: %v", err)
	}

	go func() {
		<-ctx.Done()
		if err := unsub(); err != nil {
			log.Printf("could not unsubscribe from timeline: %v\n", err)
		}
		close(tt)
	}()

	return tt, nil
}

// DeleteTimelineItem from the auth user timeline.
func (s *Service) DeleteTimelineItem(ctx context.Context, timelineItemID int64) error {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return ErrUnauthenticated
	}

	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM timeline
		WHERE id = $1 AND user_id = $2`, timelineItemID, uid); err != nil {
		return fmt.Errorf("could not delete timeline item: %v", err)
	}

	return nil
}

func (s *Service) fanoutPost(p Post) {
	query := `
		INSERT INTO timeline (user_id, post_id)
		SELECT follower_id, $1 FROM follows WHERE followee_id = $2
		RETURNING id, user_id`
	rows, err := s.db.Query(query, p.ID, p.UserID)
	if err != nil {
		log.Printf("could not insert timeline: %v\n", err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var ti TimelineItem
		if err = rows.Scan(&ti.ID, &ti.UserID); err != nil {
			log.Printf("could not scan timeline item: %v\n", err)
			return
		}

		ti.PostID = p.ID
		ti.Post = &p

		go s.broadcastTimelineItem(ti)
	}

	if err = rows.Err(); err != nil {
		log.Printf("could not iterate timeline rows: %v\n", err)
		return
	}
}

func (s *Service) broadcastTimelineItem(ti TimelineItem) {
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(ti); err != nil {
		log.Printf("could not encode timeline item gob: %v\n", err)
		return
	}

	topic := fmt.Sprintf("timeline_item:%d", ti.UserID)
	if err := s.pubsub.Pub(topic, b.Bytes()); err != nil {
		log.Printf("could not broadcast timeline item: %v\n", err)
	}
}
