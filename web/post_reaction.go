package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
)

type renderSinglePost struct {
	Session
	Post nakama.Post
}

func (h *Handler) renderSinglePost(w http.ResponseWriter, data renderSinglePost) {
	h.renderNamedTmpl(w, postPageTmpl, "post.tmpl", data, http.StatusOK)
}

// addPostReaction handles POST /p/{postID}/reactions.
func (h *Handler) addPostReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := mux.URLParam(ctx, "postID")
	reaction := r.PostFormValue("reaction")
	err := h.Service.AddPostReaction(ctx, nakama.AddPostReaction{
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
		post, err := h.Service.Post(ctx, postID)
		if err != nil {
			h.log(err)
			// TODO: flash message
			http.Redirect(w, r, r.Referer(), http.StatusFound)
			return
		}

		h.renderSinglePost(w, renderSinglePost{
			Session: h.sessionFromReq(r),
			Post:    post,
		})
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// removePostReaction handles DELETE /p/{postID}/reactions.
func (h *Handler) removePostReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := mux.URLParam(ctx, "postID")
	reaction := r.PostFormValue("reaction")
	err := h.Service.RemovePostReaction(ctx, nakama.RemovePostReaction{
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
		post, err := h.Service.Post(ctx, postID)
		if err != nil {
			h.log(err)
			// TODO: flash message
			http.Redirect(w, r, r.Referer(), http.StatusFound)
			return
		}

		h.renderSinglePost(w, renderSinglePost{
			Session: h.sessionFromReq(r),
			Post:    post,
		})
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
