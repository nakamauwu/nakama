package web

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/nakamauwu/nakama"
	"github.com/nicolasparada/go-errs"
	"github.com/nicolasparada/go-mux"
	"golang.org/x/sync/errgroup"
)

var (
	postsPage   = parsePage("posts-page.tmpl")
	postPage    = parsePage("post-page.tmpl")
	postPartial = parseInclude("post.tmpl")
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
	postPartialData struct {
		Session
		Post nakama.Post
	}
)

func (h *Handler) renderPosts(w http.ResponseWriter, data postsData, statusCode int) {
	h.render(w, postsPage, data, statusCode)
}

func (h *Handler) renderPost(w http.ResponseWriter, data postData, statusCode int) {
	h.render(w, postPage, data, statusCode)
}

func (h *Handler) renderPostPartial(w http.ResponseWriter, data postPartialData, statusCode int) {
	h.render(w, postPartial, data, statusCode)
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
		posts, err = h.Service.Posts(ctx, nakama.ListPosts{})
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

	media, close, err := parseMediaList(r, "media")
	if err != nil {
		h.log(err)
		h.putErr(r, "create_post_err", err)
		h.session.Put(r, "create_post_form", r.PostForm)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	defer close()

	ctx := r.Context()
	_, err = h.Service.CreatePost(ctx, nakama.CreatePost{
		Content: r.PostFormValue("content"),
		Media:   media,
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
		post, err = h.Service.Post(gctx, nakama.RetrievePost{ID: postID})
		return err
	})

	g.Go(func() error {
		var err error
		comments, err = h.Service.Comments(gctx, nakama.ListComments{PostID: postID})
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

func parseMediaList(r *http.Request, key string) ([]nakama.Media, func() error, error) {
	// Check that request is of multipart/form-data content type.

	if r.MultipartForm == nil {
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MB
			return nil, nil, err
		}
	}

	if r.MultipartForm.File == nil {
		return nil, nil, errs.InvalidArgumentError("missing multipart form files")
	}

	m, ok := r.MultipartForm.File[key]
	if !ok {
		return nil, nil, errs.InvalidArgumentError(fmt.Sprintf("missing multipart form %s file", key))
	}

	var closeFuncs []func() error

	var out []nakama.Media
	for _, fh := range m {
		f, err := fh.Open()
		if err != nil {
			return nil, nil, err
		}

		media, err := nakama.ParseMedia(fh.Filename, f)
		if err != nil {
			_ = f.Close()
			return nil, nil, err
		}

		closeFuncs = append(closeFuncs, f.Close)
		out = append(out, media)
	}

	return out, func() error {
		for _, f := range closeFuncs {
			if err := f(); err != nil {
				return err
			}
		}
		return nil
	}, nil
}
