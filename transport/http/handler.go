package handler

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/securecookie"
	"github.com/matryer/way"
	"github.com/nicolasparada/nakama/storage"
	"github.com/nicolasparada/nakama/transport"
	"github.com/nicolasparada/nakama/web/static"
)

type handler struct {
	svc         transport.Service
	logger      log.Logger
	store       storage.Store
	cookieCodec *securecookie.SecureCookie
}

// New makes use of the service to provide an http.Handler with predefined routing.
func New(svc transport.Service, logger log.Logger, store storage.Store, cdc *securecookie.SecureCookie, enableStaticCache, embedStaticFiles, serveAvatars bool) http.Handler {
	h := &handler{
		svc:         svc,
		logger:      logger,
		store:       store,
		cookieCodec: cdc,
	}

	api := way.NewRouter()
	api.HandleFunc("POST", "/send_magic_link", h.sendMagicLink)
	api.HandleFunc("GET", "/verify_magic_link", h.verifyMagicLink)
	api.HandleFunc("GET", "/credential_creation_options", h.credentialCreationOptions)
	api.HandleFunc("POST", "/credentials", h.registerCredential)
	api.HandleFunc("GET", "/credential_request_options", h.credentialRequestOptions)
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

	api.HandleFunc("HEAD", "/proxy", withCacheControl(time.Hour*24*14)(h.proxy))

	var fsys http.FileSystem
	if embedStaticFiles {
		fsys = http.FS(static.Files)
	} else {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			_ = logger.Log("error", "could not get runtime caller")
			os.Exit(1)
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
