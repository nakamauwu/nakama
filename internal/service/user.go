package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/cockroachdb/cockroach-go/crdb"
	"github.com/disintegration/imaging"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/nicolasparada/nakama/internal/storage"
)

// MaxAvatarBytes to read.
const MaxAvatarBytes = 5 << 20 // 5MB

var (
	reEmail    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	reUsername = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,17}$`)
)

var (
	// ErrInvalidUserID denotes an invalid user id; that is not uuid.
	ErrInvalidUserID = errors.New("invalid user id")
	// ErrInvalidEmail denotes an invalid email address.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrInvalidUsername denotes an invalid username.
	ErrInvalidUsername = errors.New("invalid username")
	// ErrEmailTaken denotes an email already taken.
	ErrEmailTaken = errors.New("email taken")
	// ErrUsernameTaken denotes a username already taken.
	ErrUsernameTaken = errors.New("username taken")
	// ErrUserNotFound denotes a not found user.
	ErrUserNotFound = errors.New("user not found")
	// ErrForbiddenFollow denotes a forbiden follow. Like following yourself.
	ErrForbiddenFollow = errors.New("forbidden follow")
	// ErrUnsupportedAvatarFormat denotes an unsupported avatar image format.
	ErrUnsupportedAvatarFormat = errors.New("unsupported avatar format")
	// ErrUserGone denotes that the user has already been deleted.
	ErrUserGone = errors.New("user gone")
)

// User model.
type User struct {
	ID        string  `json:"id,omitempty"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatarURL"`
}

// UserProfile model.
type UserProfile struct {
	User
	Email          string `json:"email,omitempty"`
	FollowersCount int    `json:"followersCount"`
	FolloweesCount int    `json:"followeesCount"`
	Me             bool   `json:"me"`
	Following      bool   `json:"following"`
	Followeed      bool   `json:"followeed"`
}

// ToggleFollowOutput response.
type ToggleFollowOutput struct {
	Following      bool `json:"following"`
	FollowersCount int  `json:"followersCount"`
}

// CreateUser with the given email and name.
func (s *Service) CreateUser(ctx context.Context, email, username string) error {
	email = strings.TrimSpace(email)
	if !reEmail.MatchString(email) {
		return ErrInvalidEmail
	}

	username = strings.TrimSpace(username)
	if !reUsername.MatchString(username) {
		return ErrInvalidUsername
	}

	query := "INSERT INTO users (email, username) VALUES ($1, $2)"
	_, err := s.DB.ExecContext(ctx, query, email, username)
	unique := isUniqueViolation(err)

	if unique && strings.Contains(err.Error(), "email") {
		return ErrEmailTaken
	}

	if unique && strings.Contains(err.Error(), "username") {
		return ErrUsernameTaken
	}

	if err != nil {
		return fmt.Errorf("could not insert user: %w", err)
	}

	return nil
}

