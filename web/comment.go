package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
)

var commentPartial = parseInclude("comment.tmpl")

type commentPartialData struct {
	Session
	Comment nakama.Comment
}

func (h *Handler) renderCommentPartial(w http.ResponseWriter, data commentPartialData, statusCode int) {
	h.render(w, commentPartial, data, statusCode)
}

// createComment handles POST /comments.
func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		h.log(err)
		h.putErr(r, "create_comment_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	ctx := r.Context()
	_, err := h.Service.CreateComment(ctx, nakama.CreateComment{
		PostID:  r.PostFormValue("post_id"),
		Content: r.PostFormValue("content"),
	})
	if err != nil {
		h.log(err)
		h.putErr(r, "create_comment_err", err)
		h.session.Put(r, "create_comment_form", r.PostForm)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
