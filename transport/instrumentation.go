package transport

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/nakamauwu/nakama"
)

var (
	reqDur_SendMagicLink           = promauto.NewHistogram(prometheus.HistogramOpts{Name: "send_magic_link_request_duration_ms"})
	reqDur_ParseRedirectURI        = promauto.NewHistogram(prometheus.HistogramOpts{Name: "parse_redirect_uri_request_duration_ms"})
	reqDur_VerifyMagicLink         = promauto.NewHistogram(prometheus.HistogramOpts{Name: "verify_magic_link_request_duration_ms"})
	reqDur_LoginFromProvider       = promauto.NewHistogram(prometheus.HistogramOpts{Name: "login_from_provider_request_duration_ms"})
	reqDur_DevLogin                = promauto.NewHistogram(prometheus.HistogramOpts{Name: "dev_login_request_duration_ms"})
	reqDur_AuthUserIDFromToken     = promauto.NewHistogram(prometheus.HistogramOpts{Name: "auth_user_id_from_token_request_duration_ms"})
	reqDur_AuthUser                = promauto.NewHistogram(prometheus.HistogramOpts{Name: "auth_user_request_duration_ms"})
	reqDur_Token                   = promauto.NewHistogram(prometheus.HistogramOpts{Name: "token_request_duration_ms"})
	reqDur_CreateComment           = promauto.NewHistogram(prometheus.HistogramOpts{Name: "create_comment_request_duration_ms"})
	reqDur_Comments                = promauto.NewHistogram(prometheus.HistogramOpts{Name: "comments_request_duration_ms"})
	reqDur_CommentStream           = promauto.NewHistogram(prometheus.HistogramOpts{Name: "comment_stream_request_duration_ms"})
	reqDur_DeleteComment           = promauto.NewHistogram(prometheus.HistogramOpts{Name: "delete_comment_request_duration_ms"})
	reqDur_ToggleCommentReaction   = promauto.NewHistogram(prometheus.HistogramOpts{Name: "toggle_comment_reaction_request_duration_ms"})
	reqDur_Notifications           = promauto.NewHistogram(prometheus.HistogramOpts{Name: "notifications_request_duration_ms"})
	reqDur_NotificationStream      = promauto.NewHistogram(prometheus.HistogramOpts{Name: "notification_stream_request_duration_ms"})
	reqDur_HasUnreadNotifications  = promauto.NewHistogram(prometheus.HistogramOpts{Name: "has_unread_notifications_request_duration_ms"})
	reqDur_MarkNotificationAsRead  = promauto.NewHistogram(prometheus.HistogramOpts{Name: "mark_notification_as_read_request_duration_ms"})
	reqDur_MarkNotificationsAsRead = promauto.NewHistogram(prometheus.HistogramOpts{Name: "mark_notifications_as_read_request_duration_ms"})
	reqDur_Posts                   = promauto.NewHistogram(prometheus.HistogramOpts{Name: "posts_request_duration_ms"})
	reqDur_PostStream              = promauto.NewHistogram(prometheus.HistogramOpts{Name: "post_stream_request_duration_ms"})
	reqDur_Post                    = promauto.NewHistogram(prometheus.HistogramOpts{Name: "post_request_duration_ms"})
	reqDur_DeletePost              = promauto.NewHistogram(prometheus.HistogramOpts{Name: "delete_post_request_duration_ms"})
	reqDur_TogglePostReaction      = promauto.NewHistogram(prometheus.HistogramOpts{Name: "toggle_post_reaction_request_duration_ms"})
	reqDur_TogglePostSubscription  = promauto.NewHistogram(prometheus.HistogramOpts{Name: "toggle_post_subscription_request_duration_ms"})
	reqDur_CreateTimelineItem      = promauto.NewHistogram(prometheus.HistogramOpts{Name: "create_timeline_item_request_duration_ms"})
	reqDur_Timeline                = promauto.NewHistogram(prometheus.HistogramOpts{Name: "timeline_request_duration_ms"})
	reqDur_TimelineItemStream      = promauto.NewHistogram(prometheus.HistogramOpts{Name: "timeline_item_stream_request_duration_ms"})
	reqDur_DeleteTimelineItem      = promauto.NewHistogram(prometheus.HistogramOpts{Name: "delete_timeline_item_request_duration_ms"})
	reqDur_Users                   = promauto.NewHistogram(prometheus.HistogramOpts{Name: "users_request_duration_ms"})
	reqDur_Usernames               = promauto.NewHistogram(prometheus.HistogramOpts{Name: "usernames_request_duration_ms"})
	reqDur_User                    = promauto.NewHistogram(prometheus.HistogramOpts{Name: "user_request_duration_ms"})
	reqDur_UpdateUser              = promauto.NewHistogram(prometheus.HistogramOpts{Name: "update_user_request_duration_ms"})
	reqDur_UpdateAvatar            = promauto.NewHistogram(prometheus.HistogramOpts{Name: "update_avatar_request_duration_ms"})
	reqDur_UpdateCover             = promauto.NewHistogram(prometheus.HistogramOpts{Name: "update_cover_request_duration_ms"})
	reqDur_ToggleFollow            = promauto.NewHistogram(prometheus.HistogramOpts{Name: "toggle_follow_request_duration_ms"})
	reqDur_Followers               = promauto.NewHistogram(prometheus.HistogramOpts{Name: "followers_request_duration_ms"})
	reqDur_Followees               = promauto.NewHistogram(prometheus.HistogramOpts{Name: "followees_request_duration_ms"})
	reqDur_AddWebPushSubscription  = promauto.NewHistogram(prometheus.HistogramOpts{Name: "add_web_push_subscription_request_duration_ms"})
)

