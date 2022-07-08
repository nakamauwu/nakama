package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:static
var staticFS embed.FS

func (h *Handler) staticHandler() http.HandlerFunc {
	// remove "static" prefix so we can server static files
	// from the root path.
	// That's `/styles.css` instead of `/static/styles.css`.
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(sub)).ServeHTTP
}
