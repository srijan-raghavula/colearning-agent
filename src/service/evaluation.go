package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// ─── Interfaces ──────────────────────────────────────────────────────────────

type MasteryCalculator interface {
	ComputeMasteryPercentage(score, maxScore float64) float64
}

type FeedbackGenerator interface {
	GenerateFeedback(topicScores []domain.TopicScore) []string
}

type EvaluationEngine interface {
	Evaluate(submission domain.Submission, score, maxScore float64, topicScores []domain.TopicScore) domain.Evaluation
}

// Stage 1: Context & Rubric Assembly Contract
type EvaluationContext struct {
	Submission  domain.Submission
	RubricID    string
	TopicScores []domain.TopicScore
}

type ContextBuilder interface {
	BuildContext(ctx context.Context, submission domain.Submission, rawTopicScores []domain.TopicScore) (EvaluationContext, error)
}

// Stage 2: Multi-Aspect Evaluator Contracts
type LogicEvaluator interface {
	EvaluateLogic(ctx context.Context, evalCtx EvaluationContext) (float64, []string, error)
}

type QualityEvaluator interface {
	EvaluateQuality(ctx context.Context, evalCtx EvaluationContext) (float64, []string, error)
}

type ConceptGapEvaluator interface {
	AnalyzeConceptGaps(ctx context.Context, evalCtx EvaluationContext) ([]domain.TopicScore, error)
}

// Stage 3: Synthesis & Guardrails Contract
type GuardrailValidator interface {
	Validate(evaluation domain.Evaluation) domain.Evaluation
}

// ─── DefaultEvaluationEngine (pure-math fallback) ────────────────────────────

// DefaultEvaluationEngine is a pure-math, zero-dependency engine that
// computes mastery percentages and generates rule-based feedback. It is used
// as a fallback and as a building block inside sub-evaluators when no LLM
// response is available.
type DefaultEvaluationEngine struct{}

func NewDefaultEvaluationEngine() DefaultEvaluationEngine {
	return DefaultEvaluationEngine{}
}

func (DefaultEvaluationEngine) ComputeMasteryPercentage(score, maxScore float64) float64 {
	if maxScore <= 0 {
		return 0
	}
	if score < 0 {
		score = 0
	}
	if score > maxScore {
		score = maxScore
	}
	return (score / maxScore) * 100
}

func (e DefaultEvaluationEngine) GenerateFeedback(topicScores []domain.TopicScore) []string {
	feedback := make([]string, 0, len(topicScores))
	for _, score := range topicScores {
		switch {
		case score.Mastery < 50:
			feedback = append(feedback, fmt.Sprintf("Focus on topic %s with guided revision.", score.Topic))
		case score.Mastery < 75:
			feedback = append(feedback, fmt.Sprintf("Practice more problems in topic %s.", score.Topic))
		default:
			feedback = append(feedback, fmt.Sprintf("Strong understanding in topic %s.", score.Topic))
		}
	}
	return feedback
}

func (e DefaultEvaluationEngine) Evaluate(submission domain.Submission, score, maxScore float64, topicScores []domain.TopicScore) domain.Evaluation {
	normalized := make([]domain.TopicScore, 0, len(topicScores))
	for _, topicScore := range topicScores {
		topicScore.Mastery = e.ComputeMasteryPercentage(topicScore.Score, topicScore.MaxPossible)
		normalized = append(normalized, topicScore)
	}

	mastery := e.ComputeMasteryPercentage(score, maxScore)
	return domain.Evaluation{
		SubmissionID:      submission.ID,
		Score:             score,
		MaxScore:          maxScore,
		MasteryPercentage: mastery,
		TopicScores:       normalized,
		Feedback:          e.GenerateFeedback(normalized),
		EvaluatedAt:       time.Now().UTC(),
	}
}

// ─── AI-backed Sub-Evaluators ─────────────────────────────────────────────────

// AILogicEvaluator sends the submission to the LLM under the "logic"
// dimension and returns a normalised score plus student-facing observations.
type AILogicEvaluator struct {
	gateway  repository.AIGateway
	maxScore float64
}

