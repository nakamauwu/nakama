package web

import "net/http"

func (h *Handler) followUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		h.log(err)
		h.putErr(r, "follow_user_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	ctx := r.Context()
	err := h.Service.FollowUser(ctx, r.PostFormValue("user_id"))
	if err != nil {
		h.log(err)
		h.putErr(r, "follow_user_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (h *Handler) unfollowUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if err := r.ParseForm(); err != nil {
		h.log(err)
		h.putErr(r, "follow_user_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	ctx := r.Context()
	err := h.Service.UnfollowUser(ctx, r.PostFormValue("user_id"))
	if err != nil {
		h.log(err)
		h.putErr(r, "follow_user_err", err)
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
