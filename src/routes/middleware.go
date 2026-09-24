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

// teacherRoleRequired keeps the product vocabulary (teacher) while accepting
// the existing faculty role used by the legacy assessment endpoints.
func teacherRoleRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestRole := r.URL.Query().Get("role")
		if requestRole == "" {
			requestRole = r.Header.Get("X-Role")
		}
		if requestRole != "teacher" && requestRole != "faculty" {
			http.Error(w, "forbidden: teacher role required", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