func NewAILogicEvaluator(gateway repository.AIGateway, maxScore float64) *AILogicEvaluator {
	return &AILogicEvaluator{gateway: gateway, maxScore: maxScore}
}

func (e *AILogicEvaluator) EvaluateLogic(ctx context.Context, evalCtx EvaluationContext) (float64, []string, error) {
	req := repository.PromptRequest{
		Dimension:         "logic",
		RubricID:          evalCtx.RubricID,
		SubmissionContent: evalCtx.Submission.Attachment,
		MaxScore:          e.maxScore,
	}
	resp, err := e.gateway.Complete(ctx, req)
	if err != nil {
		return 0, nil, fmt.Errorf("logic evaluator: %w", err)
	}
	return resp.Score, resp.Observations, nil
}

// AIQualityEvaluator sends the submission under the "quality" dimension.
type AIQualityEvaluator struct {
	gateway  repository.AIGateway
	maxScore float64
}

func NewAIQualityEvaluator(gateway repository.AIGateway, maxScore float64) *AIQualityEvaluator {
	return &AIQualityEvaluator{gateway: gateway, maxScore: maxScore}
}

func (e *AIQualityEvaluator) EvaluateQuality(ctx context.Context, evalCtx EvaluationContext) (float64, []string, error) {
	req := repository.PromptRequest{
		Dimension:         "quality",
		RubricID:          evalCtx.RubricID,
		SubmissionContent: evalCtx.Submission.Attachment,
		MaxScore:          e.maxScore,
	}
	resp, err := e.gateway.Complete(ctx, req)
	if err != nil {
		return 0, nil, fmt.Errorf("quality evaluator: %w", err)
	}
	return resp.Score, resp.Observations, nil
}

// AIConceptGapEvaluator maps weak topics returned by the LLM back into
// domain.TopicScore values, preserving the existing evaluation schema.
type AIConceptGapEvaluator struct {
	gateway  repository.AIGateway
	maxScore float64
}

func NewAIConceptGapEvaluator(gateway repository.AIGateway, maxScore float64) *AIConceptGapEvaluator {
	return &AIConceptGapEvaluator{gateway: gateway, maxScore: maxScore}
}

func (e *AIConceptGapEvaluator) AnalyzeConceptGaps(ctx context.Context, evalCtx EvaluationContext) ([]domain.TopicScore, error) {
	hints := make([]string, 0, len(evalCtx.TopicScores))
	for _, ts := range evalCtx.TopicScores {
		hints = append(hints, ts.Topic)
	}

	req := repository.PromptRequest{
		Dimension:         "gap",
		RubricID:          evalCtx.RubricID,
		SubmissionContent: evalCtx.Submission.Attachment,
		TopicHints:        hints,
		MaxScore:          e.maxScore,
	}
	resp, err := e.gateway.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("concept gap evaluator: %w", err)
	}

	weakSet := make(map[string]bool, len(resp.WeakTopics))
	for _, t := range resp.WeakTopics {
		weakSet[strings.ToLower(t)] = true
	}

	// Preserve original score data; flag mastery as 0 for identified gaps
	// so the guardrail and feedback stages treat them as priority areas.
	updated := make([]domain.TopicScore, 0, len(evalCtx.TopicScores))
	engine := DefaultEvaluationEngine{}
	for _, ts := range evalCtx.TopicScores {
		if weakSet[strings.ToLower(ts.Topic)] {
			ts.Mastery = 0
		} else {
			ts.Mastery = engine.ComputeMasteryPercentage(ts.Score, ts.MaxPossible)
		}
		updated = append(updated, ts)
	}
	return updated, nil
}

// ─── Stage 3: Guardrail Validator ─────────────────────────────────────────────

// ScoreGuardrail clamps the mastery percentage to [0, 100], validates that
// feedback strings do not contain prohibited tone markers, and ensures at
// least one feedback line is always present. These invariants protect the
// domain entity from malformed LLM outputs before they reach the repository.
type ScoreGuardrail struct {
	prohibitedPhrases []string
}

