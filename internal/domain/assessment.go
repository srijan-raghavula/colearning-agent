package domain

type SubmissionStatus string

const (
	SubmissionStatusSubmitted SubmissionStatus = "submitted"
	SubmissionStatusLate      SubmissionStatus = "late"
	SubmissionStatusMissing   SubmissionStatus = "missing"
)

type Submission struct {
	ID         string
	StudentID  string
	SubjectID  string
	TopicID    string
	Status     SubmissionStatus
	Attachment string
}

type TopicScore struct {
	Topic       string
	Score       float64
	MaxPossible float64
}

type Evaluation struct {
	SubmissionID string
	TopicScores  []TopicScore
	Feedback     []string
}