// Users in ascending order with forward pagination and filtered by username.
func (s *Service) Users(ctx context.Context, search string, first int, after string) ([]UserProfile, error) {
	search = strings.TrimSpace(search)
	first = normalizePageSize(first)
	after = strings.TrimSpace(after)
	uid, auth := ctx.Value(KeyAuthUserID).(string)
	query, args, err := buildQuery(`
		SELECT id, email, username, avatar, followers_count, followees_count
		{{if .auth}}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{end}}
		FROM users
		{{if .auth}}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{end}}
		{{if or .search .after}}WHERE{{end}}
		{{if .search}}username ILIKE '%' || @search || '%'{{end}}
		{{if and .search .after}}AND{{end}}
		{{if .after}}username > @after{{end}}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"auth":   auth,
		"uid":    uid,
		"search": search,
		"first":  first,
		"after":  after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build users sql query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select users: %w", err)
	}

	defer rows.Close()
	uu := make([]UserProfile, 0, first)
	for rows.Next() {
		var u UserProfile
		var avatar sql.NullString
		dest := []interface{}{
			&u.ID, &u.Email,
			&u.Username,
			&avatar,
			&u.FollowersCount,
			&u.FolloweesCount,
		}
		if auth {
			dest = append(dest, &u.Following, &u.Followeed)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan user: %w", err)
		}

		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = ""
			u.Email = ""
		}
		u.AvatarURL = s.avatarURL(avatar)
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate user rows: %w", err)
	}

	return uu, nil
}

// Usernames to autocomplete a mention box or something.
func (s *Service) Usernames(ctx context.Context, startingWith string, first int, after string) ([]string, error) {
	startingWith = strings.TrimSpace(startingWith)
	if startingWith == "" {
		return []string{}, nil
	}

	uid, auth := ctx.Value(KeyAuthUserID).(string)
	first = normalizePageSize(first)
	query, args, err := buildQuery(`
		SELECT username FROM users
		WHERE username ILIKE @startingWith || '%'
		{{if .auth}}AND users.id != @uid{{end}}
		{{if .after}}AND username > @after{{end}}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"startingWith": startingWith,
		"auth":         auth,
		"uid":          uid,
		"after":        after,
		"first":        first,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build usernames sql query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select usernames: %w", err)
	}

	defer rows.Close()

	uu := make([]string, 0, first)
	for rows.Next() {
		var u string
		if err = rows.Scan(&u); err != nil {
			return nil, fmt.Errorf("could not scan username: %w", err)
		}

		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate username rows: %w", err)
	}

	return uu, nil
}

func (s *Service) userByID(ctx context.Context, id string) (User, error) {
	var u User
	var avatar sql.NullString
	query := "SELECT username, avatar FROM users WHERE id = $1"
	err := s.DB.QueryRowContext(ctx, query, id).Scan(&u.Username, &avatar)
	if err == sql.ErrNoRows {
		return u, ErrUserNotFound
	}

	if err != nil {
		return u, fmt.Errorf("could not query select user: %w", err)
	}

	u.ID = id
	u.AvatarURL = s.avatarURL(avatar)

	return u, nil
}

// User with the given username.
func (s *Service) User(ctx context.Context, username string) (UserProfile, error) {
	var u UserProfile

	username = strings.TrimSpace(username)
	if !reUsername.MatchString(username) {
		return u, ErrInvalidUsername
	}

	uid, auth := ctx.Value(KeyAuthUserID).(string)
	query, args, err := buildQuery(`
		SELECT id, email, avatar, followers_count, followees_count
		{{if .auth}}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{end}}
		FROM users
		{{if .auth}}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{end}}
		WHERE username = @username`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
	})
	if err != nil {
		return u, fmt.Errorf("could not build user sql query: %w", err)
	}

	var avatar sql.NullString
	dest := []interface{}{&u.ID, &u.Email, &avatar, &u.FollowersCount, &u.FolloweesCount}
	if auth {
		dest = append(dest, &u.Following, &u.Followeed)
	}
	err = s.DB.QueryRowContext(ctx, query, args...).Scan(dest...)
	if err == sql.ErrNoRows {
		return u, ErrUserNotFound
	}

	if err != nil {
		return u, fmt.Errorf("could not query select user: %w", err)
	}

	u.Username = username
	u.Me = auth && uid == u.ID
	if !u.Me {
		u.ID = ""
		u.Email = ""
	}
	u.AvatarURL = s.avatarURL(avatar)
	return u, nil
}

