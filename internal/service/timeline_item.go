package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
)

// ErrInvalidTimelineItemID denotes an invalid timeline item id; that is not uuid.
var ErrInvalidTimelineItemID = errors.New("invalid timeline item id")

// TimelineItem model.
type TimelineItem struct {
	ID     string `json:"id"`
	UserID string `json:"-"`
	PostID string `json:"-"`
	Post   *Post  `json:"post,omitempty"`
}

type timelineItemClient struct {
	timeline chan TimelineItem
	userID   string
}

// Timeline of the authenticated user in descending order and with backward pagination.
func (s *Service) Timeline(ctx context.Context, last int, before string) ([]TimelineItem, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
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
		{{if .before}}AND timeline.id < @before{{end}}
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

// TimelineItemStream to receive timeline items in realtime.
func (s *Service) TimelineItemStream(ctx context.Context) (<-chan TimelineItem, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return nil, ErrUnauthenticated
	}

	tt := make(chan TimelineItem)
	client := &timelineItemClient{timeline: tt, userID: uid}
	s.timelineItemClients.Store(client, nil)

	go func() {
		<-ctx.Done()
		s.timelineItemClients.Delete(client)
		close(tt)
	}()

	return tt, nil
}

// DeleteTimelineItem from the auth user timeline.
func (s *Service) DeleteTimelineItem(ctx context.Context, timelineItemID string) error {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return ErrUnauthenticated
	}

	if !reUUID.MatchString(timelineItemID) {
		return ErrInvalidTimelineItemID
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
	s.timelineItemClients.Range(func(key, _ interface{}) bool {
		client, ok := key.(*timelineItemClient)
		if !ok {
			log.Println("broadcast timeline item: no client type")
			return false
		}

		if client.userID == ti.UserID {
			client.timeline <- ti
		}

		return true
	})
}
