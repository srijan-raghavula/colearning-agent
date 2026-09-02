package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// StubAIGateway is a deterministic, rule-based implementation of the
// AIGateway interface. It requires no external API key and produces
// consistent, inspectable responses that satisfy the same typed contract
// as a real LLM adapter.
//
// Design rationale: using a stub rather than mocking at the call site means
// the full StagedEvaluationPipeline — including concurrency, guardrails,
// and score aggregation — can be exercised in unit and integration tests
// without network I/O. A real OpenAI or Anthropic adapter would implement
// the identical AIGateway interface; swapping adapters requires changing
// only the bootstrap wiring.
type StubAIGateway struct{}

func NewStubAIGateway() *StubAIGateway {
	return &StubAIGateway{}
}

// Complete applies deterministic rules that mirror what a real LLM is
// expected to return for each evaluation dimension. The rules are
// intentionally simple so that test assertions can be written against
// exact expected values.
func (s *StubAIGateway) Complete(_ context.Context, req repository.PromptRequest) (repository.PromptResponse, error) {
	content := strings.ToLower(req.SubmissionContent)

	switch req.Dimension {
	case "logic":
		return s.evaluateLogic(content, req.MaxScore), nil
	case "quality":
		return s.evaluateQuality(content, req.MaxScore), nil
	case "gap":
		return s.analyzeGaps(content, req.TopicHints, req.MaxScore), nil
	default:
		return repository.PromptResponse{}, fmt.Errorf("unknown evaluation dimension: %q", req.Dimension)
	}
}

func (s *StubAIGateway) evaluateLogic(content string, maxScore float64) repository.PromptResponse {
	score := maxScore * 0.7 // baseline: meets requirements
	observations := []string{"Basic logical structure is correct."}

	if strings.Contains(content, "edge case") || strings.Contains(content, "boundary") {
		score = maxScore * 0.9
		observations = append(observations, "Edge-case handling is present — well done.")
	}
	if strings.Contains(content, "todo") || strings.Contains(content, "fixme") {
		score = maxScore * 0.4
		observations = append(observations, "Incomplete logic detected: TODO/FIXME markers remain.")
	}

	return repository.PromptResponse{Score: score, Observations: observations}
}

func (s *StubAIGateway) evaluateQuality(content string, maxScore float64) repository.PromptResponse {
	score := maxScore * 0.65
	observations := []string{"Code is readable but could be more modular."}

	if strings.Contains(content, "func ") && strings.Contains(content, "return") {
		score = maxScore * 0.85
		observations = append(observations, "Functions are well-scoped with clear return paths.")
	}
	if strings.Contains(content, "var ") && !strings.Contains(content, ":=") {
		score = maxScore * 0.5
		observations = append(observations, "Consider using short variable declarations for idiomatic Go.")
	}

	return repository.PromptResponse{Score: score, Observations: observations}
}

func (s *StubAIGateway) analyzeGaps(content string, hints []string, maxScore float64) repository.PromptResponse {
	weak := []string{}
	for _, topic := range hints {
		if !strings.Contains(content, strings.ToLower(topic)) {
			weak = append(weak, topic)
		}
	}

	coverage := 1.0 - float64(len(weak))/float64(max(len(hints), 1))
	score := maxScore * coverage
	obs := []string{fmt.Sprintf("%.0f%% of expected topics were addressed.", coverage*100)}
	if len(weak) > 0 {
		obs = append(obs, fmt.Sprintf("Missing coverage of: %s.", strings.Join(weak, ", ")))
	}

	return repository.PromptResponse{Score: score, Observations: obs, WeakTopics: weak}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
