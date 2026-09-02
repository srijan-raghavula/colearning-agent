package usecase

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

type SubmitRequest struct {
	ID         string
	StudentID  string
	SubjectID  string
	TopicID    string
	Title      string
	Attachment string
	DueAt      time.Time
}

type StudentPortal struct {
	submissions repository.SubmissionRepository
	evaluations repository.EvaluationReader
}

func NewStudentPortal(submissions repository.SubmissionRepository, evaluations repository.EvaluationReader) StudentPortal {
	return StudentPortal{submissions: submissions, evaluations: evaluations}
}

func (p StudentPortal) Submit(ctx context.Context, req SubmitRequest) (domain.Submission, error) {
	if req.ID == "" || req.StudentID == "" || req.SubjectID == "" || req.TopicID == "" || req.Title == "" {
		return domain.Submission{}, errors.New("missing required submission fields")
	}
	if req.DueAt.IsZero() {
		return domain.Submission{}, errors.New("due date is required")
	}

	now := time.Now().UTC()
	status := domain.SubmissionStatusSubmitted
	if now.After(req.DueAt) {
		status = domain.SubmissionStatusLate
	}

	submission := domain.Submission{
		ID:          req.ID,
		StudentID:   req.StudentID,
		SubjectID:   req.SubjectID,
		TopicID:     req.TopicID,
		Title:       req.Title,
		Attachment:  req.Attachment,
		DueAt:       req.DueAt.UTC(),
		SubmittedAt: now,
		Status:      status,
	}

	if err := p.submissions.Save(ctx, submission); err != nil {
		return domain.Submission{}, err
	}
	return submission, nil
}

func (p StudentPortal) Dashboard(ctx context.Context, studentID string) (domain.StudentDashboard, error) {
	submissions, err := p.submissions.ListByStudent(ctx, studentID)
	if err != nil {
		return domain.StudentDashboard{}, err
	}
	evaluations, err := p.evaluations.ListByStudent(ctx, studentID, submissions)
	if err != nil {
		return domain.StudentDashboard{}, err
	}

	evaluationBySubmission := map[string]domain.Evaluation{}
	for _, evaluation := range evaluations {
		evaluationBySubmission[evaluation.SubmissionID] = evaluation
	}

	records := make([]domain.StudentSubmissionRecord, 0, len(submissions))
	for _, submission := range submissions {
		record := domain.StudentSubmissionRecord{Submission: submission}
		if evaluation, ok := evaluationBySubmission[submission.ID]; ok {
			evaluationCopy := evaluation
			record.Evaluation = &evaluationCopy
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Submission.DueAt.Before(records[j].Submission.DueAt)
	})

	return domain.StudentDashboard{StudentID: studentID, Records: records}, nil
}
