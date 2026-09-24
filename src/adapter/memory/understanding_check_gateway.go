package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// StubFormativeCheckGateway makes the optional understanding check executable
// without introducing a final grade or coupling the learning loop to the
// legacy submission evaluator.
type StubFormativeCheckGateway struct{}

func NewStubFormativeCheckGateway() *StubFormativeCheckGateway {
	return &StubFormativeCheckGateway{}
}

func (g *StubFormativeCheckGateway) Check(_ context.Context, req repository.FormativeCheckRequest) (repository.FormativeCheckResponse, error) {
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return repository.FormativeCheckResponse{}, fmt.Errorf("an answer or explanation is required for an understanding check")
	}
	lower := strings.ToLower(answer)
	feedback := []string{
		"You have named a useful starting point. Keep the explanation connected to the idea, not only the procedure.",
	}
	evidence := []string{"The response names a relevant relationship or approach."}
	confidenceDelta := 2
	if strings.Contains(lower, "because") || strings.Contains(lower, "therefore") || strings.Contains(lower, "means") {
		feedback = append(feedback, "Your use of a connective makes the reasoning easier to follow.")
		evidence = append(evidence, "The response includes an explicit causal or explanatory link.")
		confidenceDelta = 4
	}
	if len(answer) < 30 {
		feedback = append(feedback, "Try adding one sentence that explains why the step works.")
		evidence = append(evidence, "The explanation is still brief and would benefit from more reasoning.")
		confidenceDelta--
	}
	return repository.FormativeCheckResponse{
		Summary:         "This is formative feedback on your current explanation, not a final grade.",
		Feedback:        feedback,
		Evidence:        evidence,
		NextAction:      "Revise the explanation, then ask the tutor for another check",
		ConfidenceDelta: confidenceDelta,
	}, nil
}
