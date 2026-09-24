package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

var allowedLearningResponseKinds = map[string]bool{
	"explanation": true,
	"question":    true,
	"hint":        true,
	"practice":    true,
	"summary":     true,
	"escalation":  true,
}

// TutorAgent is the application service around the provider gateway. The
// service is intentionally small in the first slice: context construction and
// persistence remain in the use-case layer, while this service owns the
// provider-neutral tutor turn contract and response guardrails.
type TutorAgent struct {
	gateway repository.LearningAgentGateway
}

func NewTutorAgent(gateway repository.LearningAgentGateway) *TutorAgent {
	return &TutorAgent{gateway: gateway}
}

func (a *TutorAgent) Respond(ctx context.Context, req repository.LearningRequest) (repository.LearningResponse, error) {
	if a == nil || a.gateway == nil {
		return repository.LearningResponse{}, errors.New("learning agent gateway is not configured")
	}
	if !req.Mode.IsValid() {
		return repository.LearningResponse{}, fmt.Errorf("unsupported tutor mode %q", req.Mode)
	}
	if !req.Policy.AllowsMode(req.Mode) {
		return repository.LearningResponse{}, fmt.Errorf("tutor mode %q is not allowed by the course policy", req.Mode)
	}

	response, err := a.gateway.Respond(ctx, req)
	if err != nil {
		return repository.LearningResponse{}, fmt.Errorf("tutor agent request failed: %w", err)
	}
	return ValidateLearningResponse(req, response)
}

// ValidateLearningResponse keeps a provider response inside the application
// contract. It is intentionally conservative: a future provider adapter can
// be richer, but the UI and persistence layers receive the same stable shape.
func ValidateLearningResponse(req repository.LearningRequest, response repository.LearningResponse) (repository.LearningResponse, error) {
	response.Kind = strings.ToLower(strings.TrimSpace(response.Kind))
	response.Body = strings.TrimSpace(response.Body)
	response.NextAction = strings.TrimSpace(response.NextAction)

	if !allowedLearningResponseKinds[response.Kind] {
		return repository.LearningResponse{}, fmt.Errorf("unsupported tutor response kind %q", response.Kind)
	}
	if response.Body == "" {
		return repository.LearningResponse{}, errors.New("tutor response body is empty")
	}
	for _, phrase := range []string{"you are stupid", "this is terrible", "awful"} {
		if strings.Contains(strings.ToLower(response.Body), phrase) {
			response.Body = "Let’s revisit this idea with a clear, constructive explanation."
			response.Escalate = true
			break
		}
	}
	if response.NextAction == "" {
		response.NextAction = "Continue the conversation"
	}

	knownSources := make(map[string]domain.LearningSourceRef, len(req.Materials))
	for _, material := range req.Materials {
		if material.Published {
			knownSources[material.Source.ID] = material.Source
		}
	}

	filtered := make([]domain.LearningSourceRef, 0, len(response.Sources))
	for _, source := range response.Sources {
		known, ok := knownSources[source.ID]
		if !ok {
			continue
		}
		if known.Kind == domain.LearningSourceExternal && !req.Policy.AllowExternalSources {
			continue
		}
		filtered = append(filtered, known)
	}
	response.Sources = filtered

	if response.ConfidenceDelta < -10 {
		response.ConfidenceDelta = -10
	}
	if response.ConfidenceDelta > 10 {
		response.ConfidenceDelta = 10
	}
	return response, nil
}
