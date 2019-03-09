package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/disintegration/imaging"
	gonanoid "github.com/matoous/go-nanoid"
)

// MaxAvatarBytes to read.
const MaxAvatarBytes = 5 << 20 // 5MB

var (
	rxEmail    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	rxUsername = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,17}$`)
	avatarsDir = path.Join("web", "static", "img", "avatars")
)

var (
	// ErrUserNotFound denotes a not found user.
	ErrUserNotFound = errors.New("user not found")
	// ErrInvalidEmail denotes an invalid email address.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrInvalidUsername denotes an invalid username.
	ErrInvalidUsername = errors.New("invalid username")
	// ErrEmailTaken denotes an email already taken.
	ErrEmailTaken = errors.New("email taken")
	// ErrUsernameTaken denotes a username already taken.
	ErrUsernameTaken = errors.New("username taken")
	// ErrUsernameStartRequired denotes a required username start.
	ErrUsernameStartRequired = errors.New("username start required")
	// ErrForbiddenFollow denotes a forbiden follow. Like following yourself.
	ErrForbiddenFollow = errors.New("forbidden follow")
	// ErrUnsupportedAvatarFormat denotes an unsupported avatar image format.
	ErrUnsupportedAvatarFormat = errors.New("unsupported avatar format")
)

// User model.
type User struct {
	ID        int64   `json:"id,omitempty"`
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
	if !rxEmail.MatchString(email) {
		return ErrInvalidEmail
	}

	username = strings.TrimSpace(username)
	if !rxUsername.MatchString(username) {
		return ErrInvalidUsername
	}

	query := "INSERT INTO users (email, username) VALUES ($1, $2)"
	_, err := s.db.ExecContext(ctx, query, email, username)
	unique := isUniqueViolation(err)

	if unique && strings.Contains(err.Error(), "email") {
		return ErrEmailTaken
	}

	if unique && strings.Contains(err.Error(), "username") {
		return ErrUsernameTaken
	}

	if err != nil {
		return fmt.Errorf("could not insert user: %v", err)
	}

	return nil
}

// Users in ascending order with forward pagination and filtered by username.
func (s *Service) Users(ctx context.Context, search string, first int, after string) ([]UserProfile, error) {
	search = strings.TrimSpace(search)
	first = normalizePageSize(first)
	after = strings.TrimSpace(after)
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
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
		return nil, fmt.Errorf("could not build users sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select users: %v", err)
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
			return nil, fmt.Errorf("could not scan user: %v", err)
		}

		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = 0
			u.Email = ""
		}
		u.AvatarURL = s.avatarURL(avatar)
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate user rows: %v", err)
	}

	return uu, nil
}

// Usernames to autocomplete a mention box or something.
func (s *Service) Usernames(ctx context.Context, startingWith string, first int, after string) ([]string, error) {
	startingWith = strings.TrimSpace(startingWith)
	if startingWith == "" {
		return nil, ErrUsernameStartRequired
	}

	uid, auth := ctx.Value(KeyAuthUserID).(int64)
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
		return nil, fmt.Errorf("could not build usernames sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select usernames: %v", err)
	}

	defer rows.Close()

	uu := make([]string, 0, first)
	for rows.Next() {
		var u string
		if err = rows.Scan(&u); err != nil {
			return nil, fmt.Errorf("could not scan username: %v", err)
		}

		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate username rows: %v", err)
	}

	return uu, nil
}

func (s *Service) userByID(ctx context.Context, id int64) (User, error) {
	var u User
	var avatar sql.NullString
	query := "SELECT username, avatar FROM users WHERE id = $1"
	err := s.db.QueryRowContext(ctx, query, id).Scan(&u.Username, &avatar)
	if err == sql.ErrNoRows {
		return u, ErrUserNotFound
	}

	if err != nil {
		return u, fmt.Errorf("could not query select user: %v", err)
	}

	u.ID = id
	u.AvatarURL = s.avatarURL(avatar)

	return u, nil
}

// User with the given username.
func (s *Service) User(ctx context.Context, username string) (UserProfile, error) {
	var u UserProfile

	username = strings.TrimSpace(username)
	if !rxUsername.MatchString(username) {
		return u, ErrInvalidUsername
	}

	uid, auth := ctx.Value(KeyAuthUserID).(int64)
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
		return u, fmt.Errorf("could not build user sql query: %v", err)
	}

	var avatar sql.NullString
	dest := []interface{}{&u.ID, &u.Email, &avatar, &u.FollowersCount, &u.FolloweesCount}
	if auth {
		dest = append(dest, &u.Following, &u.Followeed)
	}
	err = s.db.QueryRowContext(ctx, query, args...).Scan(dest...)
	if err == sql.ErrNoRows {
		return u, ErrUserNotFound
	}

	if err != nil {
		return u, fmt.Errorf("could not query select user: %v", err)
	}

	u.Username = username
	u.Me = auth && uid == u.ID
	if !u.Me {
		u.ID = 0
		u.Email = ""
	}
	u.AvatarURL = s.avatarURL(avatar)
	return u, nil
}

// UpdateAvatar of the authenticated user returning the new avatar URL.
func (s *Service) UpdateAvatar(ctx context.Context, r io.Reader) (string, error) {
	uid, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return "", ErrUnauthenticated
	}

	r = io.LimitReader(r, MaxAvatarBytes)
	img, format, err := image.Decode(r)
	if err == image.ErrFormat {
		return "", ErrUnsupportedAvatarFormat
	}

	if err != nil {
		return "", fmt.Errorf("could not read avatar: %v", err)
	}

	if format != "png" && format != "jpeg" {
		return "", ErrUnsupportedAvatarFormat
	}

	avatar, err := gonanoid.Nanoid()
	if err != nil {
		return "", fmt.Errorf("could not generate avatar filename: %v", err)
	}

	if format == "png" {
		avatar += ".png"
	} else {
		avatar += ".jpg"
	}

	avatarPath := path.Join(avatarsDir, avatar)
	f, err := os.Create(avatarPath)
	if err != nil {
		return "", fmt.Errorf("could not create avatar file: %v", err)
	}

	defer f.Close()
	img = imaging.Fill(img, 400, 400, imaging.Center, imaging.CatmullRom)
	if format == "png" {
		err = png.Encode(f, img)
	} else {
		err = jpeg.Encode(f, img, nil)
	}
	if err != nil {
		return "", fmt.Errorf("could not write avatar to disk: %v", err)
	}

	var oldAvatar sql.NullString
	if err = s.db.QueryRowContext(ctx, `
		UPDATE users SET avatar = $1 WHERE id = $2
		RETURNING (SELECT avatar FROM users WHERE id = $2) AS old_avatar`, avatar, uid).
		Scan(&oldAvatar); err != nil {
		defer os.Remove(avatarPath)
		return "", fmt.Errorf("could not update avatar: %v", err)
	}

	if oldAvatar.Valid {
		defer os.Remove(path.Join(avatarsDir, oldAvatar.String))
	}

	avatarURL := s.origin
	avatarURL.Path = "/img/avatars/" + avatar

	return avatarURL.String(), nil
}

// ToggleFollow between two users.
func (s *Service) ToggleFollow(ctx context.Context, username string) (ToggleFollowOutput, error) {
	var out ToggleFollowOutput
	followerID, ok := ctx.Value(KeyAuthUserID).(int64)
	if !ok {
		return out, ErrUnauthenticated
	}

	username = strings.TrimSpace(username)
	if !rxUsername.MatchString(username) {
		return out, ErrInvalidUsername
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return out, fmt.Errorf("could not begin tx: %v", err)
	}

	defer tx.Rollback()

	var followeeID int64
	query := "SELECT id FROM users WHERE username = $1"
	err = tx.QueryRowContext(ctx, query, username).Scan(&followeeID)
	if err == sql.ErrNoRows {
		return out, ErrUserNotFound
	}

	if err != nil {
		return out, fmt.Errorf("could not query select user id from username: %v", err)
	}

	if followeeID == followerID {
		return out, ErrForbiddenFollow
	}

	query = `
		SELECT EXISTS (
			SELECT 1 FROM follows WHERE follower_id = $1 AND followee_id = $2
		)`
	if err = tx.QueryRowContext(ctx, query, followerID, followeeID).
		Scan(&out.Following); err != nil {
		return out, fmt.Errorf("could not query select existence of follow: %v", err)
	}

	if out.Following {
		query = "DELETE FROM follows WHERE follower_id = $1 AND followee_id = $2"
		if _, err = tx.ExecContext(ctx, query, followerID, followeeID); err != nil {
			return out, fmt.Errorf("could not delete follow: %v", err)
		}

		query = "UPDATE users SET followees_count = followees_count - 1 WHERE id = $1"
		if _, err = tx.ExecContext(ctx, query, followerID); err != nil {
			return out, fmt.Errorf("could not decrement followees count: %v", err)
		}

		query = `
			UPDATE users SET followers_count = followers_count - 1 WHERE id = $1
			RETURNING followers_count`
		if err = tx.QueryRowContext(ctx, query, followeeID).
			Scan(&out.FollowersCount); err != nil {
			return out, fmt.Errorf("could not decrement followers count: %v", err)
		}
	} else {
		query = "INSERT INTO follows (follower_id, followee_id) VALUES ($1, $2)"
		if _, err = tx.ExecContext(ctx, query, followerID, followeeID); err != nil {
			return out, fmt.Errorf("could not insert follow: %v", err)
		}

		query = "UPDATE users SET followees_count = followees_count + 1 WHERE id = $1"
		if _, err = tx.ExecContext(ctx, query, followerID); err != nil {
			return out, fmt.Errorf("could not increment followees count: %v", err)
		}

		query = `
			UPDATE users SET followers_count = followers_count + 1 WHERE id = $1
			RETURNING followers_count`
		if err = tx.QueryRowContext(ctx, query, followeeID).
			Scan(&out.FollowersCount); err != nil {
			return out, fmt.Errorf("could not increment followers count: %v", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return out, fmt.Errorf("could not commit toggle follow: %v", err)
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
	if !rxUsername.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	first = normalizePageSize(first)
	after = strings.TrimSpace(after)
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
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
		return nil, fmt.Errorf("could not build followers sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select followers: %v", err)
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
			return nil, fmt.Errorf("could not scan follower: %v", err)
		}

		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = 0
			u.Email = ""
		}
		u.AvatarURL = s.avatarURL(avatar)
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate follower rows: %v", err)
	}

	return uu, nil
}

// Followees in ascending order with forward pagination.
func (s *Service) Followees(ctx context.Context, username string, first int, after string) ([]UserProfile, error) {
	username = strings.TrimSpace(username)
	if !rxUsername.MatchString(username) {
		return nil, ErrInvalidUsername
	}

	first = normalizePageSize(first)
	after = strings.TrimSpace(after)
	uid, auth := ctx.Value(KeyAuthUserID).(int64)
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
		return nil, fmt.Errorf("could not build followees sql query: %v", err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query select followees: %v", err)
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
			return nil, fmt.Errorf("could not scan followee: %v", err)
		}

		u.Me = auth && uid == u.ID
		if !u.Me {
			u.ID = 0
			u.Email = ""
		}
		u.AvatarURL = s.avatarURL(avatar)
		uu = append(uu, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate followee rows: %v", err)
	}

	return uu, nil
}

func (s *Service) avatarURL(avatar sql.NullString) *string {
	if !avatar.Valid {
		return nil
	}

	avatarURL := s.origin
	avatarURL.Path = "/img/avatars/" + avatar.String
	str := avatarURL.String()
	return &str
}
