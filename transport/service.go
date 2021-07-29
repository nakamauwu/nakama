//go:generate moq -rm -out service_mock.go . Service

package transport

import (
	"context"
	"encoding/json"
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

	EnsureUser(ctx context.Context, email string, username *string) (nakama.User, error)

	CredentialCreationOptions(ctx context.Context) (*protocol.CredentialCreation, *webauthn.SessionData, error)
	RegisterCredential(ctx context.Context, data webauthn.SessionData, parsedReply *protocol.ParsedCredentialCreationData) error
	CredentialRequestOptions(ctx context.Context, email string, opts ...nakama.CredentialRequestOptionsOpt) (*protocol.CredentialAssertion, *webauthn.SessionData, error)
	WebAuthnLogin(ctx context.Context, data webauthn.SessionData, reply *protocol.ParsedCredentialAssertionData) (nakama.AuthOutput, error)

	DevLogin(ctx context.Context, email string) (nakama.AuthOutput, error)

	AuthUserIDFromToken(token string) (string, error)
	AuthUser(ctx context.Context) (nakama.User, error)
	Token(ctx context.Context) (nakama.TokenOutput, error)

	CreateComment(ctx context.Context, postID string, content string) (nakama.Comment, error)
	Comments(ctx context.Context, postID string, last uint64, before *string) (nakama.Comments, error)
	CommentStream(ctx context.Context, postID string) (<-chan nakama.Comment, error)
	DeleteComment(ctx context.Context, commentID string) error
	ToggleCommentReaction(ctx context.Context, commentID string, in nakama.ReactionInput) ([]nakama.Reaction, error)

	Notifications(ctx context.Context, last uint64, before *string) (nakama.Notifications, error)
	NotificationStream(ctx context.Context) (<-chan nakama.Notification, error)
	HasUnreadNotifications(ctx context.Context) (bool, error)
	MarkNotificationAsRead(ctx context.Context, notificationID string) error
	MarkNotificationsAsRead(ctx context.Context) error

	Posts(ctx context.Context, last uint64, before *string, opts ...nakama.PostsOpt) (nakama.Posts, error)
	PostStream(ctx context.Context) (<-chan nakama.Post, error)
	Post(ctx context.Context, postID string) (nakama.Post, error)
	DeletePost(ctx context.Context, postID string) error
	TogglePostReaction(ctx context.Context, postID string, in nakama.ReactionInput) ([]nakama.Reaction, error)
	TogglePostSubscription(ctx context.Context, postID string) (nakama.ToggleSubscriptionOutput, error)

	CreateTimelineItem(ctx context.Context, content string, spoilerOf *string, nsfw bool) (nakama.TimelineItem, error)
	Timeline(ctx context.Context, last uint64, before *string) (nakama.Timeline, error)
	TimelineItemStream(ctx context.Context) (<-chan nakama.TimelineItem, error)
	DeleteTimelineItem(ctx context.Context, timelineItemID string) error

	Users(ctx context.Context, search string, first uint64, after *string) (nakama.UserProfiles, error)
	Usernames(ctx context.Context, startingWith string, first uint64, after *string) (nakama.Usernames, error)
	User(ctx context.Context, username string) (nakama.UserProfile, error)
	UpdateUser(ctx context.Context, params nakama.UpdateUserParams) (nakama.UpdatedUserFields, error)
	UpdateAvatar(ctx context.Context, r io.Reader) (string, error)
	UpdateCover(ctx context.Context, r io.Reader) (string, error)
	ToggleFollow(ctx context.Context, username string) (nakama.ToggleFollowOutput, error)
	Followers(ctx context.Context, username string, first uint64, after *string) (nakama.UserProfiles, error)
	Followees(ctx context.Context, username string, first uint64, after *string) (nakama.UserProfiles, error)

	AddWebPushSubscription(ctx context.Context, sub json.RawMessage) error
}
