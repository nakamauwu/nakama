package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFS embed.FS

// staticHandler handles GET /static/*
func (h *Handler) staticHandler() http.Handler {
	staticFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}

	return http.StripPrefix("/static", http.FileServerFS(staticFS))
}
