package sqlite

import (
	"context"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

// ProgressRepo adapts SQLite's explicit SaveProgress method to the learning
// repository port.
type ProgressRepo struct{ *SQLiteRepo }

func (r ProgressRepo) Save(ctx context.Context, progress domain.ConceptProgress) error {
	return r.SQLiteRepo.SaveProgress(ctx, progress)
}

// EvidenceRepo adapts SQLite's explicit SaveEvidence method to the learning
// repository port.
type EvidenceRepo struct{ *SQLiteRepo }

func (r EvidenceRepo) Save(ctx context.Context, evidence domain.LearningEvidence) error {
	return r.SQLiteRepo.SaveEvidence(ctx, evidence)
}

// UnderstandingCheckRepo adapts SQLite's check-specific methods to the
// provider-neutral learning repository port.
type UnderstandingCheckRepo struct{ *SQLiteRepo }

func (r UnderstandingCheckRepo) Save(ctx context.Context, check domain.UnderstandingCheck) error {
	return r.SQLiteRepo.SaveCheck(ctx, check)
}

func (r UnderstandingCheckRepo) ListByCourse(ctx context.Context, courseID string) ([]domain.UnderstandingCheck, error) {
	return r.SQLiteRepo.ListCheckByCourse(ctx, courseID)
}
