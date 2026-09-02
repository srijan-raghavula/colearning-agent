package repository

import (
	"context"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type SubmissionWriter interface {
	Save(ctx context.Context, submission domain.Submission) error
}

type SubmissionReader interface {
	FindByID(ctx context.Context, id string) (domain.Submission, bool, error)
	ListByStudent(ctx context.Context, studentID string) ([]domain.Submission, error)
	ListAll(ctx context.Context) ([]domain.Submission, error)
}

type SubmissionRepository interface {
	SubmissionWriter
	SubmissionReader
}
