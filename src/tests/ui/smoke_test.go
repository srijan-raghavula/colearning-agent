package ui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/srijan-raghavula/colearning-agent/src/bootstrap"
	"github.com/srijan-raghavula/colearning-agent/src/routes"
)

func TestLandingPageSmoke(t *testing.T) {
	projectRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err != nil {
		t.Fatalf("locate project root: %v", err)
	}
	t.Chdir(projectRoot)
	t.Setenv("COLEARN_DB_PATH", filepath.Join(t.TempDir(), "colearning.db"))
	t.Setenv("LEARNING_STORE_PATH", "off")
	app, err := bootstrap.NewApp()
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if app.Close != nil {
		defer app.Close()
	}
	mux := http.NewServeMux()
	routes.Register(mux, app.Routes)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
