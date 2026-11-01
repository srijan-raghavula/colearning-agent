package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srijan-raghavula/colearning-agent/src/bootstrap"
	"github.com/srijan-raghavula/colearning-agent/src/routes"
)

func TestEndToEndStudentFlow(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err != nil {
		t.Fatalf("locate project root: %v", err)
	}
	t.Chdir(projectRoot)
	dataDir := t.TempDir()
	t.Setenv("COLEARN_DB_PATH", filepath.Join(dataDir, "colearning.db"))
	t.Setenv("LEARNING_STORE_PATH", filepath.Join(dataDir, "learning-state.json"))
	t.Setenv("COLEARN_JWT_SECRET", strings.Repeat("t", 32))

	app, err := bootstrap.NewApp()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if app.Close != nil {
		t.Cleanup(func() { _ = app.Close() })
	}
	mux := http.NewServeMux()
	routes.Register(mux, app.Routes)

	form := url.Values{"email": {"student@demo.edu"}, "password": {"student123"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("Accept", "application/json")
	loginRecorder := httptest.NewRecorder()
	mux.ServeHTTP(loginRecorder, loginReq)
	loginResp := loginRecorder.Result()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want %d", loginResp.StatusCode, http.StatusOK)
	}
	var authCookie *http.Cookie
	for _, cookie := range loginResp.Cookies() {
		if cookie.Name == "auth_token" {
			authCookie = cookie
		}
	}
	if authCookie == nil {
		t.Fatal("login response did not set auth cookie")
	}

	doRequest := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.AddCookie(authCookie)
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		return resp
	}

	profile := doRequest(http.MethodGet, "/profile", "")
	profileBody, _ := io.ReadAll(profile.Result().Body)
	if profile.Code != http.StatusOK || !strings.Contains(string(profileBody), "student@demo.edu") {
		t.Fatalf("profile response status = %d or authenticated profile details missing", profile.Code)
	}

	dashboard := doRequest(http.MethodGet, "/api/v1/student/dashboard?student_id=S-002", "")
	var dashboardJSON map[string]any
	if dashboard.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d", dashboard.Code, http.StatusOK)
	}
	if err := json.NewDecoder(dashboard.Body).Decode(&dashboardJSON); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	if dashboardJSON["student_id"] != "S-001" || dashboardJSON["student_name"] != "Alex Morgan" {
		t.Fatalf("dashboard identity = (%v, %v), want authenticated account", dashboardJSON["student_id"], dashboardJSON["student_name"])
	}

	turn := doRequest(http.MethodPost, "/api/v1/student/sessions/session-s001-linear/turns", `{"student_id":"S-002","message":"I will check by substituting my answer."}`)
	var turnJSON map[string]any
	if turn.Code != http.StatusOK {
		t.Fatalf("tutor turn status = %d, want %d", turn.Code, http.StatusOK)
	}
	if err := json.NewDecoder(turn.Body).Decode(&turnJSON); err != nil {
		t.Fatalf("decode tutor turn: %v", err)
	}
	if _, ok := turnJSON["turn"]; !ok {
		t.Fatal("tutor response has no generated turn")
	}
}
