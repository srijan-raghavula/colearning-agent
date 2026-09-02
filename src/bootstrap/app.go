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

	// ── AI Evaluation Pipeline Assembly ──────────────────────────────────────
	// Swap memory.NewStubAIGateway() for an OpenAI/Anthropic adapter here;
	// no other code in the system needs to change — that's the value of the
	// AIGateway port.
	aiGateway := memory.NewStubAIGateway()
	const subDimensionMax = 10.0

	evalPipeline := service.NewStagedEvaluationPipeline(
		service.NewAILogicEvaluator(aiGateway, subDimensionMax),
		service.NewAIQualityEvaluator(aiGateway, subDimensionMax),
		service.NewAIConceptGapEvaluator(aiGateway, subDimensionMax),
		service.NewScoreGuardrail(),
	)
	_ = evalPipeline // pipeline available for injection into evaluation use-case

	// Seed evaluations use the pure-math engine for bootstrap determinism
	evaluationEngine := service.NewDefaultEvaluationEngine()
	evaluations := []domain.Evaluation{
		evaluationEngine.Evaluate(submissions[0], 18, 20, []domain.TopicScore{{Topic: "algebra", Score: 8, MaxPossible: 10}, {Topic: "functions", Score: 10, MaxPossible: 10}}),
		evaluationEngine.Evaluate(submissions[1], 12, 20, []domain.TopicScore{{Topic: "forces", Score: 6, MaxPossible: 10}, {Topic: "vectors", Score: 6, MaxPossible: 10}}),
		evaluationEngine.Evaluate(submissions[3], 16, 20, []domain.TopicScore{{Topic: "energy", Score: 8, MaxPossible: 10}, {Topic: "conservation", Score: 8, MaxPossible: 10}}),
	}

	submissionRepo := memory.NewSubmissionRepository(submissions)
	evaluationRepo := memory.NewEvaluationRepository(evaluations)

	renderer, err := view.NewRenderer("src/templates/**/*.html")
	if err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}

	studentPortal := usecase.NewStudentPortal(submissionRepo, evaluationRepo)
	facultyPortal := usecase.NewFacultyPortal(submissionRepo, evaluationRepo)

	return App{
		Config: httpConfig,
		Routes: routes.Dependencies{
			Health:  controller.NewHealthController(),
			Student: controller.NewStudentController(studentPortal, renderer),
			Faculty: controller.NewFacultyController(facultyPortal, renderer),
		},
	}
}
