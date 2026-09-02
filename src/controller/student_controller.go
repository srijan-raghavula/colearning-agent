package controller

import (
	"net/http"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/usecase"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type StudentController struct {
	portal   usecase.StudentPortal
	renderer *view.Renderer
}

func NewStudentController(portal usecase.StudentPortal, renderer *view.Renderer) StudentController {
	return StudentController{portal: portal, renderer: renderer}
}

func (c StudentController) Page(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = "S-001"
	}
	c.renderDashboard(w, r, studentID, "student/page.html")
}

func (c StudentController) SubmissionsPartial(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = "S-001"
	}
	c.renderDashboard(w, r, studentID, "student/submissions.html")
}

func (c StudentController) Submit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	dueAt, err := time.Parse(time.RFC3339, r.FormValue("due_at"))
	if err != nil {
		http.Error(w, "due_at must be RFC3339", http.StatusBadRequest)
		return
	}

	studentID := r.FormValue("student_id")
	if studentID == "" {
		studentID = "S-001"
	}

	_, err = c.portal.Submit(r.Context(), usecase.SubmitRequest{
		ID:         r.FormValue("id"),
		StudentID:  studentID,
		SubjectID:  r.FormValue("subject_id"),
		TopicID:    r.FormValue("topic_id"),
		Title:      r.FormValue("title"),
		Attachment: r.FormValue("attachment"),
		DueAt:      dueAt,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.renderDashboard(w, r, studentID, "student/submissions.html")
}

func (c StudentController) renderDashboard(w http.ResponseWriter, r *http.Request, studentID, templateName string) {
	dashboard, err := c.portal.Dashboard(r.Context(), studentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.renderer.Render(w, templateName, map[string]any{
		"Role":      "student",
		"StudentID": studentID,
		"Dashboard": dashboard,
	})
}
