package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/memory"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/service"
)

func learningFixture() (StudentLearningPortal, TeacherLearningPortal) {
	now := time.Now().UTC()
	course := domain.LearningCourse{
		ID:          "course-1",
		Title:       "Test course",
		Subject:     "Test",
		TeacherID:   "teacher-1",
		TeacherName: "Test Teacher",
		TutorPolicy: domain.TutorPolicy{
			DefaultMode:          domain.TutorModeGuided,
			AllowedModes:         []domain.TutorMode{domain.TutorModeGuided, domain.TutorModeSocratic},
			AllowExternalSources: true,
		},
	}
	concept := domain.LearningConcept{ID: "concept-1", CourseID: course.ID, Title: "A concept", Order: 1}
	objective := domain.LearningObjective{ID: "objective-1", CourseID: course.ID, ConceptID: concept.ID, Description: "Understand the concept."}
	material := domain.LearningMaterial{
		ID:        "material-1",
		CourseID:  course.ID,
		ConceptID: concept.ID,
		Title:     "Teacher note",
		Body:      "Use the relationship as your anchor.",
		Published: true,
		Source:    domain.LearningSourceRef{ID: "material-1", Title: "Teacher note", Kind: domain.LearningSourceTeacher},
	}
	content := memory.NewLearningContentRepository([]domain.LearningCourse{course}, []domain.LearningConcept{concept}, []domain.LearningObjective{objective}, []domain.LearningMaterial{material})
	sessions := memory.NewLearningSessionRepository(nil, nil)
	progress := memory.NewConceptProgressRepository([]domain.ConceptProgress{{
		StudentID: "S-001", CourseID: course.ID, ConceptID: concept.ID,
		Status: domain.ConceptProgressLearning, Confidence: 40, EvidenceCount: 1, LastActivityAt: now,
	}})
	evidence := memory.NewLearningEvidenceRepository(nil)
	checks := memory.NewUnderstandingCheckRepository(nil)
	agent := service.NewTutorAgent(memory.NewStubLearningAgentGateway())
	checker := service.NewUnderstandingCheckService(memory.NewStubFormativeCheckGateway())
	return NewStudentLearningPortal(content, sessions, progress, evidence, checks, agent, checker), NewTeacherLearningPortal(content, progress, evidence)
}

func TestStudentDashboardBuildsLearningReadModel(t *testing.T) {
	student, _ := learningFixture()
	dashboard, err := student.Dashboard(context.Background(), "S-001")
	if err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.Course.Title != "Test course" {
		t.Fatalf("unexpected course: %q", dashboard.Course.Title)
	}
	if len(dashboard.Concepts) != 1 || dashboard.FeaturedConcept.ID != "concept-1" {
		t.Fatalf("unexpected concept cards: %#v", dashboard.Concepts)
	}
	if len(dashboard.Materials) != 1 || dashboard.Materials[0].Source.Kind != domain.LearningSourceTeacher {
		t.Fatalf("unexpected materials: %#v", dashboard.Materials)
	}
}

func TestContinuePersistsConversationAndProgress(t *testing.T) {
	student, _ := learningFixture()
	ctx := context.Background()
	session, err := student.StartOrResume(ctx, StartLearningRequest{StudentID: "S-001", Mode: domain.TutorModeSocratic})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	result, err := student.Continue(ctx, SendLearningTurnRequest{
		StudentID: "S-001",
		SessionID: session.ID,
		Mode:      domain.TutorModeSocratic,
		Message:   "Can you ask me a question?",
	})
	if err != nil {
		t.Fatalf("continue: %v", err)
	}
	if result.Turn.Role != domain.LearningTurnTutor || result.Turn.Kind != "question" {
		t.Fatalf("unexpected tutor turn: %#v", result.Turn)
	}
	if result.Progress.EvidenceCount != 2 {
		t.Fatalf("expected evidence count 2, got %d", result.Progress.EvidenceCount)
	}
	if result.Progress.Confidence != 43 {
		t.Fatalf("expected confidence 43, got %d", result.Progress.Confidence)
	}
	resumed, err := student.StartOrResume(ctx, StartLearningRequest{StudentID: "S-001", Mode: domain.TutorModeGuided})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.ID != session.ID {
		t.Fatalf("expected session %q, got %q", session.ID, resumed.ID)
	}
	_, turns, err := student.Session(ctx, session.ID)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected two persisted turns, got %d", len(turns))
	}
	check, err := student.CheckUnderstanding(ctx, session.ID, "S-001")
	if err != nil {
		t.Fatalf("understanding check: %v", err)
	}
	if check.Summary == "" || len(check.Feedback) == 0 {
		t.Fatalf("expected formative feedback, got %#v", check)
	}
	secondCheck, err := student.CheckUnderstanding(ctx, session.ID, "S-001")
	if err != nil {
		t.Fatalf("repeat understanding check: %v", err)
	}
	if secondCheck.ID != check.ID {
		t.Fatalf("expected repeated check to be idempotent, got %q and %q", check.ID, secondCheck.ID)
	}
	latest, err := student.LatestCheck(ctx, session.ID)
	if err != nil || latest == nil || latest.ID != check.ID {
		t.Fatalf("expected latest check to be persisted, latest=%#v err=%v", latest, err)
	}
}

func TestTutorPolicyRejectsDisabledMode(t *testing.T) {
	student, _ := learningFixture()
	_, err := student.StartOrResume(context.Background(), StartLearningRequest{
		StudentID: "S-001",
		Mode:      domain.TutorModeDirect,
	})
	if err == nil {
		t.Fatal("expected disallowed tutor mode to fail")
	}
}

func TestExternalMaterialRequiresHTTPSourceReference(t *testing.T) {
	_, teacher := learningFixture()
	_, err := teacher.PublishMaterial(context.Background(), PublishMaterialRequest{
		CourseID: "course-1", ConceptID: "concept-1", Title: "External", Body: "Read this.",
		SourceKind: domain.LearningSourceExternal, SourceRef: "javascript:alert(1)",
	})
	if err == nil {
		t.Fatal("expected invalid external source reference to fail")
	}
}

func TestTeacherCanPublishMaterialAndUpdatePolicy(t *testing.T) {
	_, teacher := learningFixture()
	ctx := context.Background()
	material, err := teacher.PublishMaterial(ctx, PublishMaterialRequest{
		CourseID: "course-1", ConceptID: "concept-1", Title: "New note", Body: "Try a visual model.",
	})
	if err != nil {
		t.Fatalf("publish material: %v", err)
	}
	if !material.Published || material.Source.Kind != domain.LearningSourceTeacher {
		t.Fatalf("unexpected material: %#v", material)
	}
	policy, err := teacher.UpdateTutorPolicy(ctx, UpdateTutorPolicyRequest{
		CourseID: "course-1", DefaultMode: domain.TutorModeDirect,
		AllowedModes: []domain.TutorMode{domain.TutorModeDirect}, SourcePolicy: "Teacher only.",
	})
	if err != nil {
		t.Fatalf("update policy: %v", err)
	}
	if policy.DefaultMode != domain.TutorModeDirect || len(policy.AllowedModes) != 1 {
		t.Fatalf("unexpected policy: %#v", policy)
	}
}
