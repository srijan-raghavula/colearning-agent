package routes

import (
	"encoding/json"
	"net/http"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/controller"
)

func RegisterAuth(mux *http.ServeMux, authCtrl controller.AuthController) {
	mux.HandleFunc("POST /api/auth/login", authCtrl.Login)

	// This capability endpoint describes the live deterministic demo workflow.
	mux.HandleFunc("GET /api/agentic/tutor-plan", auth.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"agent":                  "TutorAgent",
			"provider":               "deterministic demo adapter",
			"runs_on":                "student tutor turns",
			"plan_steps":             []string{"load authorized context", "read progress", "filter published sources", "generate response", "validate response"},
			"read_only_tools":        []string{"read_course_context", "read_student_progress", "read_published_materials"},
			"formative_checker":      "runs on explicit student request",
			"execution_record_store": "SQLite agent_logs",
		})
	}))
}
