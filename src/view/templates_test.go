package view

import (
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

func productTemplateRenderer(t *testing.T) *Renderer {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate view package")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "../.."))
	renderer, err := NewRenderer(filepath.Join(root, "src/templates/**/*.html"))
	if err != nil {
		t.Fatalf("parse product templates: %v", err)
	}
	return renderer
}

func testLearningDashboard() domain.StudentLearningDashboard {
	now := time.Now().UTC()
	concept := domain.LearningConcept{ID: "concept-1", CourseID: "course-1", Title: "Linear equations", Summary: "Solve and explain.", Order: 1, Level: "Foundation"}
	material := domain.LearningMaterial{ID: "material-1", CourseID: "course-1", ConceptID: concept.ID, Title: "Teacher note", Body: "Balance both sides.", Published: true, Version: 1, Source: domain.LearningSourceRef{ID: "material-1", Title: "Teacher note", Kind: domain.LearningSourceTeacher}}
	progress := domain.ConceptProgress{StudentID: "S-001", CourseID: "course-1", ConceptID: concept.ID, Status: domain.ConceptProgressLearning, Confidence: 68, EvidenceCount: 2, NextAction: "Explain the check"}
	course := domain.LearningCourse{ID: "course-1", Title: "Algebra", Subject: "Mathematics", Description: "A path.", TeacherName: "Maya Chen", TutorPolicy: domain.TutorPolicy{DefaultMode: domain.TutorModeGuided, AllowedModes: []domain.TutorMode{domain.TutorModeGuided}, SourcePolicy: "Teacher material leads."}}
	session := domain.LearningSession{ID: "session-1", StudentID: "S-001", CourseID: course.ID, ConceptID: concept.ID, Mode: domain.TutorModeGuided, Status: domain.LearningSessionActive, StartedAt: now, LastActiveAt: now}
	turn := domain.LearningTurn{ID: "turn-1", SessionID: session.ID, Role: domain.LearningTurnTutor, Kind: "explanation", Mode: domain.TutorModeGuided, Body: "An equation is a relationship.", CreatedAt: now}
	return domain.StudentLearningDashboard{StudentID: "S-001", StudentName: "Alex Morgan", Course: course, Concepts: []domain.ConceptProgressCard{{Concept: concept, Progress: progress, State: progress.Status.Label()}}, FeaturedConcept: concept, FeaturedProgress: progress, ActiveSession: &session, Turns: []domain.LearningTurn{turn}, Materials: []domain.LearningMaterial{material}, Objective: domain.LearningObjective{Description: "Understand balance."}, Policy: course.TutorPolicy, Stats: domain.StudentLearningStats{ConceptsStarted: 1, ConceptsSecure: 0, LearningMinutes: 90, StreakDays: 3}}
}

func testTeacherDashboard() domain.TeacherLearningDashboard {
	student := testLearningDashboard()
	concept := student.Concepts[0].Concept
	progress := student.Concepts[0].Progress
	progress.StudentID = "class"
	return domain.TeacherLearningDashboard{Course: student.Course, Concepts: []domain.ConceptProgressCard{{Concept: concept, Progress: progress, State: progress.Status.Label()}}, Materials: student.Materials, Policy: student.Policy, Stats: domain.TeacherLearningStats{ActiveLearners: 1, ConceptsPublished: 1, AverageConfidence: 68, NeedsSupport: 0}, SupportNeeded: []domain.LearnerSupportCard{}, RecentEvidence: []domain.LearningEvidence{}}
}

func TestProductTemplatesRender(t *testing.T) {
	renderer := productTemplateRenderer(t)
	student := testLearningDashboard()
	teacher := testTeacherDashboard()
	workspace := map[string]any{
		"Dashboard":    student,
		"Session":      student.ActiveSession,
		"Turns":        student.Turns,
		"Concept":      student.FeaturedConcept,
		"Materials":    student.Materials,
		"SelectedMode": domain.TutorModeGuided,
	}
	cases := []struct {
		name string
		data any
	}{
		{"app/landing.html", map[string]any{"Role": "guest"}},
		{"student/dashboard.html", map[string]any{"Role": "student", "StudentID": student.StudentID, "Dashboard": student}},
		{"student/workspace.html", map[string]any{"Role": "student", "StudentID": student.StudentID, "Workspace": workspace}},
		{"student/workspace_conversation.html", map[string]any{"Role": "student", "StudentID": student.StudentID, "Workspace": workspace}},
		{"faculty/dashboard.html", map[string]any{"Role": "teacher", "Dashboard": teacher}},
		{"faculty/content.html", map[string]any{"Role": "teacher", "Dashboard": teacher}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			renderer.Render(recorder, tc.name, tc.data)
			if recorder.Code != 200 {
				t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
			}
			if recorder.Body.Len() == 0 {
				t.Fatal("expected rendered content")
			}
		})
	}
}
