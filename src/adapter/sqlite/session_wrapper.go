package sqlite

import (
	"context"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type SessionRepo struct{ *SQLiteRepo }

func (r SessionRepo) Save(ctx context.Context, s domain.LearningSession) error {
	return r.SQLiteRepo.Save(ctx, s)
}
func (r SessionRepo) FindByID(ctx context.Context, id string) (domain.LearningSession, bool, error) {
	return r.SQLiteRepo.FindByID(ctx, id)
}
func (r SessionRepo) FindActive(ctx context.Context, sid, cid, concept string) (domain.LearningSession, bool, error) {
	return r.SQLiteRepo.FindActive(ctx, sid, cid, concept)
}
func (r SessionRepo) AppendTurn(ctx context.Context, t domain.LearningTurn) error {
	return r.SQLiteRepo.AppendTurn(ctx, t)
}
func (r SessionRepo) ListTurns(ctx context.Context, sid string) ([]domain.LearningTurn, error) {
	return r.SQLiteRepo.ListTurns(ctx, sid)
}
