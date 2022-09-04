package web

import (
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
	"github.com/nakamauwu/nakama/media"
	"github.com/nicolasparada/go-mux"
	"golang.org/x/sync/errgroup"
)

var postPageTmpl = parseTmpl("post-page.tmpl")

type postData struct {
	Session
	Post              nakama.PostRow
	PostMedia         []media.Media
	Comments          []nakama.CommentsRow
	CreateCommentForm url.Values
	CreateCommentErr  error
}

func (h *Handler) renderPost(w http.ResponseWriter, data postData, statusCode int) {
	h.renderTmpl(w, postPageTmpl, data, statusCode)
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

	var post nakama.PostRow
	var postMedia []media.Media
	var comments []nakama.CommentsRow

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		post, err = h.Service.Post(gctx, postID)
		if err != nil {
			return err
		}

		postMedia = h.MediaExtractor.Extract(gctx, post.Content)

		return nil
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
		PostMedia:         postMedia,
		Comments:          comments,
		CreateCommentForm: h.popForm(r, "create_comment_form"),
		CreateCommentErr:  h.popErr(r, "create_comment_err"),
	}, http.StatusOK)
}
