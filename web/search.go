package web

import (
	"net/http"

	"github.com/nakamauwu/nakama"
)

var searchPage = parsePage("search-page.tmpl")

type searchData struct {
	Session
	Err         error
	SearchQuery string
	Users       []nakama.User
}

func (h *Handler) renderSearch(w http.ResponseWriter, data searchData, statusCode int) {
	h.render(w, searchPage, data, statusCode)
}

// showSearch handles GET /search.
func (h *Handler) showSearch(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("q")
	var users []nakama.User

	if searchQuery != "" {
		var err error
		ctx := r.Context()
		users, err = h.Service.Users(ctx, nakama.UsersParams{
			UsernameQuery: searchQuery,
		})
		if err != nil {
			h.renderSearch(w, searchData{
				Session: h.sessionFromReq(r),
				Err:     maskErr(err),
			}, http.StatusOK)
			return
		}
	}

	h.renderSearch(w, searchData{
		Session:     h.sessionFromReq(r),
		SearchQuery: searchQuery,
		Users:       users,
	}, http.StatusOK)
}
