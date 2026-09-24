package routes

import (
	"net/http"

	"github.com/srijan-raghavula/colearning-agent/src/controller"
)

type Dependencies struct {
	Health   controller.HealthController
	Student  controller.StudentController
	Faculty  controller.FacultyController
	Learning controller.LearningController
}

func Register(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("/healthz", deps.Health.Handle)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("src/templates"))))

	// Product surfaces. The demo role is carried by a query parameter or
	// X-Role header until a real identity provider is introduced.
	mux.HandleFunc("/", deps.Learning.LandingPage)
	mux.HandleFunc("/student", roleRequired("student", deps.Learning.StudentPage))
	mux.HandleFunc("/student/workspace", roleRequired("student", deps.Learning.StudentWorkspace))
	mux.HandleFunc("/student/session", roleRequired("student", deps.Learning.StartStudentSession))
	mux.HandleFunc("/student/turn", roleRequired("student", deps.Learning.SendStudentTurn))
	mux.HandleFunc("/student/check", roleRequired("student", deps.Learning.CheckStudentUnderstanding))

	// Teacher is the canonical product route. /faculty remains an alias for
	// compatibility with the original assessment portal.
	mux.HandleFunc("/teacher", teacherRoleRequired(deps.Learning.TeacherPage))
	mux.HandleFunc("/teacher/content", teacherRoleRequired(deps.Learning.TeacherContent))
	mux.HandleFunc("/teacher/materials", teacherRoleRequired(deps.Learning.PublishMaterial))
	mux.HandleFunc("/teacher/tutor-policy", teacherRoleRequired(deps.Learning.UpdateTutorPolicy))
	mux.HandleFunc("/faculty", teacherRoleRequired(deps.Learning.TeacherPage))
	mux.HandleFunc("/faculty/content", teacherRoleRequired(deps.Learning.TeacherContent))
	mux.HandleFunc("/faculty/materials", teacherRoleRequired(deps.Learning.PublishMaterial))
	mux.HandleFunc("/faculty/tutor-policy", teacherRoleRequired(deps.Learning.UpdateTutorPolicy))

	// Versioned JSON API. The HTML portal and API intentionally share the same
	// use cases so the UI cannot accidentally grow a second business path.
	mux.HandleFunc("GET /api/v1/student/dashboard", roleRequired("student", deps.Learning.StudentDashboardJSON))
	mux.HandleFunc("POST /api/v1/student/sessions", roleRequired("student", deps.Learning.StartStudentSessionJSON))
	mux.HandleFunc("GET /api/v1/student/sessions/{session_id}", roleRequired("student", deps.Learning.StudentSessionJSON))
	mux.HandleFunc("POST /api/v1/student/sessions/{session_id}/turns", roleRequired("student", deps.Learning.SendStudentTurnJSON))
	mux.HandleFunc("POST /api/v1/student/sessions/{session_id}/understanding-checks", roleRequired("student", deps.Learning.CheckStudentUnderstandingJSON))
	mux.HandleFunc("GET /api/v1/teacher/dashboard", teacherRoleRequired(deps.Learning.TeacherDashboardJSON))
	mux.HandleFunc("POST /api/v1/teacher/materials", teacherRoleRequired(deps.Learning.PublishMaterialJSON))
	mux.HandleFunc("PUT /api/v1/teacher/courses/{course_id}/tutor-policy", teacherRoleRequired(deps.Learning.UpdateTutorPolicyJSON))

	// Legacy assessment endpoints remain available while the co-learning
	// surface is proven, but they are no longer linked from the product UI.
	mux.HandleFunc("/student/submissions", roleRequired("student", deps.Student.SubmissionsPartial))
	mux.HandleFunc("/student/submit", roleRequired("student", deps.Student.Submit))
	mux.HandleFunc("/faculty/summary", roleRequired("faculty", deps.Faculty.SummaryPartial))
	mux.HandleFunc("/faculty/missing", roleRequired("faculty", deps.Faculty.MissingPartial))
}
