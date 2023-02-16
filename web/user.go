package web

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-errs"
	"github.com/nicolasparada/go-mux"
	"golang.org/x/sync/errgroup"
)

var (
	userPage     = parsePage("user-page.tmpl")
	settingsPage = parsePage("settings-page.tmpl")
)

type (
	userData struct {
		Session
		User          nakama.User
		Posts         []nakama.Post
		UserFollowErr error
	}
	settingsData struct {
		Session
		User            nakama.User
		UpdateUserForm  url.Values
		UpdateUserErr   error
		UpdateAvatarErr error
	}
)

func (h *Handler) renderUser(w http.ResponseWriter, data userData, statusCode int) {
	h.render(w, userPage, data, statusCode)
}

func (h *Handler) renderSettings(w http.ResponseWriter, data settingsData, statusCode int) {
	h.render(w, settingsPage, data, statusCode)
}

// showUser handles GET /@{username}.
func (h *Handler) showUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := mux.URLParam(ctx, "username")

	var user nakama.User
	var posts []nakama.Post

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		user, err = h.Service.User(gctx, nakama.RetrieveUser{Username: username})
		return err
	})

	g.Go(func() error {
		var err error
		posts, err = h.Service.Posts(gctx, nakama.ListPosts{Username: username})
		return err
	})

	if err := g.Wait(); err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderUser(w, userData{
		Session:       h.sessionFromReq(r),
		User:          user,
		Posts:         posts,
		UserFollowErr: h.popErr(r, "user_follow_err"),
	}, http.StatusOK)
}

// showSettings handles GET /settings.
func (h *Handler) showSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := h.Service.CurrentUser(ctx)
	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderSettings(w, settingsData{
		Session:         h.sessionFromReq(r),
		User:            user,
		UpdateUserForm:  h.popForm(r, "update_username_form"),
		UpdateUserErr:   h.popErr(r, "update_username_err"),
		UpdateAvatarErr: h.popErr(r, "update_avatar_err"),
	}, http.StatusOK)
}

// updateUser handles PATCH /username.
func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	var in nakama.UpdateUser
	if r.PostForm.Has("username") {
		s := r.PostFormValue("username")
		in.Username = &s
	}

	ctx := r.Context()
	err := h.Service.UpdateUser(ctx, in)
	if err != nil {
		h.log(err)
		h.session.Put(r, "update_user_form", r.PostForm)
		h.putErr(r, "update_user_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	usr, _ := nakama.UserFromContext(ctx)

	if in.Username != nil {
		usr.Username = *in.Username
	}

	h.session.Put(r, sessionKeyUser, usr)
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// updateAvatar handles PUT /avatar.
func (h *Handler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	avatar, _, err := r.FormFile("avatar")
	if errors.Is(err, http.ErrMissingFile) {
		err = errs.InvalidArgumentError("missing avatar file")
	}

	if err != nil {
		h.log(err)
		h.putErr(r, "update_avatar_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	ctx := r.Context()
	out, err := h.Service.UpdateAvatar(ctx, avatar)
	if err != nil {
		h.log(err)
		h.putErr(r, "update_avatar_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	usr, _ := nakama.UserFromContext(ctx)

	usr.AvatarPath = &out.Path
	usr.AvatarWidth = &out.Width
	usr.AvatarHeight = &out.Height

	h.session.Put(r, sessionKeyUser, usr)
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
