package web

import "net/http"

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func isHXReq(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
