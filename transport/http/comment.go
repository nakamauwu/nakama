package http

import (
	"encoding/json"
	"mime"
	"net/http"
	"strconv"

	"github.com/matryer/way"

	"github.com/nakamauwu/nakama"
)

type createCommentInput struct {
	Content string
}

func (h *handler) createComment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var in createCommentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	c, err := h.svc.CreateComment(ctx, postID, in.Content)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if c.Reactions == nil {
		c.Reactions = []nakama.Reaction{} // non null array
	}

	h.respond(w, c, http.StatusCreated)
}

func (h *handler) comments(w http.ResponseWriter, r *http.Request) {
	if a, _, err := mime.ParseMediaType(r.Header.Get("Accept")); err == nil && a == "text/event-stream" {
		h.commentStream(w, r)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()
	postID := way.Param(ctx, "post_id")
	last, _ := strconv.ParseUint(q.Get("last"), 10, 64)
	before := emptyStrPtr(q.Get("before"))
	cc, err := h.svc.Comments(ctx, postID, last, before)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	if cc == nil {
		cc = []nakama.Comment{} // non null array
	}

	for i := range cc {
		if cc[i].Reactions == nil {
			cc[i].Reactions = []nakama.Reaction{} // non null array
		}
	}

	h.respond(w, paginatedRespBody{
		Items:     cc,
		EndCursor: cc.EndCursor(),
	}, http.StatusOK)
}

func (h *handler) commentStream(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		h.respondErr(w, errStreamingUnsupported)
		return
	}

	ctx := r.Context()
	postID := way.Param(ctx, "post_id")
	cc, err := h.svc.CommentStream(ctx, postID)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	header := w.Header()
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("Content-Type", "text/event-stream; charset=utf-8")

	select {
	case c := <-cc:
		if c.Reactions == nil {
			c.Reactions = []nakama.Reaction{}
		}
		h.writeSSE(w, c)
		f.Flush()
	case <-ctx.Done():
		return
	}
}

func (h *handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID := way.Param(ctx, "comment_id")
	err := h.svc.DeleteComment(ctx, commentID)
	if err != nil {
		h.respondErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type toggleCommentReactionReqBody nakama.ReactionInput

func (h *handler) toggleCommentReaction(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var in toggleCommentReactionReqBody
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.respondErr(w, errBadRequest)
		return
	}

	ctx := r.Context()
	commentID := way.Param(ctx, "comment_id")
	out, err := h.svc.ToggleCommentReaction(ctx, commentID, nakama.ReactionInput(in))
	if err != nil {
		h.respondErr(w, err)
		return
	}

	h.respond(w, out, http.StatusOK)
}
