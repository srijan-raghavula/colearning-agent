package repository

import "context"

// PromptRequest is the structured contract sent to any LLM provider.
// Using a typed struct instead of raw strings forces callers to be explicit
// about which evaluation dimension they are requesting and what the scoring
// bounds are, which makes guardrail validation feasible.
type PromptRequest struct {
	// Dimension names the evaluation aspect: "logic", "quality", or "gap".
	Dimension string

	// RubricID identifies the assignment rubric for context retrieval.
	RubricID string

	// SubmissionContent is the student's actual answer or code.
	SubmissionContent string

	// TopicHints lists the topics the assignment covers (for gap analysis).
	TopicHints []string

	// MaxScore is the ceiling for this dimension's raw score output.
	MaxScore float64
}

// PromptResponse carries the structured result from an LLM provider.
// Returning a typed struct (rather than a raw string) lets guardrails
// validate bounds and tone before the score enters the evaluation entity.
type PromptResponse struct {
	// Score is the raw numeric score for this dimension.
	Score float64

	// Observations are concise, student-facing feedback lines.
	Observations []string

	// WeakTopics lists topic identifiers the model identified as gaps.
	WeakTopics []string
}

// AIGateway is the port that decouples the evaluation pipeline from any
// specific LLM SDK. Concrete adapters (OpenAI, Anthropic, stub) implement
// this interface; the pipeline never imports an SDK directly.
type AIGateway interface {
	// Complete sends a structured prompt to the underlying provider and
	// returns a parsed, validated response. The implementation is
	// responsible for marshalling/unmarshalling and retrying transient
	// network errors.
	Complete(ctx context.Context, req PromptRequest) (PromptResponse, error)
}
