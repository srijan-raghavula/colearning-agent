package routes

import (
	"encoding/json"
	"net/http"
)

func roleRequired(role string, next http.HandlerFunc) http.HandlerFunc {
	return requireRole(next, func(requestRole string) bool {
		return requestRole == role
	}, "forbidden: wrong role", false)
}

func apiRoleRequired(role string, next http.HandlerFunc) http.HandlerFunc {
	return requireRole(next, func(requestRole string) bool {
		return requestRole == role
	}, "forbidden: wrong role", true)
}

// teacherRoleRequired keeps the product vocabulary (teacher) while accepting
// the existing faculty role used by the legacy assessment endpoints.
func teacherRoleRequired(next http.HandlerFunc) http.HandlerFunc {
	return requireRole(next, isTeacherRole, "forbidden: teacher role required", false)
}

func apiTeacherRoleRequired(next http.HandlerFunc) http.HandlerFunc {
	return requireRole(next, isTeacherRole, "forbidden: teacher role required", true)
}

func requireRole(next http.HandlerFunc, accepts func(string) bool, message string, api bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !accepts(requestRole(r)) {
			if api {
				writeAPIError(w, http.StatusForbidden, message)
			} else {
				http.Error(w, message, http.StatusForbidden)
			}
			return
		}
		next(w, r)
	}
}

func requestRole(r *http.Request) string {
	role := r.URL.Query().Get("role")
	if role == "" {
		role = r.Header.Get("X-Role")
	}
	return role
}

func isTeacherRole(role string) bool {
	return role == "teacher" || role == "faculty"
}

func handleAPI(mux *http.ServeMux, method, pattern string, handler http.HandlerFunc) {
	mux.HandleFunc(method+" "+pattern, handler)
	mux.HandleFunc(pattern, apiMethodNotAllowed(method))
}

func apiMethodNotAllowed(method string) http.HandlerFunc {
	allow := method
	if method == http.MethodGet {
		allow += ", " + http.MethodHead
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func apiNotFound(w http.ResponseWriter, _ *http.Request) {
	writeAPIError(w, http.StatusNotFound, "not found")
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
