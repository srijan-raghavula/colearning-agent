package apphttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareAddsRequestAndSecurityHeaders(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing request ID")
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff header")
	}
	if response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing frame protection header")
	}
	if response.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing content security policy")
	}
	if response.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" {
		t.Fatal("missing cross-origin opener policy")
	}
}

func TestMiddlewareReplacesUnsafeRequestID(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "bad value")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Header().Get("X-Request-ID") == "bad value" {
		t.Fatal("unsafe request ID was trusted")
	}
}

func TestMiddlewareRecoversFromPanic(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
