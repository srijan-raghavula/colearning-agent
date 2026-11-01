package routes

import (
	"net/http"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/controller"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type Dependencies struct {
	Health   controller.HealthController
	Student  controller.StudentController
	Faculty  controller.FacultyController
	Learning controller.LearningController
	Renderer *view.Renderer
	Accounts auth.UserStore
}

func Register(mux *http.ServeMux, deps Dependencies) {
	// Basic health and static assets
	mux.HandleFunc("/healthz", deps.Health.Handle)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("src/templates"))))

	// Product landing page
	mux.HandleFunc("GET /{$}", deps.Learning.LandingPage)

	// Authentication routes (login demo, teacher demo, etc.)
	authCtrl := controller.NewAuthController(deps.Renderer, deps.Accounts)
	RegisterAuth(mux, authCtrl)
	// Signup endpoint (uses controller logic)
	mux.HandleFunc("/api/auth/signup", authCtrl.Signup)

	mux.HandleFunc("GET /login", authCtrl.LoginPage)
	mux.HandleFunc("GET /signup", authCtrl.SignupPage)
	mux.HandleFunc("GET /logout", authCtrl.Logout)
	mux.HandleFunc("GET /profile", authenticated(authCtrl.ProfilePage))

	// Student routes
	mux.HandleFunc("/student", roleRequired("student", deps.Learning.StudentPage))
	mux.HandleFunc("/student/workspace", roleRequired("student", deps.Learning.StudentWorkspace))
	mux.HandleFunc("/student/session", roleRequired("student", deps.Learning.StartStudentSession))
	mux.HandleFunc("/student/turn", roleRequired("student", deps.Learning.SendStudentTurn))
	mux.HandleFunc("/student/check", roleRequired("student", deps.Learning.CheckStudentUnderstanding))

	// Teacher / faculty routes
	mux.HandleFunc("/teacher", teacherRoleRequired(deps.Learning.TeacherPage))
	mux.HandleFunc("/teacher/content", teacherRoleRequired(deps.Learning.TeacherContent))
	mux.HandleFunc("/teacher/materials", teacherRoleRequired(deps.Learning.PublishMaterial))
	mux.HandleFunc("/teacher/tutor-policy", teacherRoleRequired(deps.Learning.UpdateTutorPolicy))
	mux.HandleFunc("/faculty", teacherRoleRequired(deps.Learning.TeacherPage))
	mux.HandleFunc("/faculty/content", teacherRoleRequired(deps.Learning.TeacherContent))
	mux.HandleFunc("/faculty/materials", teacherRoleRequired(deps.Learning.PublishMaterial))
	mux.HandleFunc("/faculty/tutor-policy", teacherRoleRequired(deps.Learning.UpdateTutorPolicy))

	// Versioned JSON API
	handleAPI(mux, http.MethodGet, "/api/v1/student/dashboard", apiRoleRequired("student", deps.Learning.StudentDashboardJSON))
	handleAPI(mux, http.MethodPost, "/api/v1/student/sessions", apiRoleRequired("student", deps.Learning.StartStudentSessionJSON))
	handleAPI(mux, http.MethodGet, "/api/v1/student/sessions/{session_id}", apiRoleRequired("student", deps.Learning.StudentSessionJSON))
	handleAPI(mux, http.MethodPost, "/api/v1/student/sessions/{session_id}/turns", apiRoleRequired("student", deps.Learning.SendStudentTurnJSON))
	handleAPI(mux, http.MethodPost, "/api/v1/student/sessions/{session_id}/understanding-checks", apiRoleRequired("student", deps.Learning.CheckStudentUnderstandingJSON))
	handleAPI(mux, http.MethodGet, "/api/v1/teacher/dashboard", apiTeacherRoleRequired(deps.Learning.TeacherDashboardJSON))
	handleAPI(mux, http.MethodPost, "/api/v1/teacher/materials", apiTeacherRoleRequired(deps.Learning.PublishMaterialJSON))
	handleAPI(mux, http.MethodPut, "/api/v1/teacher/courses/{course_id}/tutor-policy", apiTeacherRoleRequired(deps.Learning.UpdateTutorPolicyJSON))

	// Fallback for unknown API routes
	mux.HandleFunc("/api/v1", apiNotFound)
	mux.HandleFunc("/api/v1/", apiNotFound)

	// Legacy assessment endpoints (not linked from UI)
	mux.HandleFunc("/student/submissions", roleRequired("student", deps.Student.SubmissionsPartial))
	mux.HandleFunc("/student/submit", roleRequired("student", deps.Student.Submit))
	mux.HandleFunc("/faculty/summary", teacherRoleRequired(deps.Faculty.SummaryPartial))
	mux.HandleFunc("/faculty/missing", teacherRoleRequired(deps.Faculty.MissingPartial))
}
