package memory

import (
	"context"
	"sync"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type EvaluationRepository struct {
	mu    sync.RWMutex
	items map[string]domain.Evaluation
}

func NewEvaluationRepository(seed []domain.Evaluation) *EvaluationRepository {
	items := make(map[string]domain.Evaluation, len(seed))
	for _, evaluation := range seed {
		items[evaluation.SubmissionID] = evaluation
	}
	return &EvaluationRepository{items: items}
}

func (r *EvaluationRepository) Save(_ context.Context, evaluation domain.Evaluation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[evaluation.SubmissionID] = evaluation
	return nil
}

func (r *EvaluationRepository) FindBySubmissionID(_ context.Context, submissionID string) (domain.Evaluation, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	evaluation, ok := r.items[submissionID]
	return evaluation, ok, nil
}

func (r *EvaluationRepository) ListByStudent(_ context.Context, studentID string, submissions []domain.Submission) ([]domain.Evaluation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	submissionIDs := map[string]bool{}
	for _, submission := range submissions {
		if submission.StudentID == studentID {
			submissionIDs[submission.ID] = true
		}
	}

	result := make([]domain.Evaluation, 0)
	for submissionID := range submissionIDs {
		evaluation, ok := r.items[submissionID]
		if ok {
			result = append(result, evaluation)
		}
	}
	return result, nil
}

func (r *EvaluationRepository) ListAll(_ context.Context) ([]domain.Evaluation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Evaluation, 0, len(r.items))
	for _, evaluation := range r.items {
		result = append(result, evaluation)
	}
	return result, nil
}