type ServiceWithInstrumentation struct {
	Next Service
}

func (mw *ServiceWithInstrumentation) SendMagicLink(ctx context.Context, in nakama.SendMagicLink) error {
	defer func(begin time.Time) {
		reqDur_SendMagicLink.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.SendMagicLink(ctx, in)
}

func (mw *ServiceWithInstrumentation) ParseRedirectURI(rawurl string) (*url.URL, error) {
	defer func(begin time.Time) {
		reqDur_ParseRedirectURI.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.ParseRedirectURI(rawurl)
}

func (mw *ServiceWithInstrumentation) VerifyMagicLink(ctx context.Context, email, code string, username *string) (nakama.AuthOutput, error) {
	defer func(begin time.Time) {
		reqDur_VerifyMagicLink.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.VerifyMagicLink(ctx, email, code, username)
}

func (mw *ServiceWithInstrumentation) LoginFromProvider(ctx context.Context, name string, user nakama.ProvidedUser) (nakama.User, error) {
	defer func(begin time.Time) {
		reqDur_LoginFromProvider.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.LoginFromProvider(ctx, name, user)
}

func (mw *ServiceWithInstrumentation) DevLogin(ctx context.Context, email string) (nakama.AuthOutput, error) {
	defer func(begin time.Time) {
		reqDur_DevLogin.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.DevLogin(ctx, email)
}

func (mw *ServiceWithInstrumentation) AuthUserIDFromToken(token string) (string, error) {
	defer func(begin time.Time) {
		reqDur_AuthUserIDFromToken.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.AuthUserIDFromToken(token)
}

func (mw *ServiceWithInstrumentation) AuthUser(ctx context.Context) (nakama.User, error) {
	defer func(begin time.Time) {
		reqDur_AuthUser.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.AuthUser(ctx)
}

func (mw *ServiceWithInstrumentation) Token(ctx context.Context) (nakama.TokenOutput, error) {
	defer func(begin time.Time) {
		reqDur_Token.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Token(ctx)
}

func (mw *ServiceWithInstrumentation) CreateComment(ctx context.Context, postID, content string) (nakama.Comment, error) {
	defer func(begin time.Time) {
		reqDur_CreateComment.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.CreateComment(ctx, postID, content)
}

func (mw *ServiceWithInstrumentation) Comments(ctx context.Context, postID string, last uint64, before *string) (nakama.Comments, error) {
	defer func(begin time.Time) {
		reqDur_Comments.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Comments(ctx, postID, last, before)
}

func (mw *ServiceWithInstrumentation) CommentStream(ctx context.Context, postID string) (<-chan nakama.Comment, error) {
	defer func(begin time.Time) {
		reqDur_CommentStream.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.CommentStream(ctx, postID)
}

func (mw *ServiceWithInstrumentation) DeleteComment(ctx context.Context, commentID string) error {
	defer func(begin time.Time) {
		reqDur_DeleteComment.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.DeleteComment(ctx, commentID)
}

func (mw *ServiceWithInstrumentation) ToggleCommentReaction(ctx context.Context, commentID string, in nakama.ReactionInput) ([]nakama.Reaction, error) {
	defer func(begin time.Time) {
		reqDur_ToggleCommentReaction.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.ToggleCommentReaction(ctx, commentID, in)
}

func (mw *ServiceWithInstrumentation) Notifications(ctx context.Context, last uint64, before *string) (nakama.Notifications, error) {
	defer func(begin time.Time) {
		reqDur_Notifications.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Notifications(ctx, last, before)
}

func (mw *ServiceWithInstrumentation) NotificationStream(ctx context.Context) (<-chan nakama.Notification, error) {
	defer func(begin time.Time) {
		reqDur_NotificationStream.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.NotificationStream(ctx)
}

func (mw *ServiceWithInstrumentation) HasUnreadNotifications(ctx context.Context) (bool, error) {
	defer func(begin time.Time) {
		reqDur_HasUnreadNotifications.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.HasUnreadNotifications(ctx)
}

func (mw *ServiceWithInstrumentation) MarkNotificationAsRead(ctx context.Context, notificationID string) error {
	defer func(begin time.Time) {
		reqDur_MarkNotificationAsRead.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.MarkNotificationAsRead(ctx, notificationID)
}

func (mw *ServiceWithInstrumentation) MarkNotificationsAsRead(ctx context.Context) error {
	defer func(begin time.Time) {
		reqDur_MarkNotificationsAsRead.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.MarkNotificationsAsRead(ctx)
}

func (mw *ServiceWithInstrumentation) Posts(ctx context.Context, last uint64, before *string, opts ...nakama.PostsOpt) (nakama.Posts, error) {
	defer func(begin time.Time) {
		reqDur_Posts.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Posts(ctx, last, before, opts...)
}

func (mw *ServiceWithInstrumentation) PostStream(ctx context.Context) (<-chan nakama.Post, error) {
	defer func(begin time.Time) {
		reqDur_PostStream.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.PostStream(ctx)
}

func (mw *ServiceWithInstrumentation) Post(ctx context.Context, postID string) (nakama.Post, error) {
	defer func(begin time.Time) {
		reqDur_Post.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Post(ctx, postID)
}

func (mw *ServiceWithInstrumentation) DeletePost(ctx context.Context, postID string) error {
	defer func(begin time.Time) {
		reqDur_DeletePost.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.DeletePost(ctx, postID)
}

func (mw *ServiceWithInstrumentation) TogglePostReaction(ctx context.Context, postID string, in nakama.ReactionInput) ([]nakama.Reaction, error) {
	defer func(begin time.Time) {
		reqDur_TogglePostReaction.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.TogglePostReaction(ctx, postID, in)
}

func (mw *ServiceWithInstrumentation) TogglePostSubscription(ctx context.Context, postID string) (nakama.ToggleSubscriptionOutput, error) {
	defer func(begin time.Time) {
		reqDur_TogglePostSubscription.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.TogglePostSubscription(ctx, postID)
}

func (mw *ServiceWithInstrumentation) CreateTimelineItem(ctx context.Context, content string, spoilerOf *string, nsfw bool, media []io.ReadSeeker) (nakama.TimelineItem, error) {
	defer func(begin time.Time) {
		reqDur_CreateTimelineItem.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.CreateTimelineItem(ctx, content, spoilerOf, nsfw, media)
}

func (mw *ServiceWithInstrumentation) Timeline(ctx context.Context, last uint64, before *string) (nakama.Timeline, error) {
	defer func(begin time.Time) {
		reqDur_Timeline.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Timeline(ctx, last, before)
}

func (mw *ServiceWithInstrumentation) TimelineItemStream(ctx context.Context) (<-chan nakama.TimelineItem, error) {
	defer func(begin time.Time) {
		reqDur_TimelineItemStream.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.TimelineItemStream(ctx)
}

func (mw *ServiceWithInstrumentation) DeleteTimelineItem(ctx context.Context, timelineItemID string) error {
	defer func(begin time.Time) {
		reqDur_DeleteTimelineItem.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.DeleteTimelineItem(ctx, timelineItemID)
}

func (mw *ServiceWithInstrumentation) Users(ctx context.Context, search string, first uint64, after *string) (nakama.UserProfiles, error) {
	defer func(begin time.Time) {
		reqDur_Users.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Users(ctx, search, first, after)
}

func (mw *ServiceWithInstrumentation) Usernames(ctx context.Context, startingWith string, first uint64, after *string) (nakama.Usernames, error) {
	defer func(begin time.Time) {
		reqDur_Usernames.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Usernames(ctx, startingWith, first, after)
}

func (mw *ServiceWithInstrumentation) User(ctx context.Context, username string) (nakama.UserProfile, error) {
	defer func(begin time.Time) {
		reqDur_User.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.User(ctx, username)
}

func (mw *ServiceWithInstrumentation) UpdateUser(ctx context.Context, params nakama.UpdateUserParams) error {
	defer func(begin time.Time) {
		reqDur_UpdateUser.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.UpdateUser(ctx, params)
}

func (mw *ServiceWithInstrumentation) UpdateAvatar(ctx context.Context, r io.ReadSeeker) (string, error) {
	defer func(begin time.Time) {
		reqDur_UpdateAvatar.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.UpdateAvatar(ctx, r)
}

func (mw *ServiceWithInstrumentation) UpdateCover(ctx context.Context, r io.ReadSeeker) (string, error) {
	defer func(begin time.Time) {
		reqDur_UpdateCover.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.UpdateCover(ctx, r)
}

func (mw *ServiceWithInstrumentation) ToggleFollow(ctx context.Context, username string) (nakama.ToggleFollowOutput, error) {
	defer func(begin time.Time) {
		reqDur_ToggleFollow.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.ToggleFollow(ctx, username)
}

func (mw *ServiceWithInstrumentation) Followers(ctx context.Context, username string, first uint64, after *string) (nakama.UserProfiles, error) {
	defer func(begin time.Time) {
		reqDur_Followers.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Followers(ctx, username, first, after)
}

func (mw *ServiceWithInstrumentation) Followees(ctx context.Context, username string, first uint64, after *string) (nakama.UserProfiles, error) {
	defer func(begin time.Time) {
		reqDur_Followees.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.Followees(ctx, username, first, after)
}

func (mw *ServiceWithInstrumentation) AddWebPushSubscription(ctx context.Context, sub webpush.Subscription) error {
	defer func(begin time.Time) {
		reqDur_AddWebPushSubscription.Observe(float64(time.Since(begin)) / float64(time.Millisecond))
	}(time.Now())
	return mw.Next.AddWebPushSubscription(ctx, sub)
}
