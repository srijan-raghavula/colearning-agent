package routes

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
)

func roleRequired(role string, next http.HandlerFunc, stores ...auth.IdentityStore) http.HandlerFunc {
	return requireRole(next, func(requestRole string) bool {
		return requestRole == role
	}, "forbidden: wrong role", false, firstIdentityStore(stores))
}

func apiRoleRequired(role string, next http.HandlerFunc, stores ...auth.IdentityStore) http.HandlerFunc {
	return requireRole(next, func(requestRole string) bool {
		return requestRole == role
	}, "forbidden: wrong role", true, firstIdentityStore(stores))
}

// teacherRoleRequired keeps the product vocabulary (teacher) while accepting
// the existing faculty role used by the legacy assessment endpoints.
func teacherRoleRequired(next http.HandlerFunc, stores ...auth.IdentityStore) http.HandlerFunc {
	return requireRole(next, isTeacherRole, "forbidden: teacher role required", false, firstIdentityStore(stores))
}

func apiTeacherRoleRequired(next http.HandlerFunc, stores ...auth.IdentityStore) http.HandlerFunc {
	return requireRole(next, isTeacherRole, "forbidden: teacher role required", true, firstIdentityStore(stores))
}

func firstIdentityStore(stores []auth.IdentityStore) auth.IdentityStore {
	if len(stores) == 0 {
		return nil
	}
	return stores[0]
}

func requireRole(next http.HandlerFunc, accepts func(string) bool, message string, api bool, store auth.IdentityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := auth.IdentityFromRequestWithStore(r, store)
		if err != nil {
			if api || strings.HasPrefix(r.URL.Path, "/api/") {
				writeAPIError(w, http.StatusUnauthorized, "authentication required")
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}
		if !accepts(identity.Role) {
			if api {
				writeAPIError(w, http.StatusForbidden, message)
			} else {
				http.Error(w, message, http.StatusForbidden)
			}
			return
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	}
}

func authenticated(next http.HandlerFunc, stores ...auth.IdentityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := auth.IdentityFromRequestWithStore(r, firstIdentityStore(stores))
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	}
}

func apiAuthenticated(next http.HandlerFunc, stores ...auth.IdentityStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := auth.IdentityFromRequestWithStore(r, firstIdentityStore(stores))
		if err != nil {
			writeAPIError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	}
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
