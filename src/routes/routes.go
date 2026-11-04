package routes

import (
	"net/http"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/controller"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type Dependencies struct {
	Health      controller.HealthController
	Student     controller.StudentController
	Faculty     controller.FacultyController
	Learning    controller.LearningController
	Courses     controller.CourseController
	Curriculum  controller.CurriculumController
	Enrollments controller.CourseEnrollmentStore
	Renderer    *view.Renderer
	Accounts    auth.UserStore
}

func Register(mux *http.ServeMux, deps Dependencies) {
	identityStore, _ := deps.Accounts.(auth.IdentityStore)
	// Basic health and static assets
	mux.HandleFunc("/healthz", deps.Health.Handle)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("src/templates"))))

	// Standalone frontend shell. Go remains the API and authentication server;
	// the browser renders product pages from JSON responses.
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("src/frontend/assets"))))

	// Authentication routes (login demo, teacher demo, etc.)
	authCtrl := controller.NewAuthController(deps.Renderer, deps.Accounts).WithEnrollmentStore(deps.Enrollments)
	RegisterAuth(mux, authCtrl, identityStore)
	// Signup endpoint (uses controller logic)
	mux.HandleFunc("/api/auth/signup", authCtrl.Signup)
	mux.HandleFunc("/api/auth/me", authCtrl.Me)
	mux.HandleFunc("/api/v1/auth/profile", apiAuthenticated(authCtrl.ProfileJSON, identityStore))

	mux.HandleFunc("GET /logout", authCtrl.Logout)
	// Compatibility profile endpoints remain available while the SPA uses
	// /api/auth/me for its session header.
	mux.HandleFunc("GET /profile", authenticated(authCtrl.ProfilePage, identityStore))
	mux.HandleFunc("POST /profile", authenticated(authCtrl.UpdateProfile, identityStore))
	mux.HandleFunc("POST /profile/password", authenticated(authCtrl.UpdatePassword, identityStore))

	// Legacy portal routes remain as a migration bridge for existing links.
	mux.HandleFunc("/student", roleRequired("student", deps.Learning.StudentPage, identityStore))
	mux.HandleFunc("/student/workspace", roleRequired("student", deps.Learning.StudentWorkspace, identityStore))
	mux.HandleFunc("/student/session", roleRequired("student", deps.Learning.StartStudentSession, identityStore))
	mux.HandleFunc("/student/turn", roleRequired("student", deps.Learning.SendStudentTurn, identityStore))
	mux.HandleFunc("/student/check", roleRequired("student", deps.Learning.CheckStudentUnderstanding, identityStore))
	mux.HandleFunc("GET /student/courses", roleRequired("student", deps.Courses.StudentCatalog, identityStore))
	mux.HandleFunc("GET /student/courses/{course_id}", roleRequired("student", deps.Courses.StudentCourseDetails, identityStore))
	mux.HandleFunc("POST /student/courses/{course_id}/enroll", roleRequired("student", deps.Courses.RequestEnrollment, identityStore))
	mux.HandleFunc("/teacher", teacherRoleRequired(deps.Learning.TeacherPage, identityStore))
	mux.HandleFunc("/teacher/content", teacherRoleRequired(deps.Learning.TeacherContent, identityStore))
	mux.HandleFunc("/teacher/materials", teacherRoleRequired(deps.Learning.PublishMaterial, identityStore))
	mux.HandleFunc("/teacher/tutor-policy", teacherRoleRequired(deps.Learning.UpdateTutorPolicy, identityStore))
	mux.HandleFunc("/faculty", teacherRoleRequired(deps.Learning.TeacherPage, identityStore))
	mux.HandleFunc("/faculty/content", teacherRoleRequired(deps.Learning.TeacherContent, identityStore))
	mux.HandleFunc("/faculty/materials", teacherRoleRequired(deps.Learning.PublishMaterial, identityStore))
	mux.HandleFunc("/faculty/tutor-policy", teacherRoleRequired(deps.Learning.UpdateTutorPolicy, identityStore))
	mux.HandleFunc("GET /teacher/enrollments", teacherRoleRequired(deps.Courses.TeacherRequests, identityStore))
	mux.HandleFunc("POST /teacher/enrollments", teacherRoleRequired(deps.Courses.ReviewEnrollment, identityStore))
	mux.HandleFunc("GET /teacher/courses", teacherRoleRequired(deps.Courses.TeacherCourseList, identityStore))
	mux.HandleFunc("GET /teacher/courses/new", teacherRoleRequired(deps.Courses.TeacherCourseEditor, identityStore))
	mux.HandleFunc("POST /teacher/courses", teacherRoleRequired(deps.Courses.SaveTeacherCourse, identityStore))
	mux.HandleFunc("GET /teacher/courses/{course_id}", teacherRoleRequired(deps.Courses.TeacherCourseEditor, identityStore))
	mux.HandleFunc("POST /teacher/courses/{course_id}", teacherRoleRequired(deps.Courses.SaveTeacherCourse, identityStore))
	mux.HandleFunc("GET /teacher/courses/{course_id}/curriculum", teacherRoleRequired(deps.Curriculum.Page, identityStore))
	mux.HandleFunc("POST /teacher/courses/{course_id}/curriculum/save", teacherRoleRequired(deps.Curriculum.SaveAll, identityStore))
	mux.HandleFunc("POST /teacher/courses/{course_id}/curriculum/generate", teacherRoleRequired(deps.Curriculum.GenerateDraft, identityStore))
	mux.HandleFunc("POST /teacher/curriculum/modules", teacherRoleRequired(deps.Curriculum.SaveModule, identityStore))
	mux.HandleFunc("POST /teacher/curriculum/chapters", teacherRoleRequired(deps.Curriculum.SaveChapter, identityStore))
	mux.HandleFunc("POST /teacher/curriculum/lessons", teacherRoleRequired(deps.Curriculum.SaveLesson, identityStore))
	mux.HandleFunc("POST /teacher/curriculum/content", teacherRoleRequired(deps.Curriculum.SaveContent, identityStore))

	// Versioned JSON API
	handleAPI(mux, http.MethodGet, "/api/v1/student/dashboard", apiRoleRequired("student", deps.Learning.StudentDashboardJSON, identityStore))
	handleAPI(mux, http.MethodPost, "/api/v1/student/sessions", apiRoleRequired("student", deps.Learning.StartStudentSessionJSON, identityStore))
	handleAPI(mux, http.MethodGet, "/api/v1/student/sessions/{session_id}", apiRoleRequired("student", deps.Learning.StudentSessionJSON, identityStore))
	handleAPI(mux, http.MethodPost, "/api/v1/student/sessions/{session_id}/turns", apiRoleRequired("student", deps.Learning.SendStudentTurnJSON, identityStore))
	handleAPI(mux, http.MethodPost, "/api/v1/student/agent/next", apiRoleRequired("student", deps.Learning.RunStudentAgentJSON, identityStore))
	handleAPI(mux, http.MethodPost, "/api/v1/student/sessions/{session_id}/understanding-checks", apiRoleRequired("student", deps.Learning.CheckStudentUnderstandingJSON, identityStore))
	handleAPI(mux, http.MethodGet, "/api/v1/teacher/dashboard", apiTeacherRoleRequired(deps.Learning.TeacherDashboardJSON, identityStore))
	handleAPI(mux, http.MethodGet, "/api/v1/student/courses/{course_id}/curriculum", apiRoleRequired("student", deps.Curriculum.StructureJSON, identityStore))
	handleAPI(mux, http.MethodGet, "/api/v1/teacher/courses/{course_id}/curriculum", apiTeacherRoleRequired(deps.Curriculum.StructureJSON, identityStore))
	handleAPI(mux, http.MethodPost, "/api/v1/teacher/courses", apiTeacherRoleRequired(deps.Courses.CreateCourseJSON, identityStore))
	handleAPI(mux, http.MethodPost, "/api/v1/teacher/materials", apiTeacherRoleRequired(deps.Learning.PublishMaterialJSON, identityStore))
	handleAPI(mux, http.MethodPost, "/api/v1/teacher/materials/pdf", apiTeacherRoleRequired(deps.Learning.UploadMaterialPDF, identityStore))
	handleAPI(mux, http.MethodPut, "/api/v1/teacher/courses/{course_id}/tutor-policy", apiTeacherRoleRequired(deps.Learning.UpdateTutorPolicyJSON, identityStore))

	// Fallback for unknown API routes
	mux.HandleFunc("/api/v1", apiNotFound)
	mux.HandleFunc("/api/v1/", apiNotFound)

	// Legacy assessment endpoints (not linked from UI)
	mux.HandleFunc("/student/submissions", roleRequired("student", deps.Student.SubmissionsPartial, identityStore))
	mux.HandleFunc("/student/submit", roleRequired("student", deps.Student.Submit, identityStore))
	mux.HandleFunc("/faculty/summary", teacherRoleRequired(deps.Faculty.SummaryPartial, identityStore))
	mux.HandleFunc("/faculty/missing", teacherRoleRequired(deps.Faculty.MissingPartial, identityStore))

	mux.HandleFunc("/", frontendShell)
}

func frontendShell(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "src/frontend/index.html")
}
