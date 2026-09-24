package service

import (
	"context"
	"testing"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

func TestTutorAgentFiltersExternalSourcesWhenDisabled(t *testing.T) {
	material := domain.LearningMaterial{
		ID:        "external-1",
		Published: true,
		Source:    domain.LearningSourceRef{ID: "external-1", Title: "External", Kind: domain.LearningSourceExternal},
	}
	agent := NewTutorAgent(&fixedLearningGateway{response: repository.LearningResponse{
		Kind:    "explanation",
		Body:    "A grounded explanation.",
		Sources: []domain.LearningSourceRef{material.Source},
	}})
	response, err := agent.Respond(context.Background(), repository.LearningRequest{
		Mode:      domain.TutorModeGuided,
		Materials: []domain.LearningMaterial{material},
		Policy:    domain.TutorPolicy{DefaultMode: domain.TutorModeGuided, AllowExternalSources: false},
	})
	if err != nil {
		t.Fatalf("respond: %v", err)
	}
	if len(response.Sources) != 0 {
		t.Fatalf("expected external source to be filtered, got %#v", response.Sources)
	}
}

type fixedLearningGateway struct {
	response repository.LearningResponse
}

func (g *fixedLearningGateway) Respond(_ context.Context, _ repository.LearningRequest) (repository.LearningResponse, error) {
	return g.response, nil
}
