package view

import (
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
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
		"Dashboard": student,
		"Session":   student.ActiveSession,
		"Turns": []domain.LearningTurn{
			{ID: "student-turn", Role: domain.LearningTurnStudent, Mode: domain.TutorModeGuided, Body: "I think I can solve it."},
			student.Turns[0],
		},
		"Concept":      student.FeaturedConcept,
		"Materials":    student.Materials,
		"SelectedMode": domain.TutorModeGuided,
	}
	cases := []struct {
		name string
		data any
	}{
		{"app/landing.html", map[string]any{"Role": "guest"}},
		{"student/dashboard.html", map[string]any{"Role": "student", "StudentID": student.StudentID, "Dashboard": student, "EnrolledCourses": []domain.CourseEnrollment{}, "ApprovedCourses": []domain.CourseEnrollment{}}},
		{"student/workspace.html", map[string]any{"Role": "student", "StudentID": student.StudentID, "Workspace": workspace}},
		{"student/workspace_conversation.html", map[string]any{"Role": "student", "StudentID": student.StudentID, "Workspace": workspace}},
		{"faculty/dashboard.html", map[string]any{"Role": "teacher", "ActiveNav": "overview", "Dashboard": teacher, "Courses": []domain.LearningCourse{teacher.Course}}},
		{"faculty/content.html", map[string]any{"Role": "teacher", "ActiveNav": "content", "Identity": map[string]string{"Name": "Maya Chen"}, "Dashboard": teacher, "Courses": []domain.LearningCourse{teacher.Course}, "SelectedCourse": true, "MaterialRows": []any{}}},
		{"faculty/enrollments.html", map[string]any{"ActiveNav": "enrollments", "Identity": map[string]string{"Name": "Maya Chen"}}},
		{"faculty/courses.html", map[string]any{"ActiveNav": "courses", "Identity": map[string]string{"Name": "Maya Chen"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			renderer.Render(recorder, tc.name, tc.data)
			if recorder.Code != 200 {
				t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
			if recorder.Body.Len() == 0 {
				t.Fatal("expected rendered content")
			}
			if tc.name == "student/workspace_conversation.html" && (!strings.Contains(recorder.Body.String(), `class="message student"`) || !strings.Contains(recorder.Body.String(), `class="message tutor"`)) {
				t.Fatal("conversation should distinguish student and tutor message sides")
			}
			if tc.name == "student/workspace.html" && (!strings.Contains(recorder.Body.String(), `role="tab"`) || !strings.Contains(recorder.Body.String(), `id="context-panel"`) || !strings.Contains(recorder.Body.String(), `aria-selected="false"`)) {
				t.Fatal("learning space should render accessible conversation and sources/goals tabs")
			}
			if tc.name == "faculty/enrollments.html" && (!strings.Contains(recorder.Body.String(), `href="/teacher/courses"`) || !strings.Contains(recorder.Body.String(), "Enrollment studio")) {
				t.Fatal("enrollment studio should retain the shared teacher navigation")
			}
		})
	}
}