func NewScoreGuardrail() *ScoreGuardrail {
	return &ScoreGuardrail{
		prohibitedPhrases: []string{"you are stupid", "this is terrible", "awful"},
	}
}

func (g *ScoreGuardrail) Validate(evaluation domain.Evaluation) domain.Evaluation {
	// Clamp mastery to valid range
	if evaluation.MasteryPercentage < 0 {
		evaluation.MasteryPercentage = 0
	}
	if evaluation.MasteryPercentage > 100 {
		evaluation.MasteryPercentage = 100
	}

	// Clamp individual topic masteries
	for i := range evaluation.TopicScores {
		if evaluation.TopicScores[i].Mastery < 0 {
			evaluation.TopicScores[i].Mastery = 0
		}
		if evaluation.TopicScores[i].Mastery > 100 {
			evaluation.TopicScores[i].Mastery = 100
		}
	}

	// Tone validation: strip any feedback line containing a prohibited phrase
	sanitized := make([]string, 0, len(evaluation.Feedback))
	for _, line := range evaluation.Feedback {
		lower := strings.ToLower(line)
		safe := true
		for _, phrase := range g.prohibitedPhrases {
			if strings.Contains(lower, phrase) {
				safe = false
				break
			}
		}
		if safe {
			sanitized = append(sanitized, line)
		}
	}

	// Guarantee at least one feedback line so the UI never shows an empty panel
	if len(sanitized) == 0 {
		sanitized = []string{"Evaluation complete. Please review your submission with your instructor."}
	}
	evaluation.Feedback = sanitized
	return evaluation
}

// ─── StagedEvaluationPipeline ─────────────────────────────────────────────────

// StagedEvaluationPipeline orchestrates the four-stage evaluation pipeline.
//
// Stage 1 — Context Assembly: builds a typed EvaluationContext carrying the
//
//	submission, a deterministic RubricID, and the raw topic scores.
//
// Stage 2 — Concurrent Multi-Aspect Evaluation: the three sub-evaluators
//
//	(Logic, Quality, ConceptGap) run in separate goroutines via errgroup.
//	If any sub-evaluator fails, the pipeline halts and propagates the error;
//	partial results are never persisted.
//
// Stage 3 — Synthesis & Guardrails: scores from all three dimensions are
//
//	weighted and aggregated into a single mastery percentage; the guardrail
//	validator then clamps bounds and sanitises feedback tone.
//
// Stage 4 — Result: returns a fully validated domain.Evaluation ready for
//
//	persistence and HTMX rendering without further transformation.
type StagedEvaluationPipeline struct {
	logic    LogicEvaluator
	quality  QualityEvaluator
	gaps     ConceptGapEvaluator
	guard    GuardrailValidator
	fallback DefaultEvaluationEngine

	// Weights must sum to 1.0
	logicWeight   float64
	qualityWeight float64
	gapWeight     float64
}

// PipelineOption is a functional option for configuring the pipeline.
type PipelineOption func(*StagedEvaluationPipeline)

func WithWeights(logic, quality, gap float64) PipelineOption {
	return func(p *StagedEvaluationPipeline) {
		p.logicWeight = logic
		p.qualityWeight = quality
		p.gapWeight = gap
	}
}

