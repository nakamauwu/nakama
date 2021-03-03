//go:generate moq -out service_mock.go . Service

package handler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/internal/service"
)

type handler struct {
	Service
}

// Service interface.
type Service interface {
	SendMagicLink(ctx context.Context, email, redirectURI string) error
	AuthURI(ctx context.Context, verificationCode, redirectURI string) (string, error)
	DevLogin(ctx context.Context, email string) (service.DevLoginOutput, error)
	AuthUserIDFromToken(token string) (string, error)
	AuthUser(ctx context.Context) (service.User, error)
	Token(ctx context.Context) (service.TokenOutput, error)

	CreateComment(ctx context.Context, postID string, content string) (service.Comment, error)
	Comments(ctx context.Context, postID string, last int, before string) ([]service.Comment, error)
	CommentStream(ctx context.Context, postID string) (<-chan service.Comment, error)
	ToggleCommentLike(ctx context.Context, commentID string) (service.ToggleLikeOutput, error)

	Notifications(ctx context.Context, last int, before string) ([]service.Notification, error)
	NotificationStream(ctx context.Context) (<-chan service.Notification, error)
	HasUnreadNotifications(ctx context.Context) (bool, error)
	MarkNotificationAsRead(ctx context.Context, notificationID string) error
	MarkNotificationsAsRead(ctx context.Context) error

	CreatePost(ctx context.Context, content string, spoilerOf *string, nsfw bool) (service.TimelineItem, error)
	Posts(ctx context.Context, username string, last int, before string) ([]service.Post, error)
	Post(ctx context.Context, postID string) (service.Post, error)
	TogglePostLike(ctx context.Context, postID string) (service.ToggleLikeOutput, error)
	TogglePostSubscription(ctx context.Context, postID string) (service.ToggleSubscriptionOutput, error)

	Timeline(ctx context.Context, last int, before string) ([]service.TimelineItem, error)
	TimelineItemStream(ctx context.Context) (<-chan service.TimelineItem, error)
	DeleteTimelineItem(ctx context.Context, timelineItemID string) error

	CreateUser(ctx context.Context, email, username string) error
	Users(ctx context.Context, search string, first int, after string) ([]service.UserProfile, error)
	Usernames(ctx context.Context, startingWith string, first int, after string) ([]string, error)
	User(ctx context.Context, username string) (service.UserProfile, error)
	UpdateAvatar(ctx context.Context, r io.Reader) (string, error)
	ToggleFollow(ctx context.Context, username string) (service.ToggleFollowOutput, error)
	Followers(ctx context.Context, username string, first int, after string) ([]service.UserProfile, error)
	Followees(ctx context.Context, username string, first int, after string) ([]service.UserProfile, error)
}

// New makes use of the service to provide an http.Handler with predefined routing.
func New(s Service, dev bool) http.Handler {
	h := &handler{s}

	api := way.NewRouter()
	api.HandleFunc("POST", "/send_magic_link", h.sendMagicLink)
	api.HandleFunc("GET", "/auth_redirect", h.authRedirect)
	api.HandleFunc("POST", "/dev_login", h.devLogin)
	api.HandleFunc("GET", "/auth_user", h.authUser)
	api.HandleFunc("GET", "/token", h.token)
	api.HandleFunc("POST", "/users", h.createUser)
	api.HandleFunc("GET", "/users", h.users)
	api.HandleFunc("GET", "/usernames", h.usernames)
	api.HandleFunc("GET", "/users/:username", h.user)
	api.HandleFunc("PUT", "/auth_user/avatar", h.updateAvatar)
	api.HandleFunc("POST", "/users/:username/toggle_follow", h.toggleFollow)
	api.HandleFunc("GET", "/users/:username/followers", h.followers)
	api.HandleFunc("GET", "/users/:username/followees", h.followees)
	api.HandleFunc("POST", "/posts", h.createPost)
	api.HandleFunc("GET", "/users/:username/posts", h.posts)
	api.HandleFunc("GET", "/posts/:post_id", h.post)
	api.HandleFunc("POST", "/posts/:post_id/toggle_like", h.togglePostLike)
	api.HandleFunc("POST", "/posts/:post_id/toggle_subscription", h.togglePostSubscription)
	api.HandleFunc("GET", "/timeline", h.timeline)
	api.HandleFunc("DELETE", "/timeline/:timeline_item_id", h.deleteTimelineItem)
	api.HandleFunc("POST", "/posts/:post_id/comments", h.createComment)
	api.HandleFunc("GET", "/posts/:post_id/comments", h.comments)
	api.HandleFunc("POST", "/comments/:comment_id/toggle_like", h.toggleCommentLike)
	api.HandleFunc("GET", "/notifications", h.notifications)
	api.HandleFunc("GET", "/has_unread_notifications", h.hasUnreadNotifications)
	api.HandleFunc("POST", "/notifications/:notification_id/mark_as_read", h.markNotificationAsRead)
	api.HandleFunc("POST", "/mark_notifications_as_read", h.markNotificationsAsRead)

	cache := withCacheControl(time.Hour * 24 * 14)
	api.HandleFunc("HEAD", "/proxy", cache(proxy))

	fs := http.FileServer(&spaFileSystem{http.Dir("web/static")})
	if dev {
		fs = withoutCache(fs)
	}

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", h.withAuth(api)))
	r.Handle("GET", "/...", fs)

	return r
}
