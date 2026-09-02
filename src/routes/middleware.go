package routes

import "net/http"

func roleRequired(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestRole := r.URL.Query().Get("role")
		if requestRole == "" {
			requestRole = r.Header.Get("X-Role")
		}
		if requestRole != role {
			http.Error(w, "forbidden: wrong role", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