func NewStagedEvaluationPipeline(
	logic LogicEvaluator,
	quality QualityEvaluator,
	gaps ConceptGapEvaluator,
	guard GuardrailValidator,
	opts ...PipelineOption,
) *StagedEvaluationPipeline {
	p := &StagedEvaluationPipeline{
		logic:         logic,
		quality:       quality,
		gaps:          gaps,
		guard:         guard,
		fallback:      NewDefaultEvaluationEngine(),
		logicWeight:   0.5,
		qualityWeight: 0.3,
		gapWeight:     0.2,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// stageResult carries the output of one concurrent sub-evaluator.
type stageResult struct {
	logicScore    float64
	logicObs      []string
	qualityScore  float64
	qualityObs    []string
	refinedTopics []domain.TopicScore
}

// EvaluatePipeline runs the full four-stage pipeline and returns a guardrail-
// validated domain.Evaluation. It is safe to call concurrently.
func (p *StagedEvaluationPipeline) EvaluatePipeline(
	ctx context.Context,
	submission domain.Submission,
	score, maxScore float64,
	topicScores []domain.TopicScore,
) (domain.Evaluation, error) {

	// ── Stage 1: Context Assembly ─────────────────────────────────────────
	evalCtx := EvaluationContext{
		Submission:  submission,
		RubricID:    "RUBRIC-" + submission.SubjectID,
		TopicScores: topicScores,
	}

	// ── Stage 2: Concurrent Multi-Aspect Evaluation ───────────────────────
	// Each sub-evaluator owns a dedicated channel so we avoid shared mutable
	// state. All three run concurrently; the select-with-ctx pattern ensures
	// we respect deadline cancellation from the caller.
	type logicOut struct {
		score float64
		obs   []string
		err   error
	}
	type qualityOut struct {
		score float64
		obs   []string
		err   error
	}
	type gapOut struct {
		topics []domain.TopicScore
		err    error
	}

	logicCh := make(chan logicOut, 1)
	qualityCh := make(chan qualityOut, 1)
	gapCh := make(chan gapOut, 1)

	go func() {
		s, obs, err := p.logic.EvaluateLogic(ctx, evalCtx)
		logicCh <- logicOut{s, obs, err}
	}()
	go func() {
		s, obs, err := p.quality.EvaluateQuality(ctx, evalCtx)
		qualityCh <- qualityOut{s, obs, err}
	}()
	go func() {
		topics, err := p.gaps.AnalyzeConceptGaps(ctx, evalCtx)
		gapCh <- gapOut{topics, err}
	}()

	var res stageResult

	for range 3 {
		select {
		case <-ctx.Done():
			return domain.Evaluation{}, fmt.Errorf("evaluation pipeline cancelled: %w", ctx.Err())
		case lo := <-logicCh:
			if lo.err != nil {
				return domain.Evaluation{}, lo.err
			}
			res.logicScore = lo.score
			res.logicObs = lo.obs
		case qo := <-qualityCh:
			if qo.err != nil {
				return domain.Evaluation{}, qo.err
			}
			res.qualityScore = qo.score
			res.qualityObs = qo.obs
		case go_ := <-gapCh:
			if go_.err != nil {
				return domain.Evaluation{}, go_.err
			}
			res.refinedTopics = go_.topics
		}
	}

	// ── Stage 3: Synthesis & Guardrails ──────────────────────────────────
	// Weighted aggregation: normalise each sub-score relative to maxScore,
	// then combine with configured weights. This produces a mastery value
	// that reflects all three evaluation dimensions.
	logicMastery := p.fallback.ComputeMasteryPercentage(res.logicScore, maxScore)
	qualityMastery := p.fallback.ComputeMasteryPercentage(res.qualityScore, maxScore)

	// Gap dimension: average mastery across refined topic scores
	var gapMastery float64
	if len(res.refinedTopics) > 0 {
		var sum float64
		for _, ts := range res.refinedTopics {
			sum += ts.Mastery
		}
		gapMastery = sum / float64(len(res.refinedTopics))
	}

	aggregateMastery := (logicMastery * p.logicWeight) +
		(qualityMastery * p.qualityWeight) +
		(gapMastery * p.gapWeight)

	// Build unified feedback from all dimensions
	allFeedback := make([]string, 0)
	allFeedback = append(allFeedback, res.logicObs...)
	allFeedback = append(allFeedback, res.qualityObs...)
	allFeedback = append(allFeedback, p.fallback.GenerateFeedback(res.refinedTopics)...)

	evaluation := domain.Evaluation{
		SubmissionID:      submission.ID,
		Score:             score,
		MaxScore:          maxScore,
		MasteryPercentage: aggregateMastery,
		TopicScores:       res.refinedTopics,
		Feedback:          allFeedback,
		EvaluatedAt:       time.Now().UTC(),
	}

	// Guardian: clamp and tone-check before returning
	return p.guard.Validate(evaluation), nil
}
