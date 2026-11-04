package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/service"
	"github.com/srijan-raghavula/colearning-agent/src/usecase"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type LearningController struct {
	student     usecase.StudentLearningPortal
	teacher     usecase.TeacherLearningPortal
	renderer    *view.Renderer
	accounts    accountLookup
	enrollments CourseEnrollmentStore
	courses     courseCatalog
}

type teacherMaterialView struct {
	Material domain.LearningMaterial
	Target   string
}

type accountLookup interface {
	FindAccountByID(context.Context, string) (auth.Account, bool, error)
}

func NewLearningController(student usecase.StudentLearningPortal, teacher usecase.TeacherLearningPortal, renderer *view.Renderer) LearningController {
	return LearningController{student: student, teacher: teacher, renderer: renderer}
}

func (c LearningController) WithAccountLookup(accounts accountLookup) LearningController {
	c.accounts = accounts
	return c
}

func (c LearningController) WithEnrollmentStore(enrollments CourseEnrollmentStore) LearningController {
	c.enrollments = enrollments
	return c
}

func (c LearningController) WithCourseCatalog(courses courseCatalog) LearningController {
	c.courses = courses
	return c
}

func (c LearningController) requireOwnedCourse(w http.ResponseWriter, r *http.Request, courseID string) bool {
	if c.courses == nil {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeAPIError(w, http.StatusInternalServerError, errors.New("course ownership verification is unavailable"))
		} else {
			http.Error(w, "Course ownership verification is unavailable", http.StatusInternalServerError)
		}
		return false
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok || (identity.Role != "teacher" && identity.Role != "faculty") {
		writeAPIError(w, http.StatusForbidden, errors.New("teacher identity required"))
		return false
	}
	course, found, err := c.courses.FindCourse(r.Context(), courseID)
	if err != nil {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeAPIError(w, http.StatusInternalServerError, errors.New("could not verify course ownership"))
		} else {
			http.Error(w, "Could not verify course ownership", http.StatusInternalServerError)
		}
		return false
	}
	if !found || course.TeacherID != identity.ID {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeAPIError(w, http.StatusForbidden, errors.New("course is not owned by this teacher"))
		} else {
			http.Error(w, "Course is not owned by this teacher", http.StatusForbidden)
		}
		return false
	}
	return true
}

func (c LearningController) requireCourseEnrollment(w http.ResponseWriter, r *http.Request, courseID string) bool {
	if c.enrollments == nil {
		return true
	}
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok || identity.Role != "student" {
		return false
	}
	enrollment, found, err := c.enrollments.FindCourseEnrollment(r.Context(), identity.ID, courseID)
	if err != nil {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeAPIError(w, http.StatusInternalServerError, errors.New("could not verify course enrollment"))
		} else {
			http.Error(w, "Could not verify course enrollment", http.StatusInternalServerError)
		}
		return false
	}
	if found && enrollment.Status == domain.EnrollmentApproved {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeAPIError(w, http.StatusForbidden, errors.New("approved course enrollment required"))
	} else {
		http.Redirect(w, r, "/student/courses/"+url.PathEscape(courseID), http.StatusSeeOther)
	}
	return false
}

type learningWorkspaceView struct {
	Dashboard       domain.StudentLearningDashboard
	Session         domain.LearningSession
	Turns           []domain.LearningTurn
	Concept         domain.LearningConcept
	Materials       []domain.LearningMaterial
	SelectedMode    domain.TutorMode
	AvailableModes  []domain.TutorMode
	ConceptArchived bool
	Check           *domain.UnderstandingCheck
	Notice          string
	Error           string
}

func (c LearningController) LandingPage(w http.ResponseWriter, r *http.Request) {
	c.renderer.Render(w, "app/landing.html", map[string]any{
		"Role": "guest",
	})
}

