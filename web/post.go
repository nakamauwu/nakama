package web

import (
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
)

var postTmpl = parseTmpl("post.tmpl")

type postData struct {
	Session
	Post              nakama.PostRow
	Comments          []nakama.CommentsRow
	CreateCommentForm url.Values
	CreateCommentErr  error
}

func (h *Handler) renderPost(w http.ResponseWriter, data postData, statusCode int) {
	h.renderTmpl(w, postTmpl, data, statusCode)
}

// createPost handles POST /posts.
func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		h.log(err)
		h.putErr(r, "create_post_err", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	ctx := r.Context()
	_, err := h.Service.CreatePost(ctx, nakama.CreatePostInput{
		Content: r.PostFormValue("content"),
	})
	if err != nil {
		h.log(err)
		h.putErr(r, "create_post_err", err)
		h.session.Put(r, "create_post_form", r.PostForm)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// showPost handles GET /p/{postID}.
func (h *Handler) showPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	postID := mux.URLParam(ctx, "postID")

	// TODO: fetch both post and comments in parallel.
	// TODO: reverse order of comments.

	p, err := h.Service.Post(ctx, postID)
	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	cc, err := h.Service.Comments(ctx, postID)
	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderPost(w, postData{
		Session:           h.sessionFromReq(r),
		Post:              p,
		Comments:          cc,
		CreateCommentForm: h.popForm(r, "create_comment_form"),
		CreateCommentErr:  h.popErr(r, "create_comment_err"),
	}, http.StatusOK)
}
