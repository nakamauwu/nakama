package service

import (
	"context"
	"database/sql"
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

type timelineItemClient struct {
	timeline chan TimelineItem
	userID   int64
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

// TimelineItemSubscription to receive timeline items in realtime.
func (s *Service) TimelineItemSubscription(ctx context.Context) (chan TimelineItem, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return nil, ErrUnauthenticated
	}

	tt := make(chan TimelineItem)
	c := &timelineItemClient{timeline: tt, userID: uid}
	s.timelineItemClients.Store(c, nil)

	go func() {
		<-ctx.Done()
		s.timelineItemClients.Delete(c)
		close(tt)
	}()

	return tt, nil
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
	s.timelineItemClients.Range(func(key, _ interface{}) bool {
		client := key.(*timelineItemClient)
		if client.userID == ti.UserID {
			client.timeline <- ti
		}
		return true
	})
}