func (c LearningController) StudentPage(w http.ResponseWriter, r *http.Request) {
	studentID := studentIDFromRequest(r)
	courseID := r.URL.Query().Get("course_id")
	if courseID == "" {
		var ok bool
		courseID, ok = c.firstApprovedCourse(r, studentID)
		if !ok {
			http.Redirect(w, r, "/student/courses", http.StatusSeeOther)
			return
		}
	}
	dashboard, err := c.student.DashboardForCourse(r.Context(), studentID, courseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !c.requireCourseEnrollment(w, r, dashboard.Course.ID) {
		return
	}
	if len(dashboard.Concepts) == 0 {
		if dashboard.Course.ID != "" {
			http.Redirect(w, r, "/student/courses/"+url.PathEscape(dashboard.Course.ID), http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/student/courses", http.StatusSeeOther)
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.StudentName = identity.Name
	}
	teacherTitle := ""
	if c.accounts != nil && dashboard.Course.TeacherID != "" {
		teacher, found, lookupErr := c.accounts.FindAccountByID(r.Context(), dashboard.Course.TeacherID)
		if lookupErr == nil && found {
			dashboard.Course.TeacherName = teacher.Name
			teacherTitle = teacher.TeachingTitle
			if teacherTitle == "" {
				teacherTitle = teacher.Department
			}
		}
	}
	var enrolledCourses []domain.CourseEnrollment
	var approvedCourses []domain.CourseEnrollment
	if identity, ok := auth.IdentityFromContext(r.Context()); ok && c.enrollments != nil {
		enrolledCourses, err = c.enrollments.ListStudentEnrollments(r.Context(), identity.ID)
		if err != nil {
			http.Error(w, "Could not load your courses", http.StatusInternalServerError)
			return
		}
		for _, enrollment := range enrolledCourses {
			if enrollment.Status == domain.EnrollmentApproved {
				approvedCourses = append(approvedCourses, enrollment)
			}
		}
	}
	c.renderer.Render(w, "student/dashboard.html", map[string]any{
		"Role":            "student",
		"StudentID":       dashboard.StudentID,
		"Dashboard":       dashboard,
		"TeacherTitle":    teacherTitle,
		"EnrolledCourses": enrolledCourses,
		"ApprovedCourses": approvedCourses,
		"Notice":          r.URL.Query().Get("notice"),
	})
}

func (c LearningController) StudentWorkspace(w http.ResponseWriter, r *http.Request) {
	studentID := studentIDFromRequest(r)
	courseID := r.URL.Query().Get("course_id")
	if courseID == "" {
		var ok bool
		courseID, ok = c.firstApprovedCourse(r, studentID)
		if !ok {
			http.Redirect(w, r, "/student/courses", http.StatusSeeOther)
			return
		}
	}
	dashboard, err := c.student.DashboardForCourse(r.Context(), studentID, courseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.StudentName = identity.Name
	}
	w.Header().Set("Cache-Control", "no-store")

	sessionID := r.URL.Query().Get("session_id")
	conceptID := r.URL.Query().Get("concept_id")
	if sessionID == "" {
		courseID := r.URL.Query().Get("course_id")
		if courseID == "" {
			courseID = dashboard.Course.ID
		}
		if !c.requireCourseEnrollment(w, r, courseID) {
			return
		}
		session, startErr := c.student.StartOrResume(r.Context(), usecase.StartLearningRequest{
			StudentID: studentID,
			CourseID:  courseID,
			ConceptID: conceptID,
			Mode:      domain.TutorMode(r.URL.Query().Get("mode")),
		})
		if startErr != nil {
			http.Error(w, startErr.Error(), http.StatusBadRequest)
			return
		}
		redirect := "/student/workspace?session_id=" + url.QueryEscape(session.ID)
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}

	session, turns, err := c.student.Session(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !c.requireCourseEnrollment(w, r, session.CourseID) {
		return
	}
	if session.StudentID != studentID {
		http.Error(w, "learning session does not belong to this student", http.StatusForbidden)
		return
	}
	if dashboard.Course.ID != session.CourseID {
		dashboard, err = c.student.DashboardForCourse(r.Context(), studentID, session.CourseID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if identity, ok := auth.IdentityFromContext(r.Context()); ok {
			dashboard.StudentName = identity.Name
		}
	}
	concept, archived, conceptErr := c.sessionConcept(r.Context(), &dashboard, session.ConceptID)
	if conceptErr != nil {
		http.Error(w, "Could not load the session concept", http.StatusInternalServerError)
		return
	}
	selectedMode := displayTutorMode(dashboard.Policy, domain.TutorMode(r.URL.Query().Get("mode")), session.Mode)
	materials := materialsForConcept(dashboard.Materials, concept.ID)
	if len(materials) == 0 {
		materials = dashboard.Materials
	}
	check, checkErr := c.student.LatestCheck(r.Context(), session.ID)
	if checkErr != nil {
		http.Error(w, checkErr.Error(), http.StatusInternalServerError)
		return
	}
	c.renderer.Render(w, "student/workspace.html", map[string]any{
		"Role": "student",
		"Workspace": learningWorkspaceView{
			Dashboard:       dashboard,
			Session:         session,
			Turns:           turns,
			Concept:         concept,
			Materials:       materials,
			SelectedMode:    selectedMode,
			AvailableModes:  availableTutorModes(dashboard.Policy),
			ConceptArchived: archived,
			Check:           check,
			Notice:          r.URL.Query().Get("notice"),
			Error:           r.URL.Query().Get("error"),
		},
	})
}

func (c LearningController) firstApprovedCourse(r *http.Request, studentID string) (string, bool) {
	if c.enrollments == nil {
		return "", false
	}
	items, err := c.enrollments.ListStudentEnrollments(r.Context(), studentID)
	if err != nil {
		return "", false
	}
	for _, item := range items {
		if item.Status == domain.EnrollmentApproved {
			return item.CourseID, true
		}
	}
	return "", false
}

func (c LearningController) StartStudentSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	studentID := studentIDFromRequest(r)
	courseID := r.FormValue("course_id")
	if courseID == "" {
		dashboard, err := c.student.Dashboard(r.Context(), studentID)
		if err != nil {
			http.Error(w, "Could not load course", http.StatusInternalServerError)
			return
		}
		courseID = dashboard.Course.ID
	}
	if !c.requireCourseEnrollment(w, r, courseID) {
		return
	}
	session, err := c.student.StartOrResume(r.Context(), usecase.StartLearningRequest{
		StudentID: studentID,
		CourseID:  courseID,
		ConceptID: r.FormValue("concept_id"),
		Mode:      domain.TutorMode(r.FormValue("mode")),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirect := "/student/workspace?session_id=" + url.QueryEscape(session.ID) + "&notice=Session%20ready"
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func (c LearningController) SendStudentTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		if r.Header.Get("HX-Request") != "" {
			c.renderTurnError(w, r, "invalid form")
		} else {
			http.Redirect(w, r, workspaceLocation(r.FormValue("session_id"), "", "Please check your message and try again."), http.StatusSeeOther)
		}
		return
	}
	studentID := studentIDFromRequest(r)
	sessionID := r.FormValue("session_id")
	session, _, sessionErr := c.student.Session(r.Context(), sessionID)
	if sessionErr != nil || session.StudentID != studentID {
		http.Error(w, "learning session not found", http.StatusNotFound)
		return
	}
	if !c.requireCourseEnrollment(w, r, session.CourseID) {
		return
	}
	mode := domain.TutorMode(r.FormValue("mode"))
	message := strings.TrimSpace(r.FormValue("message"))
	result, err := c.student.Continue(r.Context(), usecase.SendLearningTurnRequest{
		StudentID: studentID,
		SessionID: sessionID,
		Mode:      mode,
		Message:   message,
	})
	if err != nil {
		const message = "Your tutor is temporarily unavailable. Your message is saved; submit it again to retry."
		if r.Header.Get("HX-Request") != "" {
			c.renderTurnError(w, r, message)
			return
		}
		http.Redirect(w, r, workspaceLocation(sessionID, "", message), http.StatusSeeOther)
		return
	}

	workspace, err := c.workspaceView(r.Context(), studentID, result.Session.ID, result.Turn.Mode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") != "" {
		c.renderer.Render(w, "student/workspace_conversation.html", map[string]any{
			"Role":      "student",
			"Workspace": workspace,
		})
		return
	}
	http.Redirect(w, r, workspaceLocation(result.Session.ID, "Your turn was sent.", ""), http.StatusSeeOther)
}

func (c LearningController) CheckStudentUnderstanding(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, workspaceLocation(r.FormValue("session_id"), "", "Please try the understanding check again."), http.StatusSeeOther)
		return
	}
	studentID := studentIDFromRequest(r)
	sessionID := r.FormValue("session_id")
	session, _, sessionErr := c.student.Session(r.Context(), sessionID)
	if sessionErr != nil || session.StudentID != studentID {
		http.Error(w, "learning session not found", http.StatusNotFound)
		return
	}
	if !c.requireCourseEnrollment(w, r, session.CourseID) {
		return
	}
	_, err := c.student.CheckUnderstanding(r.Context(), sessionID, studentID)
	if err != nil {
		const message = "The understanding check could not be completed. Please try again."
		if r.Header.Get("HX-Request") != "" {
			c.renderTurnError(w, r, message)
			return
		}
		http.Redirect(w, r, workspaceLocation(r.FormValue("session_id"), "", message), http.StatusSeeOther)
		return
	}
	workspace, err := c.workspaceView(r.Context(), studentID, sessionID, domain.TutorMode(r.FormValue("mode")))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") != "" {
		c.renderer.Render(w, "student/workspace_conversation.html", map[string]any{
			"Role":      "student",
			"Workspace": workspace,
		})
		return
	}
	http.Redirect(w, r, workspaceLocation(sessionID, "Understanding check complete.", ""), http.StatusSeeOther)
}

func workspaceLocation(sessionID, notice, message string) string {
	query := url.Values{}
	query.Set("session_id", sessionID)
	if notice != "" {
		query.Set("notice", notice)
	}
	if message != "" {
		query.Set("error", message)
	}
	return "/student/workspace?" + query.Encode()
}

func (c LearningController) renderTurnError(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusBadRequest)
	if r.Header.Get("HX-Request") != "" {
		_, _ = io.WriteString(w, `<div class="inline-error">`+templateEscape(message)+`</div>`)
		return
	}
	_, _ = io.WriteString(w, message)
}

func (c LearningController) workspaceView(ctx context.Context, studentID, sessionID string, mode domain.TutorMode) (learningWorkspaceView, error) {
	session, turns, err := c.student.Session(ctx, sessionID)
	if err != nil {
		return learningWorkspaceView{}, err
	}
	// A learner can have sessions in several courses. Build the refreshed read
	// model from the session's course so the transcript, policy, progress, and
	// sources all describe the same course after a turn or understanding check.
	dashboard, err := c.student.DashboardForCourse(ctx, studentID, session.CourseID)
	if err != nil {
		return learningWorkspaceView{}, err
	}
	if identity, ok := auth.IdentityFromContext(ctx); ok {
		dashboard.StudentName = identity.Name
	}
	concept, archived, err := c.sessionConcept(ctx, &dashboard, session.ConceptID)
	if err != nil {
		return learningWorkspaceView{}, err
	}
	materials := materialsForConcept(dashboard.Materials, concept.ID)
	if len(materials) == 0 {
		materials = dashboard.Materials
	}
	mode = displayTutorMode(dashboard.Policy, mode, session.Mode)
	check, err := c.student.LatestCheck(ctx, session.ID)
	if err != nil {
		return learningWorkspaceView{}, err
	}
	return learningWorkspaceView{
		Dashboard:       dashboard,
		Session:         session,
		Turns:           turns,
		Concept:         concept,
		Materials:       materials,
		SelectedMode:    mode,
		AvailableModes:  availableTutorModes(dashboard.Policy),
		ConceptArchived: archived,
		Check:           check,
	}, nil
}

func (c LearningController) sessionConcept(ctx context.Context, dashboard *domain.StudentLearningDashboard, conceptID string) (domain.LearningConcept, bool, error) {
	concept := findConcept(dashboard.Concepts, conceptID)
	if concept.ID == "" && c.courses != nil {
		var found bool
		var err error
		concept, found, err = c.courses.FindConcept(ctx, conceptID)
		if err != nil {
			return domain.LearningConcept{}, false, err
		}
		if !found {
			return domain.LearningConcept{}, false, errors.New("concept was not found")
		}
	}
	if concept.ID != "" {
		dashboard.FeaturedConcept = concept
		if concept.Archived {
			dashboard.FeaturedProgress = domain.ConceptProgress{
				StudentID: dashboard.StudentID, CourseID: dashboard.Course.ID, ConceptID: concept.ID,
				Status: domain.ConceptProgressNotStarted, NextAction: "Review archived conversation history",
			}
			dashboard.Materials = nil
		}
		for _, card := range dashboard.Concepts {
			if card.Concept.ID == conceptID {
				dashboard.FeaturedProgress = card.Progress
				break
			}
		}
	}
	return concept, concept.Archived, nil
}

var tutorModeOptions = []domain.TutorMode{
	domain.TutorModeGuided,
	domain.TutorModeSocratic,
	domain.TutorModeDirect,
	domain.TutorModePractice,
	domain.TutorModeHint,
	domain.TutorModeDiagnostic,
}

func availableTutorModes(policy domain.TutorPolicy) []domain.TutorMode {
	if len(policy.AllowedModes) == 0 {
		return tutorModeOptions
	}
	modes := make([]domain.TutorMode, 0, len(policy.AllowedModes))
	for _, mode := range tutorModeOptions {
		if policy.AllowsMode(mode) {
			modes = append(modes, mode)
		}
	}
	return modes
}

func displayTutorMode(policy domain.TutorPolicy, requested, current domain.TutorMode) domain.TutorMode {
	for _, mode := range availableTutorModes(policy) {
		if mode == requested {
			return requested
		}
	}
	for _, mode := range availableTutorModes(policy) {
		if mode == current {
			return current
		}
	}
	if policy.AllowsMode(policy.DefaultMode) && policy.DefaultMode.IsValid() {
		return policy.DefaultMode
	}
	if modes := availableTutorModes(policy); len(modes) > 0 {
		return modes[0]
	}
	return domain.TutorModeGuided
}

func (c LearningController) StudentDashboardJSON(w http.ResponseWriter, r *http.Request) {
	studentID := studentIDFromRequest(r)
	var enrollments []domain.CourseEnrollment
	if c.enrollments != nil {
		var err error
		enrollments, err = c.enrollments.ListStudentEnrollments(r.Context(), studentID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, errors.New("could not load student courses"))
			return
		}
		hasApprovedCourse := false
		for _, enrollment := range enrollments {
			if enrollment.Status == domain.EnrollmentApproved {
				hasApprovedCourse = true
				break
			}
		}
		if !hasApprovedCourse && r.URL.Query().Get("course_id") == "" {
			dashboard := domain.StudentLearningDashboard{StudentID: studentID}
			if identity, ok := auth.IdentityFromContext(r.Context()); ok {
				dashboard.StudentName = identity.Name
			}
			writeJSON(w, http.StatusOK, toAPIStudentDashboard(dashboard, enrollments))
			return
		}
	}
	dashboard, err := c.student.DashboardForCourse(r.Context(), studentID, r.URL.Query().Get("course_id"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	if !c.requireCourseEnrollment(w, r, dashboard.Course.ID) {
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.StudentName = identity.Name
	}
	writeJSON(w, http.StatusOK, toAPIStudentDashboard(dashboard, enrollments))
}

func (c LearningController) StartStudentSessionJSON(w http.ResponseWriter, r *http.Request) {
	var req usecase.StartLearningRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// Derive ownership from the authenticated identity, never the request body.
	req.StudentID = studentIDFromRequest(r)
	if req.CourseID == "" {
		dashboard, err := c.student.Dashboard(r.Context(), req.StudentID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		req.CourseID = dashboard.Course.ID
	}
	if !c.requireCourseEnrollment(w, r, req.CourseID) {
		return
	}
	session, err := c.student.StartOrResume(r.Context(), req)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPISession(session))
}

func (c LearningController) StudentSessionJSON(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("session_id")
	}
	session, turns, err := c.student.Session(r.Context(), sessionID)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	if session.StudentID != studentIDFromRequest(r) {
		writeAPIError(w, http.StatusForbidden, errors.New("learning session does not belong to this student"))
		return
	}
	if !c.requireCourseEnrollment(w, r, session.CourseID) {
		return
	}
	apiTurns := make([]apiTurn, 0, len(turns))
	for _, turn := range turns {
		apiTurns = append(apiTurns, toAPITurn(turn))
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": toAPISession(session), "turns": apiTurns})
}

func (c LearningController) SendStudentTurnJSON(w http.ResponseWriter, r *http.Request) {
	var req usecase.SendLearningTurnRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	// Resolve the learner from the authenticated identity rather than trusting
	// the JSON body's student_id field.
	req.StudentID = studentIDFromRequest(r)
	req.SessionID = r.PathValue("session_id")
	if req.SessionID == "" {
		req.SessionID = r.URL.Query().Get("session_id")
	}
	session, _, sessionErr := c.student.Session(r.Context(), req.SessionID)
	if sessionErr != nil || session.StudentID != req.StudentID {
		writeAPIError(w, http.StatusNotFound, errors.New("learning session not found"))
		return
	}
	if !c.requireCourseEnrollment(w, r, session.CourseID) {
		return
	}
	result, err := c.student.Continue(r.Context(), req)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, errors.New("tutor is temporarily unavailable; retry the same message"))
		return
	}
	writeJSON(w, http.StatusOK, toAPITurnResult(result))
}

