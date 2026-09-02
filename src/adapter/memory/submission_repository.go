package memory

import (
	"context"
	"sync"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type SubmissionRepository struct {
	mu    sync.RWMutex
	items map[string]domain.Submission
}

func NewSubmissionRepository(seed []domain.Submission) *SubmissionRepository {
	items := make(map[string]domain.Submission, len(seed))
	for _, submission := range seed {
		items[submission.ID] = submission
	}
	return &SubmissionRepository{items: items}
}

func (r *SubmissionRepository) Save(_ context.Context, submission domain.Submission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[submission.ID] = submission
	return nil
}

func (r *SubmissionRepository) FindByID(_ context.Context, id string) (domain.Submission, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	submission, ok := r.items[id]
	return submission, ok, nil
}

func (r *SubmissionRepository) ListByStudent(_ context.Context, studentID string) ([]domain.Submission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Submission, 0)
	for _, submission := range r.items {
		if submission.StudentID == studentID {
			result = append(result, submission)
		}
	}
	return result, nil
}

func (r *SubmissionRepository) ListAll(_ context.Context) ([]domain.Submission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.Submission, 0, len(r.items))
	for _, submission := range r.items {
		result = append(result, submission)
	}
	return result, nil
}
