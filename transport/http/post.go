package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/matryer/way"
	"github.com/nicolasparada/nakama"
)

func (h *handler) posts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	last, _ := strconv.ParseUint(q.Get("last"), 10, 64)
	before := emptyStrPtr(q.Get("before"))
	pp, err := h.svc.Posts(ctx, way.Param(ctx, "username"), last, before)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if pp == nil {
		pp = []nakama.Post{} // non null array
	}

	for i := range pp {
		if pp[i].Reactions == nil {
			pp[i].Reactions = []nakama.Reaction{} // non null array
		}
	}

	h.respond(w, paginatedRespBody{
		Items:     pp,
		EndCursor: pp.EndCursor(),
	}, http.StatusOK)
}

func (h *handler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	p, err := h.svc.Post(ctx, postID)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if p.Reactions == nil {
		p.Reactions = []nakama.Reaction{} // non null array
	}

	h.respond(w, p, http.StatusOK)
}

func (h *handler) deletePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	err := h.svc.DeletePost(ctx, postID)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type togglePostReactionReqBody nakama.ReactionInput

func (h *handler) togglePostReaction(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var in togglePostReactionReqBody
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	out, err := h.svc.TogglePostReaction(ctx, postID, nakama.ReactionInput(in))
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, out, http.StatusOK)
}

func (h *handler) togglePostSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	out, err := h.svc.TogglePostSubscription(ctx, postID)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, out, http.StatusOK)
}
