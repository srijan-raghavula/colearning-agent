package repository

import (
	"context"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type EvaluationWriter interface {
	Save(ctx context.Context, evaluation domain.Evaluation) error
}

type EvaluationReader interface {
	FindBySubmissionID(ctx context.Context, submissionID string) (domain.Evaluation, bool, error)
	ListByStudent(ctx context.Context, studentID string, submissions []domain.Submission) ([]domain.Evaluation, error)
	ListAll(ctx context.Context) ([]domain.Evaluation, error)
}

type EvaluationRepository interface {
	EvaluationWriter
	EvaluationReader
}
