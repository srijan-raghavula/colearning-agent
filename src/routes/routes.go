package routes

import (
	"net/http"

	"github.com/srijan-raghavula/colearning-agent/src/controller"
)

type Dependencies struct {
	Health  controller.HealthController
	Student controller.StudentController
	Faculty controller.FacultyController
}

func Register(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("/healthz", deps.Health.Handle)

	mux.HandleFunc("/student", roleRequired("student", deps.Student.Page))
	mux.HandleFunc("/student/submissions", roleRequired("student", deps.Student.SubmissionsPartial))
	mux.HandleFunc("/student/submit", roleRequired("student", deps.Student.Submit))

	mux.HandleFunc("/faculty", roleRequired("faculty", deps.Faculty.Page))
	mux.HandleFunc("/faculty/summary", roleRequired("faculty", deps.Faculty.SummaryPartial))
	mux.HandleFunc("/faculty/missing", roleRequired("faculty", deps.Faculty.MissingPartial))
}
