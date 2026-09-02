package domain

import "time"

type SubmissionStatus string

const (
	SubmissionStatusSubmitted SubmissionStatus = "submitted"
	SubmissionStatusLate      SubmissionStatus = "late"
	SubmissionStatusMissing   SubmissionStatus = "missing"
)

type Submission struct {
	ID          string
	StudentID   string
	SubjectID   string
	TopicID     string
	Title       string
	Attachment  string
	DueAt       time.Time
	SubmittedAt time.Time
	Status      SubmissionStatus
}
