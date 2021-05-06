//go:generate moq -out service_mock.go . Service

package transport

import (
	"context"
	"io"
	"net/url"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/nicolasparada/nakama"
)

// Service interface.
type Service interface {
	SendMagicLink(ctx context.Context, email, redirectURI string) error
	ParseRedirectURI(rawurl string) (*url.URL, error)
	VerifyMagicLink(ctx context.Context, email, code string, username *string) (nakama.AuthOutput, error)
	CredentialCreationOptions(ctx context.Context) (*protocol.CredentialCreation, *webauthn.SessionData, error)
	RegisterCredential(ctx context.Context, data webauthn.SessionData, parsedReply *protocol.ParsedCredentialCreationData) error
	CredentialRequestOptions(ctx context.Context, email string, opts ...nakama.CredentialRequestOptionsOpt) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	WebAuthnLogin(ctx context.Context, data webauthn.SessionData, reply *protocol.ParsedCredentialAssertionData) (nakama.AuthOutput, error)
	DevLogin(ctx context.Context, email string) (nakama.AuthOutput, error)
	AuthUserIDFromToken(token string) (string, error)
	AuthUser(ctx context.Context) (nakama.User, error)
	Token(ctx context.Context) (nakama.TokenOutput, error)

	CreateComment(ctx context.Context, postID string, content string) (nakama.Comment, error)
	Comments(ctx context.Context, postID string, last int, before string) ([]nakama.Comment, error)
	CommentStream(ctx context.Context, postID string) (<-chan nakama.Comment, error)
	ToggleCommentLike(ctx context.Context, commentID string) (nakama.ToggleLikeOutput, error)

	Notifications(ctx context.Context, last int, before string) ([]nakama.Notification, error)
	NotificationStream(ctx context.Context) (<-chan nakama.Notification, error)
	HasUnreadNotifications(ctx context.Context) (bool, error)
	MarkNotificationAsRead(ctx context.Context, notificationID string) error
	MarkNotificationsAsRead(ctx context.Context) error

	CreatePost(ctx context.Context, content string, spoilerOf *string, nsfw bool) (nakama.TimelineItem, error)
	Posts(ctx context.Context, username string, last int, before string) ([]nakama.Post, error)
	Post(ctx context.Context, postID string) (nakama.Post, error)
	DeletePost(ctx context.Context, postID string) error
	TogglePostLike(ctx context.Context, postID string) (nakama.ToggleLikeOutput, error)
	TogglePostSubscription(ctx context.Context, postID string) (nakama.ToggleSubscriptionOutput, error)

	Timeline(ctx context.Context, last int, before string) ([]nakama.TimelineItem, error)
	TimelineItemStream(ctx context.Context) (<-chan nakama.TimelineItem, error)
	DeleteTimelineItem(ctx context.Context, timelineItemID string) error

	Users(ctx context.Context, search string, first int, after string) ([]nakama.UserProfile, error)
	Usernames(ctx context.Context, startingWith string, first int, after string) ([]string, error)
	User(ctx context.Context, username string) (nakama.UserProfile, error)
	UpdateAvatar(ctx context.Context, r io.Reader) (string, error)
	ToggleFollow(ctx context.Context, username string) (nakama.ToggleFollowOutput, error)
	Followers(ctx context.Context, username string, first int, after string) ([]nakama.UserProfile, error)
	Followees(ctx context.Context, username string, first int, after string) ([]nakama.UserProfile, error)
}