func (c LearningController) RunStudentAgentJSON(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	var req struct {
		CourseID  string           `json:"course_id,omitempty"`
		SessionID string           `json:"session_id,omitempty"`
		Mode      domain.TutorMode `json:"mode,omitempty"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	dashboard, err := c.student.DashboardForCourse(r.Context(), identity.ID, req.CourseID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	sessionID := req.SessionID
	if sessionID == "" && dashboard.ActiveSession != nil {
		sessionID = dashboard.ActiveSession.ID
	}
	if sessionID == "" {
		session, startErr := c.student.StartOrResume(r.Context(), usecase.StartLearningRequest{
			StudentID: identity.ID, CourseID: dashboard.Course.ID,
			ConceptID: dashboard.FeaturedConcept.ID, Mode: req.Mode,
		})
		if startErr != nil {
			writeAPIError(w, http.StatusBadRequest, startErr)
			return
		}
		sessionID = session.ID
	}
	if req.Mode == "" {
		req.Mode = dashboard.Policy.DefaultMode
	}
	result, err := c.student.Continue(r.Context(), usecase.SendLearningTurnRequest{
		StudentID: identity.ID, SessionID: sessionID, Mode: req.Mode,
		Message: "Act as my learning coach. Inspect my current progress and course context, choose the most useful next learning action, explain it briefly, and end with one concrete question or activity for me.",
	})
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITurnResult(result))
}

func (c LearningController) CheckStudentUnderstandingJSON(w http.ResponseWriter, r *http.Request) {
	studentID := studentIDFromRequest(r)
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		sessionID = r.URL.Query().Get("session_id")
	}
	session, _, sessionErr := c.student.Session(r.Context(), sessionID)
	if sessionErr != nil || session.StudentID != studentID {
		writeAPIError(w, http.StatusNotFound, errors.New("learning session not found"))
		return
	}
	if !c.requireCourseEnrollment(w, r, session.CourseID) {
		return
	}
	check, err := c.student.CheckUnderstanding(r.Context(), sessionID, studentID)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, errors.New("understanding check is temporarily unavailable"))
		return
	}
	writeJSON(w, http.StatusCreated, toAPICheck(check))
}

func (c LearningController) TeacherPage(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	dashboard, err := c.teacher.Dashboard(r.Context(), identity.ID, r.URL.Query().Get("course_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dashboard.Course.TeacherName = identity.Name
	courses, err := c.ownedTeacherCourses(r.Context(), identity.ID)
	if err != nil {
		http.Error(w, "Could not load teacher courses", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "faculty/dashboard.html", map[string]any{
		"Role":         "teacher",
		"ActiveNav":    "overview",
		"Dashboard":    dashboard,
		"Courses":      courses,
		"StudentNames": c.studentNames(r.Context(), dashboard),
		"Notice":       r.URL.Query().Get("notice"),
	})
}

func (c LearningController) TeacherContent(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	courses, err := c.ownedTeacherCourses(r.Context(), identity.ID)
	if err != nil {
		http.Error(w, "Could not load teacher courses", http.StatusInternalServerError)
		return
	}
	selectedCourseID := r.URL.Query().Get("course_id")
	if selectedCourseID == "" && len(courses) == 1 {
		selectedCourseID = courses[0].ID
	}
	selectedCourse := selectedCourseID != ""
	dashboard := domain.TeacherLearningDashboard{}
	if selectedCourse {
		dashboard, err = c.teacher.Dashboard(r.Context(), identity.ID, selectedCourseID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dashboard.Course.TeacherName = identity.Name
	}
	materialRows := make([]teacherMaterialView, 0, len(dashboard.Materials))
	for _, material := range dashboard.Materials {
		target := "Course-wide source"
		if material.ConceptID != "" {
			target = "Lesson target unavailable"
			for _, concept := range dashboard.Concepts {
				if concept.Concept.ID == material.ConceptID {
					target = concept.Concept.Title
					break
				}
			}
		}
		materialRows = append(materialRows, teacherMaterialView{Material: material, Target: target})
	}
	w.Header().Set("Cache-Control", "no-store")
	c.renderer.Render(w, "faculty/content.html", map[string]any{
		"Role":           "teacher",
		"ActiveNav":      "content",
		"Identity":       identity,
		"Dashboard":      dashboard,
		"Courses":        courses,
		"SelectedCourse": selectedCourse,
		"MaterialRows":   materialRows,
		"StudentNames":   c.studentNames(r.Context(), dashboard),
		"Notice":         r.URL.Query().Get("notice"),
		"Error":          r.URL.Query().Get("error"),
	})
}

func (c LearningController) ownedTeacherCourses(ctx context.Context, teacherID string) ([]domain.LearningCourse, error) {
	if c.courses == nil {
		return nil, nil
	}
	courses, err := c.courses.ListCourses(ctx)
	if err != nil {
		return nil, err
	}
	owned := make([]domain.LearningCourse, 0, len(courses))
	for _, course := range courses {
		if course.TeacherID == teacherID {
			owned = append(owned, course)
		}
	}
	return owned, nil
}

func (c LearningController) studentNames(ctx context.Context, dashboard domain.TeacherLearningDashboard) map[string]string {
	names := make(map[string]string)
	for _, card := range dashboard.SupportNeeded {
		if card.StudentName != "" {
			names[card.StudentID] = card.StudentName
		}
	}
	if c.accounts == nil {
		return names
	}
	ids := make(map[string]struct{})
	for _, item := range dashboard.RecentEvidence {
		ids[item.StudentID] = struct{}{}
	}
	for _, item := range dashboard.SupportNeeded {
		ids[item.StudentID] = struct{}{}
	}
	for id := range ids {
		account, found, err := c.accounts.FindAccountByID(ctx, id)
		if err == nil && found && account.Name != "" {
			names[id] = account.Name
		}
	}
	return names
}

func (c LearningController) PublishMaterial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if !c.requireOwnedCourse(w, r, r.FormValue("course_id")) {
		return
	}
	_, err := c.teacher.PublishMaterial(r.Context(), usecase.PublishMaterialRequest{
		CourseID:   r.FormValue("course_id"),
		ConceptID:  r.FormValue("concept_id"),
		Title:      r.FormValue("title"),
		Body:       r.FormValue("body"),
		SourceKind: domain.LearningSourceKind(r.FormValue("source_kind")),
		SourceRef:  r.FormValue("source_ref"),
	})
	if err != nil {
		redirect := "/teacher/content?course_id=" + url.QueryEscape(r.FormValue("course_id")) + "&error=" + url.QueryEscape(err.Error())
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/teacher/content?course_id="+url.QueryEscape(r.FormValue("course_id"))+"&notice=Material%20published", http.StatusSeeOther)
}

func (c LearningController) UpdateTutorPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if !c.requireOwnedCourse(w, r, r.FormValue("course_id")) {
		return
	}
	modes := make([]domain.TutorMode, 0)
	for _, value := range r.Form["modes"] {
		if strings.TrimSpace(value) != "" {
			modes = append(modes, domain.TutorMode(value))
		}
	}
	_, err := c.teacher.UpdateTutorPolicy(r.Context(), usecase.UpdateTutorPolicyRequest{
		CourseID:             r.FormValue("course_id"),
		DefaultMode:          domain.TutorMode(r.FormValue("default_mode")),
		AllowedModes:         modes,
		AllowExternalSources: r.FormValue("allow_external_sources") == "on" || r.FormValue("allow_external_sources") == "true",
		SourcePolicy:         r.FormValue("source_policy"),
	})
	if err != nil {
		redirect := "/teacher/content?course_id=" + url.QueryEscape(r.FormValue("course_id")) + "&error=" + url.QueryEscape(err.Error())
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/teacher/content?course_id="+url.QueryEscape(r.FormValue("course_id"))+"&notice=Tutor%20policy%20updated", http.StatusSeeOther)
}

func (c LearningController) TeacherDashboardJSON(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	dashboard, err := c.teacher.Dashboard(r.Context(), identity.ID, r.URL.Query().Get("course_id"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	dashboard.Course.TeacherName = identity.Name
	courses, err := c.courses.ListCourses(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, errors.New("could not load teacher courses"))
		return
	}
	ownedCourses := make([]domain.LearningCourse, 0, len(courses))
	for _, course := range courses {
		if course.TeacherID == identity.ID {
			ownedCourses = append(ownedCourses, course)
		}
	}
	writeJSON(w, http.StatusOK, toAPITeacherDashboard(dashboard, ownedCourses))
}

func (c LearningController) PublishMaterialJSON(w http.ResponseWriter, r *http.Request) {
	var req usecase.PublishMaterialRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !c.requireOwnedCourse(w, r, req.CourseID) {
		return
	}
	material, err := c.teacher.PublishMaterial(r.Context(), req)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPIMaterial(material))
}

func (c LearningController) UploadMaterialPDF(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(service.MaxPDFMaterialBytes + 1<<20); err != nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("invalid multipart upload"))
		return
	}
	courseID := strings.TrimSpace(r.FormValue("course_id"))
	if !c.requireOwnedCourse(w, r, courseID) {
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("file is required"))
		return
	}
	defer file.Close()
	if header.Size > service.MaxPDFMaterialBytes {
		writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("pdf exceeds the 10 MB limit"))
		return
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".pdf" {
		writeAPIError(w, http.StatusBadRequest, errors.New("file must have a .pdf extension"))
		return
	}
	conceptID := strings.TrimSpace(r.FormValue("concept_id"))
	if conceptID == "" {
		writeAPIError(w, http.StatusBadRequest, errors.New("concept_id is required"))
		return
	}
	body, err := service.ExtractPDFText(file, header.Size)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = service.SafePDFTitle(header.Filename)
	}
	material, err := c.teacher.PublishMaterial(r.Context(), usecase.PublishMaterialRequest{
		CourseID: courseID, ConceptID: conceptID, Title: title, Body: body,
		SourceKind: domain.LearningSourceTeacher, SourceRef: header.Filename,
	})
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPIMaterial(material))
}

func (c LearningController) UpdateTutorPolicyJSON(w http.ResponseWriter, r *http.Request) {
	var req usecase.UpdateTutorPolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if courseID := r.PathValue("course_id"); courseID != "" {
		req.CourseID = courseID
	}
	if !c.requireOwnedCourse(w, r, req.CourseID) {
		return
	}
	policy, err := c.teacher.UpdateTutorPolicy(r.Context(), req)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIPolicy(policy))
}

func studentIDFromRequest(r *http.Request) string {
	if identity, ok := auth.IdentityFromContext(r.Context()); ok && identity.Role == "student" {
		return identity.ID
	}
	studentID := r.URL.Query().Get("student_id")
	if studentID == "" {
		studentID = r.FormValue("student_id")
	}
	if studentID == "" {
		studentID = r.Header.Get("X-Student-ID")
	}
	if studentID == "" {
		return "S-001"
	}
	return studentID
}

func findConcept(cards []domain.ConceptProgressCard, conceptID string) domain.LearningConcept {
	for _, card := range cards {
		if card.Concept.ID == conceptID {
			return card.Concept
		}
	}
	if len(cards) == 0 {
		return domain.LearningConcept{}
	}
	return domain.LearningConcept{}
}

func materialsForConcept(materials []domain.LearningMaterial, conceptID string) []domain.LearningMaterial {
	result := make([]domain.LearningMaterial, 0, len(materials))
	for _, material := range materials {
		if material.ConceptID == conceptID {
			result = append(result, material)
		}
	}
	return result
}

const maxJSONBodyBytes int64 = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	contentType := r.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		writeAPIError(w, http.StatusUnsupportedMediaType, errors.New("Content-Type must be application/json"))
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		writeAPIError(w, http.StatusUnsupportedMediaType, fmt.Errorf("invalid Content-Type: %w", err))
		return false
	}
	if mediaType != "application/json" {
		writeAPIError(w, http.StatusUnsupportedMediaType, errors.New("Content-Type must be application/json"))
		return false
	}
	if r.Body == nil {
		writeAPIError(w, http.StatusBadRequest, errors.New("request body must not be empty"))
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeAPIError(w, http.StatusRequestEntityTooLarge, errors.New("request body exceeds the 1 MiB limit"))
			return false
		}
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("could not read request body: %w", err))
		return false
	}

	trimmedBody := bytes.TrimSpace(body)
	if len(trimmedBody) == 0 {
		writeAPIError(w, http.StatusBadRequest, errors.New("request body must not be empty"))
		return false
	}
	if bytes.Equal(trimmedBody, []byte("null")) {
		writeAPIError(w, http.StatusBadRequest, errors.New("request body must contain a JSON object"))
		return false
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return false
	}

	var trailingValue json.RawMessage
	if err := decoder.Decode(&trailingValue); err != io.EOF {
		if err == nil {
			writeAPIError(w, http.StatusBadRequest, errors.New("request body must contain a single JSON value"))
		} else {
			writeAPIError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON after first value: %w", err))
		}
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func templateEscape(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;", "'", "&#39;").Replace(value)
}
