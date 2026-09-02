package memory

import "github.com/srijan-raghavula/colearning-agent/internal/domain"

type SubmissionStore struct {
	items map[string]domain.Submission
}

func NewSubmissionStore() *SubmissionStore {
	return &SubmissionStore{items: map[string]domain.Submission{}}
}

func (s *SubmissionStore) Save(submission domain.Submission) {
	s.items[submission.ID] = submission
}

func (s *SubmissionStore) FindByID(id string) (domain.Submission, bool) {
	submission, ok := s.items[id]
	return submission, ok
}
