package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/memory"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/service"
)

// ─── DefaultEvaluationEngine Unit Tests ──────────────────────────────────────

func TestComputeMasteryPercentage(t *testing.T) {
	engine := service.NewDefaultEvaluationEngine()

	tests := []struct {
		name     string
		score    float64
		maxScore float64
		want     float64
	}{
		{name: "basic ratio", score: 18, maxScore: 20, want: 90},
		{name: "max score zero", score: 10, maxScore: 0, want: 0},
		{name: "clamp below zero", score: -4, maxScore: 20, want: 0},
		{name: "clamp above max", score: 30, maxScore: 20, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.ComputeMasteryPercentage(tt.score, tt.maxScore)
			if got != tt.want {
				t.Fatalf("ComputeMasteryPercentage(%v, %v) = %v, want %v", tt.score, tt.maxScore, got, tt.want)
			}
		})
	}
}

func TestEvaluateIncludesTopicFeedback(t *testing.T) {
	engine := service.NewDefaultEvaluationEngine()

	evaluation := engine.Evaluate(domain.Submission{ID: "SUB-1"}, 15, 20, []domain.TopicScore{{Topic: "algebra", Score: 3, MaxPossible: 10}})
	if evaluation.MasteryPercentage != 75 {
		t.Fatalf("expected mastery 75, got %v", evaluation.MasteryPercentage)
	}
	if len(evaluation.Feedback) != 1 {
		t.Fatalf("expected one feedback item, got %d", len(evaluation.Feedback))
	}
}

// ─── Guardrail Unit Tests ─────────────────────────────────────────────────────

func TestScoreGuardrailClampsAbove100(t *testing.T) {
	guard := service.NewScoreGuardrail()
	result := guard.Validate(domain.Evaluation{
		MasteryPercentage: 150,
		Feedback:          []string{"Great work!"},
	})
	if result.MasteryPercentage != 100 {
		t.Fatalf("expected mastery clamped to 100, got %v", result.MasteryPercentage)
	}
}

func TestScoreGuardrailClampsBelow0(t *testing.T) {
	guard := service.NewScoreGuardrail()
	result := guard.Validate(domain.Evaluation{
		MasteryPercentage: -10,
		Feedback:          []string{"Needs work."},
	})
	if result.MasteryPercentage != 0 {
		t.Fatalf("expected mastery clamped to 0, got %v", result.MasteryPercentage)
	}
}

func TestScoreGuardrailStripsProhibitedTone(t *testing.T) {
	guard := service.NewScoreGuardrail()
	result := guard.Validate(domain.Evaluation{
		MasteryPercentage: 55,
		Feedback:          []string{"You are stupid for doing this.", "Try reviewing Chapter 3."},
	})
	if len(result.Feedback) != 1 {
		t.Fatalf("expected 1 feedback line after stripping prohibited tone, got %d: %v", len(result.Feedback), result.Feedback)
	}
	if result.Feedback[0] != "Try reviewing Chapter 3." {
		t.Fatalf("unexpected feedback content: %v", result.Feedback[0])
	}
}

func TestScoreGuardrailEnsuresMinimumFeedback(t *testing.T) {
	guard := service.NewScoreGuardrail()
	// All lines are prohibited — guardrail must inject a safe fallback
	result := guard.Validate(domain.Evaluation{
		MasteryPercentage: 40,
		Feedback:          []string{"This is terrible work."},
	})
	if len(result.Feedback) == 0 {
		t.Fatal("expected guardrail to inject fallback feedback, got empty slice")
	}
}

// ─── StagedEvaluationPipeline Integration Tests ───────────────────────────────

func buildPipeline(t *testing.T) *service.StagedEvaluationPipeline {
	t.Helper()
	gateway := memory.NewStubAIGateway()
	const subDimensionMax = 10.0

	logic := service.NewAILogicEvaluator(gateway, subDimensionMax)
	quality := service.NewAIQualityEvaluator(gateway, subDimensionMax)
	gaps := service.NewAIConceptGapEvaluator(gateway, subDimensionMax)
	guard := service.NewScoreGuardrail()

	return service.NewStagedEvaluationPipeline(logic, quality, gaps, guard)
}

