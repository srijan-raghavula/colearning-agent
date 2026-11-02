package apphttp

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// Middleware applies the application-wide HTTP guarantees. Keeping these
// outside individual controllers makes HTML and API responses consistent.
func Middleware(next http.Handler) http.Handler {
	return securityHeaders(requestID(recoverPanic(next)))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 128 {
			var raw [12]byte
			if _, err := rand.Read(raw[:]); err != nil {
				id = "request-" + time.Now().UTC().Format("20060102150405.000000000")
			} else {
				id = hex.EncodeToString(raw[:])
			}
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("http panic request_id=%s method=%s path=%s panic=%v\n%s", w.Header().Get("X-Request-ID"), r.Method, r.URL.Path, recovered, debug.Stack())
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
