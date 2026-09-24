package bootstrap

import (
	"log"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/memory"
	"github.com/srijan-raghavula/colearning-agent/src/config"
	"github.com/srijan-raghavula/colearning-agent/src/controller"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/routes"
	"github.com/srijan-raghavula/colearning-agent/src/service"
	"github.com/srijan-raghavula/colearning-agent/src/usecase"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type App struct {
	Config config.HTTP
	Routes routes.Dependencies
}

func NewApp() App {
	httpConfig := config.LoadHTTP()

	now := time.Now().UTC()
	submissions := []domain.Submission{
		{ID: "SUB-001", StudentID: "S-001", SubjectID: "math", TopicID: "algebra", Title: "Algebra Worksheet", Attachment: "algebra.pdf", DueAt: now.Add(-48 * time.Hour), SubmittedAt: now.Add(-50 * time.Hour), Status: domain.SubmissionStatusLate},
		{ID: "SUB-002", StudentID: "S-001", SubjectID: "science", TopicID: "forces", Title: "Forces Lab", Attachment: "forces.docx", DueAt: now.Add(-24 * time.Hour), SubmittedAt: now.Add(-25 * time.Hour), Status: domain.SubmissionStatusLate},
		{ID: "SUB-003", StudentID: "S-002", SubjectID: "math", TopicID: "geometry", Title: "Geometry Quiz", Attachment: "", DueAt: now.Add(-12 * time.Hour), SubmittedAt: time.Time{}, Status: domain.SubmissionStatusMissing},
		{ID: "SUB-004", StudentID: "S-002", SubjectID: "science", TopicID: "energy", Title: "Energy Report", Attachment: "energy.pdf", DueAt: now.Add(24 * time.Hour), SubmittedAt: now.Add(-1 * time.Hour), Status: domain.SubmissionStatusSubmitted},
	}

	// The staged evaluator remains implemented in service/evaluation.go for the
	// legacy assessment compatibility path. It is intentionally not wired into
	// the primary co-learning bootstrap; formative checks use their own
	// provider-neutral UnderstandingCheck contract below.
	// Seed evaluations preserve the legacy assessment demo endpoints.
	evaluationEngine := service.NewDefaultEvaluationEngine()
	evaluations := []domain.Evaluation{
		evaluationEngine.Evaluate(submissions[0], 18, 20, []domain.TopicScore{{Topic: "algebra", Score: 8, MaxPossible: 10}, {Topic: "functions", Score: 10, MaxPossible: 10}}),
		evaluationEngine.Evaluate(submissions[1], 12, 20, []domain.TopicScore{{Topic: "forces", Score: 6, MaxPossible: 10}, {Topic: "vectors", Score: 6, MaxPossible: 10}}),
		evaluationEngine.Evaluate(submissions[3], 16, 20, []domain.TopicScore{{Topic: "energy", Score: 8, MaxPossible: 10}, {Topic: "conservation", Score: 8, MaxPossible: 10}}),
	}

	submissionRepo := memory.NewSubmissionRepository(submissions)
	evaluationRepo := memory.NewEvaluationRepository(evaluations)

	// ── Co-learning vertical slice ───────────────────────────────────────────
	course := domain.LearningCourse{
		ID:           "course-algebra-foundations",
		Title:        "Algebra foundations",
		Subject:      "Mathematics",
		Description:  "A calm, guided path from relationships to equations to functions.",
		TeacherID:    "T-001",
		TeacherName:  "Maya Chen",
		StudentCount: 24,
		ConceptCount: 4,
		TutorPolicy: domain.TutorPolicy{
			DefaultMode:          domain.TutorModeGuided,
			AllowedModes:         []domain.TutorMode{domain.TutorModeGuided, domain.TutorModeSocratic, domain.TutorModeDirect, domain.TutorModePractice, domain.TutorModeHint, domain.TutorModeDiagnostic},
			AllowExternalSources: true,
			SourcePolicy:         "Teacher material leads; external sources are labeled for comparison.",
		},
		NextConceptID: "linear-equations",
	}
	concepts := []domain.LearningConcept{
		{ID: "linear-equations", CourseID: course.ID, Title: "Linear equations", Summary: "Find the unknown that makes a relationship true.", Level: "Foundation", Order: 1, EstimatedMinutes: 18},
		{ID: "graphing-equations", CourseID: course.ID, Title: "Graphing equations", Summary: "See equations as lines and read their relationships visually.", Level: "Foundation", Order: 2, EstimatedMinutes: 22, PrerequisiteIDs: []string{"linear-equations"}},
		{ID: "systems-equations", CourseID: course.ID, Title: "Systems of equations", Summary: "Use two relationships together to find a shared solution.", Level: "Practice", Order: 3, EstimatedMinutes: 25, PrerequisiteIDs: []string{"linear-equations"}},
		{ID: "quadratics", CourseID: course.ID, Title: "Quadratic thinking", Summary: "Recognize curved relationships and reason about their roots.", Level: "Stretch", Order: 4, EstimatedMinutes: 28, PrerequisiteIDs: []string{"graphing-equations"}},
	}
	objectives := []domain.LearningObjective{
		{ID: "objective-linear", CourseID: course.ID, ConceptID: "linear-equations", Description: "Solve a linear equation and explain why the solution works.", Outcomes: []string{"Isolate the unknown", "Check the solution", "Explain the relationship"}},
		{ID: "objective-graphing", CourseID: course.ID, ConceptID: "graphing-equations", Description: "Connect an equation to a line and read its key relationships.", Outcomes: []string{"Plot points", "Interpret slope", "Use a graph to explain a relationship"}},
		{ID: "objective-systems", CourseID: course.ID, ConceptID: "systems-equations", Description: "Find and interpret the solution shared by two equations.", Outcomes: []string{"Choose a method", "Solve a system", "Interpret the pair"}},
		{ID: "objective-quadratics", CourseID: course.ID, ConceptID: "quadratics", Description: "Describe the shape and roots of a quadratic relationship.", Outcomes: []string{"Recognize curvature", "Interpret roots", "Compare representations"}},
	}
	materials := []domain.LearningMaterial{
		{ID: "material-linear-teacher", CourseID: course.ID, ConceptID: "linear-equations", Title: "Teacher note: isolate, then verify", Body: "An equation is a relationship waiting to be balanced. Isolate the unknown without breaking the relationship, then substitute the result back into the original equation.", Source: domain.LearningSourceRef{ID: "material-linear-teacher", Title: "Teacher note: isolate, then verify", Kind: domain.LearningSourceTeacher}, Published: true, Version: 2},
		{ID: "material-linear-external", CourseID: course.ID, ConceptID: "linear-equations", Title: "OpenStax: solving linear equations", Body: "Use inverse operations to preserve balance while isolating a variable. Always verify the result in the original equation.", Source: domain.LearningSourceRef{ID: "material-linear-external", Title: "OpenStax: solving linear equations", Kind: domain.LearningSourceExternal, Reference: "https://openstax.org/books/elementary-algebra-2e/pages/2-1-solving-linear-equations"}, Published: true, Version: 1},
		{ID: "material-graphing-teacher", CourseID: course.ID, ConceptID: "graphing-equations", Title: "Teacher note: equations as stories", Body: "A line tells a story about how one quantity changes with another. Slope is the pace of change and the intercept is the starting point.", Source: domain.LearningSourceRef{ID: "material-graphing-teacher", Title: "Teacher note: equations as stories", Kind: domain.LearningSourceTeacher}, Published: true, Version: 1},
		{ID: "material-systems-teacher", CourseID: course.ID, ConceptID: "systems-equations", Title: "Teacher note: look for the shared point", Body: "Each equation describes a relationship. A solution to a system must satisfy both relationships at the same time.", Source: domain.LearningSourceRef{ID: "material-systems-teacher", Title: "Teacher note: look for the shared point", Kind: domain.LearningSourceTeacher}, Published: true, Version: 1},
		{ID: "material-quadratics-external", CourseID: course.ID, ConceptID: "quadratics", Title: "Khan Academy: introduction to quadratics", Body: "Quadratic relationships grow in both directions and can have zero, one, or two real roots.", Source: domain.LearningSourceRef{ID: "material-quadratics-external", Title: "Khan Academy: introduction to quadratics", Kind: domain.LearningSourceExternal, Reference: "https://www.khanacademy.org/math/algebra/x2f8bb11595b5c85:polynomials-and-their-factors"}, Published: true, Version: 1},
	}
	progress := []domain.ConceptProgress{
		{StudentID: "S-001", CourseID: course.ID, ConceptID: "linear-equations", Status: domain.ConceptProgressLearning, Confidence: 68, EvidenceCount: 4, NextAction: "Explain the solution check", LastActivityAt: now.Add(-2 * time.Hour)},
		{StudentID: "S-001", CourseID: course.ID, ConceptID: "graphing-equations", Status: domain.ConceptProgressNeedsSupport, Confidence: 38, EvidenceCount: 2, NextAction: "Connect slope to the line", LastActivityAt: now.Add(-26 * time.Hour)},
		{StudentID: "S-001", CourseID: course.ID, ConceptID: "systems-equations", Status: domain.ConceptProgressNotStarted, Confidence: 0, NextAction: "Start when prerequisites are secure"},
		{StudentID: "S-001", CourseID: course.ID, ConceptID: "quadratics", Status: domain.ConceptProgressNotStarted, Confidence: 0, NextAction: "Start when graphing is secure"},
		{StudentID: "S-002", CourseID: course.ID, ConceptID: "linear-equations", Status: domain.ConceptProgressSecure, Confidence: 86, EvidenceCount: 8, NextAction: "Move to graphing", LastActivityAt: now.Add(-4 * time.Hour)},
		{StudentID: "S-002", CourseID: course.ID, ConceptID: "graphing-equations", Status: domain.ConceptProgressNeedsSupport, Confidence: 31, EvidenceCount: 3, NextAction: "Review slope and intercept", LastActivityAt: now.Add(-20 * time.Hour)},
		{StudentID: "S-002", CourseID: course.ID, ConceptID: "systems-equations", Status: domain.ConceptProgressLearning, Confidence: 57, EvidenceCount: 3, NextAction: "Try substitution", LastActivityAt: now.Add(-3 * time.Hour)},
	}
	sessions := []domain.LearningSession{
		{ID: "session-s001-linear", StudentID: "S-001", CourseID: course.ID, ConceptID: "linear-equations", Mode: domain.TutorModeGuided, Status: domain.LearningSessionActive, StartedAt: now.Add(-48 * time.Minute), LastActiveAt: now.Add(-12 * time.Minute)},
		{ID: "session-s002-graph", StudentID: "S-002", CourseID: course.ID, ConceptID: "graphing-equations", Mode: domain.TutorModeSocratic, Status: domain.LearningSessionActive, StartedAt: now.Add(-2 * time.Hour), LastActiveAt: now.Add(-30 * time.Minute)},
	}
	turns := []domain.LearningTurn{
		{ID: "turn-s001-1", SessionID: "session-s001-linear", Role: domain.LearningTurnTutor, Kind: "explanation", Mode: domain.TutorModeGuided, Body: "An equation is a relationship waiting to be balanced. We can isolate the unknown while preserving what both sides have in common.", Sources: []domain.LearningSourceRef{materials[0].Source}, CreatedAt: now.Add(-48 * time.Minute)},
		{ID: "turn-s001-2", SessionID: "session-s001-linear", Role: domain.LearningTurnStudent, Kind: "student_message", Mode: domain.TutorModeGuided, Body: "I can isolate it, but I am not sure how to check the answer.", CreatedAt: now.Add(-12 * time.Minute)},
		{ID: "turn-s001-3", SessionID: "session-s001-linear", Role: domain.LearningTurnTutor, Kind: "hint", Mode: domain.TutorModeGuided, Body: "Try substituting your result into the original equation. The two sides should describe the same value.", Sources: []domain.LearningSourceRef{materials[0].Source}, CreatedAt: now.Add(-11 * time.Minute)},
	}
	evidence := []domain.LearningEvidence{
		{ID: "evidence-1", SessionID: "session-s001-linear", TurnID: "turn-s001-1", StudentID: "S-001", CourseID: course.ID, ConceptID: "linear-equations", Kind: "explanation", Note: "Introduced balance as the core relationship.", ObservedAt: now.Add(-48 * time.Minute)},
		{ID: "evidence-2", SessionID: "session-s001-linear", TurnID: "turn-s001-3", StudentID: "S-001", CourseID: course.ID, ConceptID: "linear-equations", Kind: "hint_request", Note: "Needs confidence with substitution checks.", ObservedAt: now.Add(-11 * time.Minute)},
		{ID: "evidence-3", SessionID: "session-s002-graph", TurnID: "turn-s002-1", StudentID: "S-002", CourseID: course.ID, ConceptID: "graphing-equations", Kind: "diagnostic_signal", Note: "Slope and intercept are still mixed together.", ObservedAt: now.Add(-30 * time.Minute)},
	}

	contentRepo := memory.NewLearningContentRepository([]domain.LearningCourse{course}, concepts, objectives, materials)
	sessionRepo := memory.NewLearningSessionRepository(sessions, turns)
	progressRepo := memory.NewConceptProgressRepository(progress)
	evidenceRepo := memory.NewLearningEvidenceRepository(evidence)
	checkRepo := memory.NewUnderstandingCheckRepository(nil)
	learningAgent := service.NewTutorAgent(memory.NewStubLearningAgentGateway())
	understandingChecker := service.NewUnderstandingCheckService(memory.NewStubFormativeCheckGateway())

	studentPortal := usecase.NewStudentPortal(submissionRepo, evaluationRepo)
	facultyPortal := usecase.NewFacultyPortal(submissionRepo, evaluationRepo)
	studentLearningPortal := usecase.NewStudentLearningPortal(contentRepo, sessionRepo, progressRepo, evidenceRepo, checkRepo, learningAgent, understandingChecker)
	teacherLearningPortal := usecase.NewTeacherLearningPortal(contentRepo, progressRepo, evidenceRepo)

	renderer, err := view.NewRenderer("src/templates/**/*.html")
	if err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}

	return App{
		Config: httpConfig,
		Routes: routes.Dependencies{
			Health:   controller.NewHealthController(),
			Student:  controller.NewStudentController(studentPortal, renderer),
			Faculty:  controller.NewFacultyController(facultyPortal, renderer),
			Learning: controller.NewLearningController(studentLearningPortal, teacherLearningPortal, renderer),
		},
	}
}
