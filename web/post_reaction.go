package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
)

// addPostReaction handles POST /p/{postID}/reactions.
func (h *Handler) addPostReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := mux.URLParam(ctx, "postID")
	reaction := r.PostFormValue("reaction")
	err := h.Service.CreatePostReaction(ctx, nakama.PostReaction{
		PostID:   postID,
		Reaction: reaction,
	})
	if err != nil {
		// TODO: flash message
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	// render just partial <article class="post"> element
	// that will be swapped by HTMX.
	if isHXReq(r) {
		post, err := h.Service.Post(ctx, nakama.RetrievePost{ID: postID})
		if err != nil {
			h.log(err)
			// TODO: flash message
			http.Redirect(w, r, r.Referer(), http.StatusFound)
			return
		}

		h.renderPostPartial(w, postPartialData{
			Session: h.sessionFromReq(r),
			Post:    post,
		}, http.StatusOK)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// removePostReaction handles DELETE /p/{postID}/reactions.
func (h *Handler) removePostReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := mux.URLParam(ctx, "postID")
	reaction := r.PostFormValue("reaction")
	err := h.Service.DeletePostReaction(ctx, nakama.PostReaction{
		PostID:   postID,
		Reaction: reaction,
	})
	if err != nil {
		h.log(err)
		// TODO: flash message
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	// render just partial <article class="post"> element
	// that will be swapped by HTMX.
	if isHXReq(r) {
		post, err := h.Service.Post(ctx, nakama.RetrievePost{ID: postID})
		if err != nil {
			h.log(err)
			// TODO: flash message
			http.Redirect(w, r, r.Referer(), http.StatusFound)
			return
		}

		h.renderPostPartial(w, postPartialData{
			Session: h.sessionFromReq(r),
			Post:    post,
		}, http.StatusOK)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