// UpdateAvatar of the authenticated user returning the new avatar URL.
func (s *Service) UpdateAvatar(ctx context.Context, r io.Reader) (string, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return "", ErrUnauthenticated
	}

	r = io.LimitReader(r, MaxAvatarBytes)
	img, format, err := image.Decode(r)
	if err == image.ErrFormat {
		return "", ErrUnsupportedAvatarFormat
	}

	if err != nil {
		return "", fmt.Errorf("could not read avatar: %w", err)
	}

	if format != "png" && format != "jpeg" {
		return "", ErrUnsupportedAvatarFormat
	}

	buf := &bytes.Buffer{}
	img = imaging.Fill(img, 400, 400, imaging.Center, imaging.CatmullRom)
	if format == "png" {
		err = png.Encode(buf, img)
	} else {
		err = jpeg.Encode(buf, img, nil)
	}
	if err != nil {
		return "", fmt.Errorf("could not write avatar to disk: %w", err)
	}

	avatarFileName, err := gonanoid.Nanoid()
	if err != nil {
		return "", fmt.Errorf("could not generate avatar filename: %w", err)
	}

	if format == "png" {
		avatarFileName += ".png"
	} else {
		avatarFileName += ".jpg"
	}

	err = s.Store.Store(ctx, avatarFileName, buf.Bytes(), storage.StoreWithContentType("image/"+format))
	if err != nil {
		return "", fmt.Errorf("could not store avatar file: %w", err)
	}

	var oldAvatar sql.NullString
	if err = s.DB.QueryRowContext(ctx, `
		UPDATE users SET avatar = $1 WHERE id = $2
		RETURNING (SELECT avatar FROM users WHERE id = $2) AS old_avatar`, avatarFileName, uid).
		Scan(&oldAvatar); err != nil {

		defer func() {
			err := s.Store.Delete(context.Background(), avatarFileName)
			if err != nil {
				log.Printf("could not delete avatar file after user update fail: %v\n", err)
			}
		}()

		return "", fmt.Errorf("could not update avatar: %w", err)
	}

	if oldAvatar.Valid {
		defer func() {
			err := s.Store.Delete(context.Background(), oldAvatar.String)
			if err != nil {
				log.Printf("could not delete old avatar: %v\n", err)
			}
		}()
	}

	avatarURL := cloneURL(s.Origin)
	avatarURL.Path = "/img/avatars/" + avatarFileName

	return avatarURL.String(), nil
}

