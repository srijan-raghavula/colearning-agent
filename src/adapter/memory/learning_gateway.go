package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// StubLearningAgentGateway is a deterministic tutor adapter for the first
// vertical slice. It gives the UI and API a real provider-neutral contract
// without requiring a network call or an API key.
type StubLearningAgentGateway struct{}

func NewStubLearningAgentGateway() *StubLearningAgentGateway {
	return &StubLearningAgentGateway{}
}

func (g *StubLearningAgentGateway) Respond(_ context.Context, req repository.LearningRequest) (repository.LearningResponse, error) {
	if strings.TrimSpace(req.Concept.Title) == "" {
		return repository.LearningResponse{}, fmt.Errorf("learning concept is required")
	}

	mode := req.Mode
	if mode == "" {
		mode = domain.TutorModeGuided
	}

	sources := allowedSources(req)
	question := strings.TrimSpace(req.Message)
	var kind, body, nextAction string
	var confidenceDelta int
	var evidenceKind = "conversation"

	switch mode {
	case domain.TutorModeSocratic:
		kind = "question"
		body = "Let’s make the idea visible. Before we continue, can you explain in your own words what this concept is trying to help you predict or solve?"
		nextAction = "Answer in your own words"
		confidenceDelta = 3
		evidenceKind = "student_explanation"
	case domain.TutorModeDirect:
		kind = "explanation"
		body = fmt.Sprintf("Think of %s as a set of connected ideas: identify the known pieces, make the relationship explicit, and check whether the result makes sense. The short version is to start with the structure, not the final answer.", req.Concept.Title)
		nextAction = "Try a quick practice prompt"
		confidenceDelta = 2
	case domain.TutorModePractice:
		kind = "practice"
		body = "Try this in one or two sentences: which part of the explanation would still hold if the numbers changed? Explain your reasoning before checking the source."
		nextAction = "Write your reasoning"
		confidenceDelta = 1
		evidenceKind = "practice_attempt"
	case domain.TutorModeHint:
		kind = "hint"
		body = "Hint: name the quantity or relationship you already know, then identify the one unknown you need to isolate. What would you try first?"
		nextAction = "Take the first step"
		confidenceDelta = 1
		evidenceKind = "hint_request"
	case domain.TutorModeDiagnostic:
		kind = "diagnostic"
		body = "Let’s diagnose the starting point. Which description feels closest to your current understanding?"
		nextAction = "Choose a confidence level"
		confidenceDelta = 1
		evidenceKind = "diagnostic_signal"
	default:
		kind = "explanation"
		body = fmt.Sprintf("Here is a useful way into %s: connect the definition to a small example, then explain why the example works. I’ll ask one short question after each step so we can adapt the explanation to you.", req.Concept.Title)
		nextAction = "Continue with an example"
		confidenceDelta = 2
	}

	if question != "" && mode != domain.TutorModeDiagnostic {
		body = fmt.Sprintf("I’m following your question about %s. %s", req.Concept.Title, body)
	}

	return repository.LearningResponse{
		Kind:            kind,
		Body:            body,
		Sources:         sources,
		Evidence:        []repository.LearningEvidenceDraft{{Kind: evidenceKind, Note: body, ConfidenceDelta: confidenceDelta}},
		NextAction:      nextAction,
		ConfidenceDelta: confidenceDelta,
	}, nil
}

func allowedSources(req repository.LearningRequest) []domain.LearningSourceRef {
	refs := make([]domain.LearningSourceRef, 0, len(req.Materials))
	for _, material := range req.Materials {
		if !material.Published {
			continue
		}
		if material.Source.Kind == domain.LearningSourceExternal && !req.Policy.AllowExternalSources {
			continue
		}
		refs = append(refs, material.Source)
		if len(refs) == 2 {
			break
		}
	}
	return refs
}
