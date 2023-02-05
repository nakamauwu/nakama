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
	err := h.Service.AddPostReaction(ctx, nakama.AddPostReaction{
		PostID:   postID,
		Reaction: reaction,
	})
	if err != nil {
		// TODO: flash message
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