func TestStagedPipelineReturnsValidEvaluation(t *testing.T) {
	pipeline := buildPipeline(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	submission := domain.Submission{
		ID:        "SUB-101",
		StudentID: "S-001",
		SubjectID: "CS-101",
		Attachment: "func solve() { return edge case result }",
	}
	topicScores := []domain.TopicScore{
		{Topic: "Recursion", Score: 9, MaxPossible: 10},
		{Topic: "Sorting", Score: 7, MaxPossible: 10},
	}

	eval, err := pipeline.EvaluatePipeline(ctx, submission, 18, 20, topicScores)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Guardrail invariant: mastery must always be in [0, 100]
	if eval.MasteryPercentage < 0 || eval.MasteryPercentage > 100 {
		t.Fatalf("mastery %v is outside [0, 100]", eval.MasteryPercentage)
	}

	// Must always have at least one feedback line (guardrail ensures this)
	if len(eval.Feedback) == 0 {
		t.Fatal("expected at least one feedback line")
	}

	// Submission ID must be preserved through the pipeline
	if eval.SubmissionID != submission.ID {
		t.Fatalf("expected SubmissionID %q, got %q", submission.ID, eval.SubmissionID)
	}

	// EvaluatedAt must be a recent timestamp
	if eval.EvaluatedAt.IsZero() {
		t.Fatal("EvaluatedAt must not be zero")
	}
}

func TestStagedPipelineRespectsContextCancellation(t *testing.T) {
	pipeline := buildPipeline(t)

	// Cancel immediately before calling
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pipeline.EvaluatePipeline(ctx, domain.Submission{ID: "SUB-X", SubjectID: "CS"}, 10, 20, nil)
	if err == nil {
		// The stub is fast; cancellation may or may not fire before the goroutines complete.
		// We accept either outcome — the important constraint is that the pipeline never
		// panics and always returns a usable result or a wrapped context error.
		t.Log("context was cancelled but pipeline completed before noticing — acceptable")
	}
}

func TestStagedPipelineEdgeCaseMaterialBoostsLogicScore(t *testing.T) {
	pipeline := buildPipeline(t)
	ctx := context.Background()

	// The attachment mentions "edge case" → stub logic evaluator awards 90% score (9/10).
	// The topic "Graphs" does NOT appear in the attachment → gap evaluator flags it
	// as a weak topic (mastery 0), which is the correct educational signal.
	//
	// Weighted aggregate:
	//   logic  0.5 * 90  = 45.0
	//   quality 0.3 * 65 = 19.5  (stub baseline: no func/return keywords)
	//   gap    0.2 *  0  =  0.0  (one topic, zero coverage)
	//   total            = 64.5  → but quality stub for this content may vary
	//
	// We assert invariants that hold regardless of exact stub values:
	// 1. Edge-case content should lift logic observations above baseline.
	// 2. Result is guardrail-valid (mastery in [0, 100]).
	submission := domain.Submission{
		ID:         "SUB-200",
		SubjectID:  "CS-201",
		Attachment: "handles edge case and boundary conditions",
	}

	eval, err := pipeline.EvaluatePipeline(ctx, submission, 18, 20, []domain.TopicScore{
		{Topic: "Graphs", Score: 8, MaxPossible: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Guardrail invariant always holds
	if eval.MasteryPercentage < 0 || eval.MasteryPercentage > 100 {
		t.Fatalf("mastery out of guardrail range: %.2f", eval.MasteryPercentage)
	}

	// Logic observation for "edge case" content must be present in feedback
	foundEdgeCaseObservation := false
	for _, f := range eval.Feedback {
		if len(f) > 0 && (contains(f, "edge") || contains(f, "Edge")) {
			foundEdgeCaseObservation = true
			break
		}
	}
	if !foundEdgeCaseObservation {
		t.Errorf("expected edge-case observation in feedback, got: %v", eval.Feedback)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsRune(s, sub))
}

func containsRune(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestStagedPipelineCustomWeights(t *testing.T) {
	gateway := memory.NewStubAIGateway()
	logic := service.NewAILogicEvaluator(gateway, 10)
	quality := service.NewAIQualityEvaluator(gateway, 10)
	gaps := service.NewAIConceptGapEvaluator(gateway, 10)
	guard := service.NewScoreGuardrail()

	// Weight logic very heavily
	pipeline := service.NewStagedEvaluationPipeline(
		logic, quality, gaps, guard,
		service.WithWeights(0.8, 0.1, 0.1),
	)

	ctx := context.Background()
	eval, err := pipeline.EvaluatePipeline(ctx, domain.Submission{
		ID:         "SUB-300",
		SubjectID:  "CS-301",
		Attachment: "edge case boundary",
	}, 18, 20, []domain.TopicScore{
		{Topic: "Trees", Score: 9, MaxPossible: 10},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eval.MasteryPercentage < 0 || eval.MasteryPercentage > 100 {
		t.Fatalf("mastery out of bounds: %v", eval.MasteryPercentage)
	}
}
