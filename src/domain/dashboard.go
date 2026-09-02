package domain

type StudentSubmissionRecord struct {
	Submission Submission
	Evaluation *Evaluation
}

type StudentDashboard struct {
	StudentID string
	Records   []StudentSubmissionRecord
}

type SubjectSummary struct {
	SubjectID        string
	TotalSubmissions int
	SubmittedCount   int
	LateCount        int
	MissingCount     int
	AverageMastery   float64
	LowMasteryTopics []string
}

type FacultyDashboard struct {
	Summaries          []SubjectSummary
	MissingSubmissions []Submission
}
