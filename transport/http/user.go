package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/matryer/way"

	"github.com/nakamauwu/nakama"
)

func (h *handler) users(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	search := q.Get("search")
	first, _ := strconv.ParseUint(q.Get("first"), 10, 64)
	after := emptyStrPtr(q.Get("after"))
	uu, err := h.svc.Users(r.Context(), search, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if uu == nil {
		uu = []nakama.UserProfile{} // non null array
	}

	h.respond(w, paginatedRespBody{
		Items:     uu,
		EndCursor: uu.EndCursor(),
	}, http.StatusOK)
}

func (h *handler) usernames(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	startingWith := q.Get("starting_with")
	first, _ := strconv.ParseUint(q.Get("first"), 10, 64)
	after := emptyStrPtr(q.Get("after"))
	uu, err := h.svc.Usernames(r.Context(), startingWith, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if uu == nil {
		uu = []string{} // non null array
	}

	h.respond(w, paginatedRespBody{
		Items:     uu,
		EndCursor: uu.EndCursor(),
	}, http.StatusOK)
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

type updateUserReqBody nakama.UpdateUserParams

func (h *handler) updateUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var reqBody updateUserReqBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	ctx := r.Context()
	err = h.svc.UpdateUser(ctx, nakama.UpdateUserParams(reqBody))
	if err != nil {
		h.respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) updateAvatar(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, nakama.MaxAvatarBytes))
	if err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	avatarURL, err := h.svc.UpdateAvatar(r.Context(), bytes.NewReader(b))
	if err != nil {
		h.respondErr(w, err)
		return
	}

	fmt.Fprint(w, avatarURL)
}

func (h *handler) updateCover(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, nakama.MaxAvatarBytes))
	if err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	coverURL, err := h.svc.UpdateCover(r.Context(), bytes.NewReader(b))
	if err != nil {
		h.respondErr(w, err)
		return
	}

	fmt.Fprint(w, coverURL)
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
	first, _ := strconv.ParseUint(q.Get("first"), 10, 64)
	after := emptyStrPtr(q.Get("after"))
	uu, err := h.svc.Followers(ctx, username, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if uu == nil {
		uu = []nakama.UserProfile{} // non null array
	}

	h.respond(w, paginatedRespBody{
		Items:     uu,
		EndCursor: uu.EndCursor(),
	}, http.StatusOK)
}

func (h *handler) followees(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	username := way.Param(ctx, "username")
	first, _ := strconv.ParseUint(q.Get("first"), 10, 64)
	after := emptyStrPtr(q.Get("after"))
	uu, err := h.svc.Followees(ctx, username, first, after)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if uu == nil {
		uu = []nakama.UserProfile{} // non null array
	}

	h.respond(w, paginatedRespBody{
		Items:     uu,
		EndCursor: uu.EndCursor(),
	}, http.StatusOK)
}
