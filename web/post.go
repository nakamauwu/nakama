package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
)

var postTmpl = parseTmpl("post.tmpl")

type postData struct {
	Session
	Post nakama.PostRow
}

func (h *Handler) renderPost(w http.ResponseWriter, data postData, statusCode int) {
	h.renderTmpl(w, postTmpl, data, statusCode)
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.putErr(r, "create_post_err", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	ctx := r.Context()
	_, err := h.Service.CreatePost(ctx, nakama.CreatePostInput{
		Content: r.PostFormValue("content"),
	})
	if err != nil {
		h.putErr(r, "create_post_err", err)
		h.session.Put(r, "create_post_form", r.PostForm)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := mux.URLParam(ctx, "postID")
	p, err := h.Service.Post(ctx, postID)
	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderPost(w, postData{
		Session: h.sessionFromReq(r),
		Post:    p,
	}, http.StatusOK)
}
