//go:generate moq -out service_mock.go . Service

package handler

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/duo-labs/webauthn/webauthn"
	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/internal/service"
	"github.com/nicolasparada/nakama/internal/storage"
	"github.com/nicolasparada/nakama/web/static"
)

type handler struct {
	Service
	ctx      context.Context
	store    storage.Store
	webauthn *webauthn.WebAuthn
}

// Service interface.
type Service interface {
	SendMagicLink(ctx context.Context, email, redirectURI string) error
	AuthURI(ctx context.Context, reqURI string) (*url.URL, error)
	CreateCredential(ctx context.Context, cred *webauthn.Credential) error
	WebAuthnUser(ctx context.Context, opts ...service.WebAuthnUserOpt) (service.WebAuthnUser, error)
	UpdateWebAuthnAuthenticatorSignCount(ctx context.Context, credentialID []byte, signCount uint32) error
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

	Users(ctx context.Context, search string, first int, after string) ([]service.UserProfile, error)
	Usernames(ctx context.Context, startingWith string, first int, after string) ([]string, error)
	User(ctx context.Context, username string) (service.UserProfile, error)
	UpdateAvatar(ctx context.Context, r io.Reader) (string, error)
	ToggleFollow(ctx context.Context, username string) (service.ToggleFollowOutput, error)
	Followers(ctx context.Context, username string, first int, after string) ([]service.UserProfile, error)
	Followees(ctx context.Context, username string, first int, after string) ([]service.UserProfile, error)
}

// New makes use of the service to provide an http.Handler with predefined routing.
// The provided context is used to stop the running server-sent events.
func New(ctx context.Context, svc Service, store storage.Store, webauthn *webauthn.WebAuthn, enableStaticCache, embedStaticFiles, serveAvatars bool) http.Handler {
	h := &handler{
		ctx:      ctx,
		Service:  svc,
		store:    store,
		webauthn: webauthn,
	}

	api := way.NewRouter()
	api.HandleFunc("POST", "/send_magic_link", h.sendMagicLink)
	api.HandleFunc("GET", "/auth_redirect", h.authRedirect)
	api.HandleFunc("GET", "/credential_creation_options", h.createCredentialCreationOptions)
	api.HandleFunc("POST", "/credentials", h.registerCredential)
	api.HandleFunc("GET", "/credential_request_options", h.createCredentialRequestOptions)
	api.HandleFunc("POST", "/webauthn_login", h.webAuthnLogin)
	api.HandleFunc("POST", "/dev_login", h.devLogin)
	api.HandleFunc("GET", "/auth_user", h.authUser)
	api.HandleFunc("GET", "/token", h.token)
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

	api.HandleFunc("HEAD", "/proxy", withCacheControl(time.Hour*24*14)(proxy))

	var fsys http.FileSystem
	if embedStaticFiles {
		log.Println("serving static content from embeded files")
		fsys = http.FS(static.Files)
	} else {
		log.Println("serving static content directly from disk")
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			log.Fatalln("could not get runtime caller")
		}
		fsys = http.Dir(filepath.Join(path.Dir(file), "..", "..", "web", "static"))
	}
	fsrv := http.FileServer(&spaFileSystem{root: fsys})
	if !enableStaticCache {
		fsrv = withoutCache(fsrv)
	}

	r := way.NewRouter()
	r.Handle("*", "/api/...", http.StripPrefix("/api", h.withAuth(api)))
	if serveAvatars {
		r.HandleFunc("GET", "/img/avatars/:name", h.avatar)
	}
	r.Handle("GET", "/...", fsrv)

	return r
}
