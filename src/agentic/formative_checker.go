package agentic

import (
	"context"
	"fmt"

	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// FormativeChecker is a cooperating evaluator that runs only when a learner
// explicitly requests a check. Its response remains formative feedback.
type FormativeChecker struct {
	Gateway repository.FormativeCheckGateway
	Logger  ExecutionLogger
}

func (a FormativeChecker) Check(ctx context.Context, request repository.FormativeCheckRequest) (repository.FormativeCheckResponse, error) {
	if a.Gateway == nil {
		return repository.FormativeCheckResponse{}, fmt.Errorf("formative check gateway is not configured")
	}
	response, err := a.Gateway.Check(ctx, request)
	if err != nil {
		return repository.FormativeCheckResponse{}, err
	}
	if a.Logger != nil {
		_ = a.Logger.AgentLogForSession(ctx, request.SessionID, "FormativeChecker", "evaluate_student_explanation", map[string]any{"course_id": request.CourseID, "concept_id": request.ConceptID}, map[string]any{"summary": response.Summary, "next_action": response.NextAction, "confidence_delta": response.ConfidenceDelta}, []map[string]any{{"name": "read_learning_objective", "args": map[string]any{"concept_id": request.ConceptID}}, {"name": "read_recent_turns", "args": map[string]any{"count": len(request.RecentTurns)}}})
	}
	return response, nil
}
