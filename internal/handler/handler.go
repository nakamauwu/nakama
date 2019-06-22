package handler

import (
	"mime"
	"net/http"
	"net/url"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/internal/service"
)

type handler struct {
	*service.Service
}

// New makes use of the service to provide an http.Handler with predefined routing.
func New(s *service.Service, origin url.URL) http.Handler {
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

	mime.AddExtensionType(".js", "application/javascript; charset=utf-8")

	fs := http.FileServer(&spaFileSystem{http.Dir("web/static")})
	if origin.Hostname() == "localhost" {
		fs = withoutCache(fs)
	}

	r := way.NewRouter()
	r.Handle("*", "/api...", http.StripPrefix("/api", h.withAuth(api)))
	r.Handle("GET", "/...", fs)

	return r
}
