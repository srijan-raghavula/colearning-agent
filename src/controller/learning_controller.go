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
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/usecase"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type LearningController struct {
	student  usecase.StudentLearningPortal
	teacher  usecase.TeacherLearningPortal
	renderer *view.Renderer
	accounts accountLookup
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

type learningWorkspaceView struct {
	Dashboard    domain.StudentLearningDashboard
	Session      domain.LearningSession
	Turns        []domain.LearningTurn
	Concept      domain.LearningConcept
	Materials    []domain.LearningMaterial
	SelectedMode domain.TutorMode
	Check        *domain.UnderstandingCheck
	Notice       string
	Error        string
}

func (c LearningController) LandingPage(w http.ResponseWriter, r *http.Request) {
	c.renderer.Render(w, "app/landing.html", map[string]any{
		"Role": "guest",
	})
}

func (c LearningController) StudentPage(w http.ResponseWriter, r *http.Request) {
	dashboard, err := c.student.Dashboard(r.Context(), studentIDFromRequest(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.StudentName = identity.Name
	}
	c.renderer.Render(w, "student/dashboard.html", map[string]any{
		"Role":      "student",
		"StudentID": dashboard.StudentID,
		"Dashboard": dashboard,
		"Notice":    r.URL.Query().Get("notice"),
	})
}

func (c LearningController) StudentWorkspace(w http.ResponseWriter, r *http.Request) {
	studentID := studentIDFromRequest(r)
	dashboard, err := c.student.Dashboard(r.Context(), studentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.StudentName = identity.Name
	}

	sessionID := r.URL.Query().Get("session_id")
	conceptID := r.URL.Query().Get("concept_id")
	if sessionID == "" {
		session, startErr := c.student.StartOrResume(r.Context(), usecase.StartLearningRequest{
			StudentID: studentID,
			CourseID:  r.URL.Query().Get("course_id"),
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
	if session.StudentID != studentID {
		http.Error(w, "learning session does not belong to this student", http.StatusForbidden)
		return
	}
	concept := findConcept(dashboard.Concepts, session.ConceptID)
	selectedMode := session.Mode
	if requestedMode := domain.TutorMode(r.URL.Query().Get("mode")); requestedMode != "" {
		selectedMode = requestedMode
	}
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
			Dashboard:    dashboard,
			Session:      session,
			Turns:        turns,
			Concept:      concept,
			Materials:    materials,
			SelectedMode: selectedMode,
			Check:        check,
			Notice:       r.URL.Query().Get("notice"),
			Error:        r.URL.Query().Get("error"),
		},
	})
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
	session, err := c.student.StartOrResume(r.Context(), usecase.StartLearningRequest{
		StudentID: studentID,
		CourseID:  r.FormValue("course_id"),
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
	mode := domain.TutorMode(r.FormValue("mode"))
	message := strings.TrimSpace(r.FormValue("message"))
	result, err := c.student.Continue(r.Context(), usecase.SendLearningTurnRequest{
		StudentID: studentID,
		SessionID: sessionID,
		Mode:      mode,
		Message:   message,
	})
	if err != nil {
		if r.Header.Get("HX-Request") != "" {
			c.renderTurnError(w, r, err.Error())
			return
		}
		http.Redirect(w, r, workspaceLocation(sessionID, "", err.Error()), http.StatusSeeOther)
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
	_, err := c.student.CheckUnderstanding(r.Context(), r.FormValue("session_id"), studentID)
	if err != nil {
		if r.Header.Get("HX-Request") != "" {
			c.renderTurnError(w, r, err.Error())
			return
		}
		http.Redirect(w, r, workspaceLocation(r.FormValue("session_id"), "", err.Error()), http.StatusSeeOther)
		return
	}
	workspace, err := c.workspaceView(r.Context(), studentID, r.FormValue("session_id"), domain.TutorMode(r.FormValue("mode")))
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
	http.Redirect(w, r, workspaceLocation(r.FormValue("session_id"), "Understanding check complete.", ""), http.StatusSeeOther)
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
	dashboard, err := c.student.Dashboard(ctx, studentID)
	if err != nil {
		return learningWorkspaceView{}, err
	}
	if identity, ok := auth.IdentityFromContext(ctx); ok {
		dashboard.StudentName = identity.Name
	}
	session, turns, err := c.student.Session(ctx, sessionID)
	if err != nil {
		return learningWorkspaceView{}, err
	}
	concept := findConcept(dashboard.Concepts, session.ConceptID)
	materials := materialsForConcept(dashboard.Materials, concept.ID)
	if len(materials) == 0 {
		materials = dashboard.Materials
	}
	if mode == "" {
		mode = session.Mode
	}
	check, err := c.student.LatestCheck(ctx, session.ID)
	if err != nil {
		return learningWorkspaceView{}, err
	}
	return learningWorkspaceView{
		Dashboard:    dashboard,
		Session:      session,
		Turns:        turns,
		Concept:      concept,
		Materials:    materials,
		SelectedMode: mode,
		Check:        check,
	}, nil
}

func (c LearningController) StudentDashboardJSON(w http.ResponseWriter, r *http.Request) {
	dashboard, err := c.student.Dashboard(r.Context(), studentIDFromRequest(r))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.StudentName = identity.Name
	}
	writeJSON(w, http.StatusOK, toAPIStudentDashboard(dashboard))
}

func (c LearningController) StartStudentSessionJSON(w http.ResponseWriter, r *http.Request) {
	var req usecase.StartLearningRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// Derive ownership from the authenticated identity, never the request body.
	req.StudentID = studentIDFromRequest(r)
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
	result, err := c.student.Continue(r.Context(), req)
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
	check, err := c.student.CheckUnderstanding(r.Context(), sessionID, studentID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAPICheck(check))
}

func (c LearningController) TeacherPage(w http.ResponseWriter, r *http.Request) {
	dashboard, err := c.teacher.Dashboard(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.Course.TeacherName = identity.Name
	}
	c.renderer.Render(w, "faculty/dashboard.html", map[string]any{
		"Role":         "teacher",
		"Dashboard":    dashboard,
		"StudentNames": c.studentNames(r.Context(), dashboard),
		"Notice":       r.URL.Query().Get("notice"),
	})
}

func (c LearningController) TeacherContent(w http.ResponseWriter, r *http.Request) {
	dashboard, err := c.teacher.Dashboard(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.Course.TeacherName = identity.Name
	}
	c.renderer.Render(w, "faculty/content.html", map[string]any{
		"Role":         "teacher",
		"Dashboard":    dashboard,
		"StudentNames": c.studentNames(r.Context(), dashboard),
		"Notice":       r.URL.Query().Get("notice"),
		"Error":        r.URL.Query().Get("error"),
	})
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
	_, err := c.teacher.PublishMaterial(r.Context(), usecase.PublishMaterialRequest{
		CourseID:   r.FormValue("course_id"),
		ConceptID:  r.FormValue("concept_id"),
		Title:      r.FormValue("title"),
		Body:       r.FormValue("body"),
		SourceKind: domain.LearningSourceKind(r.FormValue("source_kind")),
		SourceRef:  r.FormValue("source_ref"),
	})
	if err != nil {
		redirect := "/teacher/content?error=" + url.QueryEscape(err.Error())
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/teacher/content?notice=Material%20published", http.StatusSeeOther)
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
		redirect := "/teacher/content?error=" + url.QueryEscape(err.Error())
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/teacher/content?notice=Tutor%20policy%20updated", http.StatusSeeOther)
}

func (c LearningController) TeacherDashboardJSON(w http.ResponseWriter, r *http.Request) {
	dashboard, err := c.teacher.Dashboard(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	if identity, ok := auth.IdentityFromContext(r.Context()); ok {
		dashboard.Course.TeacherName = identity.Name
	}
	writeJSON(w, http.StatusOK, toAPITeacherDashboard(dashboard))
}

func (c LearningController) PublishMaterialJSON(w http.ResponseWriter, r *http.Request) {
	var req usecase.PublishMaterialRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	material, err := c.teacher.PublishMaterial(r.Context(), req)
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
	return cards[0].Concept
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
