package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
)

type renderSingleComment struct {
	Session
	Comment nakama.Comment
}

func (h *Handler) renderSingleComment(w http.ResponseWriter, data renderSingleComment) {
	h.renderNamedTmpl(w, postPageTmpl, "comment.tmpl", data, http.StatusOK)
}

// addCommentReaction handles POST /comments/{commentID}/reactions.
func (h *Handler) addCommentReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID := mux.URLParam(ctx, "commentID")
	reaction := r.PostFormValue("reaction")
	err := h.Service.CreateCommentReaction(ctx, nakama.CommentReaction{
		CommentID: commentID,
		Reaction:  reaction,
	})
	if err != nil {
		// TODO: flash message
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	// render just partial <article class="comment"> element
	// that will be swapped by HTMX.
	if isHXReq(r) {
		comment, err := h.Service.Comment(ctx, commentID)
		if err != nil {
			h.log(err)
			// TODO: flash message
			http.Redirect(w, r, r.Referer(), http.StatusFound)
			return
		}

		h.renderSingleComment(w, renderSingleComment{
			Session: h.sessionFromReq(r),
			Comment: comment,
		})
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

// removeCommentReaction handles DELETE /comments/{commentID}/reactions.
func (h *Handler) removeCommentReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentID := mux.URLParam(ctx, "commentID")
	reaction := r.PostFormValue("reaction")
	err := h.Service.DeleteCommentReaction(ctx, nakama.CommentReaction{
		CommentID: commentID,
		Reaction:  reaction,
	})
	if err != nil {
		h.log(err)
		// TODO: flash message
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	// render just partial <article class="comment"> element
	// that will be swapped by HTMX.
	if isHXReq(r) {
		comment, err := h.Service.Comment(ctx, commentID)
		if err != nil {
			h.log(err)
			// TODO: flash message
			http.Redirect(w, r, r.Referer(), http.StatusFound)
			return
		}

		h.renderSingleComment(w, renderSingleComment{
			Session: h.sessionFromReq(r),
			Comment: comment,
		})
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
