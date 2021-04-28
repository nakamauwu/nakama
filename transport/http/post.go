package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama"
)

type createPostInput struct {
	Content   string
	SpoilerOf *string
	NSFW      bool
}

func (h *handler) createPost(w http.ResponseWriter, r *http.Request) {
	var in createPostInput
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ti, err := h.svc.CreatePost(r.Context(), in.Content, in.SpoilerOf, in.NSFW)
	if err == nakama.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == nakama.ErrInvalidContent || err == nakama.ErrInvalidSpoiler {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrUserGone {
		http.Error(w, err.Error(), http.StatusGone)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, ti, http.StatusCreated)
}

func (h *handler) posts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.Atoi(q.Get("last"))
	before := q.Get("before")
	pp, err := h.svc.Posts(ctx, way.Param(ctx, "username"), last, before)
	if err == nakama.ErrInvalidUsername || err == nakama.ErrInvalidPostID {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, pp, http.StatusOK)
}

func (h *handler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	p, err := h.svc.Post(ctx, postID)
	if err == nakama.ErrInvalidPostID {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, p, http.StatusOK)
}

func (h *handler) togglePostLike(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	out, err := h.svc.TogglePostLike(ctx, postID)
	if err == nakama.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == nakama.ErrInvalidPostID {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, out, http.StatusOK)
}

func (h *handler) togglePostSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	out, err := h.svc.TogglePostSubscription(ctx, postID)
	if err == nakama.ErrUnauthenticated {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err == nakama.ErrInvalidPostID {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err == nakama.ErrPostNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if err != nil {
		respondErr(w, err)
		return
	}

	respond(w, out, http.StatusOK)
}
