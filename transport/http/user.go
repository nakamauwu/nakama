package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama"
)

func (h *handler) users(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := q.Get("search")
	first, _ := strconv.Atoi(q.Get("first"))
	after := q.Get("after")
	uu, err := h.svc.Users(r.Context(), search, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, uu, http.StatusOK)
}

func (h *handler) usernames(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startingWith := q.Get("starting_with")
	first, _ := strconv.Atoi(q.Get("first"))
	after := q.Get("after")
	uu, err := h.svc.Usernames(r.Context(), startingWith, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, uu, http.StatusOK)
}

func (h *handler) user(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := way.Param(ctx, "username")
	u, err := h.svc.User(ctx, username)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, u, http.StatusOK)
}

func (h *handler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	reader := http.MaxBytesReader(w, r.Body, nakama.MaxAvatarBytes)
	defer reader.Close()

	avatarURL, err := h.svc.UpdateAvatar(r.Context(), reader)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	fmt.Fprint(w, avatarURL)
}

func (h *handler) toggleFollow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := way.Param(ctx, "username")

	out, err := h.svc.ToggleFollow(ctx, username)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, out, http.StatusOK)
}

func (h *handler) followers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	username := way.Param(ctx, "username")
	first, _ := strconv.Atoi(q.Get("first"))
	after := q.Get("after")
	uu, err := h.svc.Followers(ctx, username, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, uu, http.StatusOK)
}

func (h *handler) followees(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	username := way.Param(ctx, "username")
	first, _ := strconv.Atoi(q.Get("first"))
	after := q.Get("after")
	uu, err := h.svc.Followees(ctx, username, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, uu, http.StatusOK)
}