// ToggleFollow between two users.
func (s *Service) ToggleFollow(ctx context.Context, username string) (ToggleFollowOutput, error) {
	var out ToggleFollowOutput
	followerID, ok := ctx.Value(KeyAuthUserID).(string)
	if !ok {
		return out, ErrUnauthenticated
	}

	username = strings.TrimSpace(username)
	if !reUsername.MatchString(username) {
		return out, ErrInvalidUsername
	}

	var followeeID string
	err := crdb.ExecuteTx(ctx, s.DB, nil, func(tx *sql.Tx) error {
		query := "SELECT id FROM users WHERE username = $1"
		err := tx.QueryRowContext(ctx, query, username).Scan(&followeeID)
		if err == sql.ErrNoRows {
			return ErrUserNotFound
		}

		if err != nil {
			return fmt.Errorf("could not query select user id from username: %w", err)
		}

		if followeeID == followerID {
			return ErrForbiddenFollow
		}

		query = `
			SELECT EXISTS (
				SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2
			)`
		row := tx.QueryRowContext(ctx, query, followerID, followeeID)
		err = row.Scan(&out.Following)
		if err != nil {
			return fmt.Errorf("could not query select existence of follow: %w", err)
		}

		if out.Following {
			query = "DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2"
			_, err = tx.ExecContext(ctx, query, followerID, followeeID)
			if err != nil {
				return fmt.Errorf("could not delete follow: %w", err)
			}

			query = "UPDATE users SET followees_count = followees_count - 1 WHERE id = $1"
			if _, err = tx.ExecContext(ctx, query, followerID); err != nil {
				return fmt.Errorf("could not decrement followees count: %w", err)
			}

			query = `
				UPDATE users SET followers_count = followers_count - 1 WHERE id = $1
				RETURNING followers_count`
			if err = tx.QueryRowContext(ctx, query, followeeID).
				Scan(&out.FollowersCount); err != nil {
				return fmt.Errorf("could not decrement followers count: %w", err)
			}
		} else {
			query = "INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2)"
			_, err = tx.ExecContext(ctx, query, followerID, followeeID)
			if err != nil {
				return fmt.Errorf("could not insert follow: %w", err)
			}

			query = "UPDATE users SET followees_count = followees_count + 1 WHERE id = $1"
			if _, err = tx.ExecContext(ctx, query, followerID); err != nil {
				return fmt.Errorf("could not increment followees count: %w", err)
			}

			query = `
				UPDATE users SET followers_count = followers_count + 1 WHERE id = $1
				RETURNING followers_count`
			row = tx.QueryRowContext(ctx, query, followeeID)
			err = row.Scan(&out.FollowersCount)
			if err != nil {
				return fmt.Errorf("could not increment followers count: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return out, err
	}

	out.Following = !out.Following

	if out.Following {
		go s.notifyFollow(followerID, followeeID)
	}

	return out, nil
}

// Followers in ascending order with forward pagination.
func (s *Service) Followers(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {
	username = strings.TrimSpace(username)
	if !reUsername.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	first = normalizePageSize(first)
	after = strings.TrimSpace(after)
	uid, auth := ctx.Value(KeyAuthUserID).(string)
	query, args, err := buildQuery(`
		SELECT id, email, username, avatar, followers_count, followees_count
		{{if .auth}}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{end}}
		FROM follows
		INNER JOIN users ON follows.follower_id = users.id
		{{if .auth}}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{end}}
		WHERE follows.followee_id = (SELECT id FROM users WHERE username = @username)
		{{if .after}}AND username > @after{{end}}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
		"first":    first,
		"after":    after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build followers sql query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select followers: %w", err)
	}

	defer rows.Close()
	uu := make([]UserProfile, 0, first)
	for rows.Next() {
		var u UserProfile
		var avatar sql.NullString
		dest := []interface{}{
			&u.ID,
			&u.Email,
			&u.Username,
			&avatar,
			&u.FollowersCount,
			&u.FolloweesCount,
		}
		if auth {
			dest = append(dest, &u.Following, &u.Followeed)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan follower: %w", err)
		}

		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = ""
			u.Email = ""
		}
		u.AvatarURL = s.avatarURL(avatar)
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate follower rows: %w", err)
	}

	return uu, nil
}

// Followees in ascending order with forward pagination.
func (s *Service) Followees(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {
	username = strings.TrimSpace(username)
	if !reUsername.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	first = normalizePageSize(first)
	after = strings.TrimSpace(after)
	uid, auth := ctx.Value(KeyAuthUserID).(string)
	query, args, err := buildQuery(`
		SELECT id, email, username, avatar, followers_count, followees_count
		{{if .auth}}
		, followers.follower_id IS NOT NULL AS following
		, followees.followee_id IS NOT NULL AS followeed
		{{end}}
		FROM follows
		INNER JOIN users ON follows.followee_id = users.id
		{{if .auth}}
		LEFT JOIN follows AS followers
			ON followers.follower_id = @uid AND followers.followee_id = users.id
		LEFT JOIN follows AS followees
			ON followees.follower_id = users.id AND followees.followee_id = @uid
		{{end}}
		WHERE follows.follower_id = (SELECT id FROM users WHERE username = @username)
		{{if .after}}AND username > @after{{end}}
		ORDER BY username ASC
		LIMIT @first`, map[string]interface{}{
		"auth":     auth,
		"uid":      uid,
		"username": username,
		"first":    first,
		"after":    after,
	})
	if err != nil {
		return nil, fmt.Errorf("could not build followees sql query: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select followees: %w", err)
	}

	defer rows.Close()
	uu := make([]UserProfile, 0, first)
	for rows.Next() {
		var u UserProfile
		var avatar sql.NullString
		dest := []interface{}{
			&u.ID,
			&u.Email,
			&u.Username,
			&avatar,
			&u.FollowersCount,
			&u.FolloweesCount,
		}
		if auth {
			dest = append(dest, &u.Following, &u.Followeed)
		}
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("could not scan followee: %w", err)
		}

		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = ""
			u.Email = ""
		}
		u.AvatarURL = s.avatarURL(avatar)
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate followee rows: %w", err)
	}

	return uu, nil
}

func (s *Service) avatarURL(avatar sql.NullString) *string {
	if !avatar.Valid {
		return nil
	}

	avatarURL := cloneURL(s.Origin)
	avatarURL.Path = "/img/avatars/" + avatar.String
	str := avatarURL.String()
	return &str
}
