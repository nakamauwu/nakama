package web

import (
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-mux"
	"golang.org/x/sync/errgroup"
)

var (
	postsPageTmpl = parseTmpl("posts-page.tmpl")
	postPageTmpl  = parseTmpl("post-page.tmpl")
)

type (
	postsData struct {
		Session
		CreatePostErr  error
		CreatePostForm url.Values
		Posts          []nakama.Post
		Mode           string
	}
	postData struct {
		Session
		Post              nakama.Post
		Comments          []nakama.Comment
		CreateCommentForm url.Values
		CreateCommentErr  error
	}
)

func (h *Handler) renderPosts(w http.ResponseWriter, data postsData, statusCode int) {
	h.renderTmpl(w, postsPageTmpl, data, statusCode)
}

func (h *Handler) renderPost(w http.ResponseWriter, data postData, statusCode int) {
	h.renderTmpl(w, postPageTmpl, data, statusCode)
}

// showPosts handles GET /.
func (h *Handler) showPosts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mode := r.URL.Query().Get("mode")

	var posts []nakama.Post
	var err error

	if _, ok := nakama.UserFromContext(ctx); ok && mode != "global" {
		posts, err = h.Service.Timeline(ctx)
	} else {
		posts, err = h.Service.Posts(ctx, nakama.PostsParams{})
	}

	if err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	h.renderPosts(w, postsData{
		Session:        h.sessionFromReq(r),
		CreatePostErr:  h.popErr(r, "create_post_err"),
		CreatePostForm: h.popForm(r, "create_post_form"),
		Posts:          posts,
		Mode:           mode,
	}, http.StatusOK)
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
	_, err := h.Service.CreatePost(ctx, nakama.CreatePost{
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

	var post nakama.Post
	var comments []nakama.Comment

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		post, err = h.Service.Post(gctx, postID)
		return err
	})

	g.Go(func() error {
		var err error
		comments, err = h.Service.Comments(gctx, postID)
		return err
	})

	if err := g.Wait(); err != nil {
		h.log(err)
		h.renderErr(w, r, err)
		return
	}

	reverse(comments)

	h.renderPost(w, postData{
		Session:           h.sessionFromReq(r),
		Post:              post,
		Comments:          comments,
		CreateCommentForm: h.popForm(r, "create_comment_form"),
		CreateCommentErr:  h.popErr(r, "create_comment_err"),
	}, http.StatusOK)
}
